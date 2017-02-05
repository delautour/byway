package core

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

// EndpointConfig  config of an endpoint
type EndpointConfig struct {
	Host    string            `json:"host"`
	Scheme  string            `json:"scheme"`
	Rewrite string            `json:"rewrite"`
	Headers map[string]string `json:"headers"`
}

// Config - Raw byway configuration
type Config struct {
	Rewrites   []RewriteConfigString                            `json:"rewrites" yaml:"rewrites"`
	Mapping    map[ServiceName]map[VersionString]EndpointConfig `json:"services" yaml:"services"`
	Topologies map[TopologyKey]map[ServiceName]VersionString    `json:"topologies" yaml:"topologies"`
}

// Headers - a list of headers to set
type Headers map[string]string

// StringRewrite - a path rewrite func
type stringRewrite func(string) string

type binding struct {
	host          string
	scheme        string
	pathRewriteFn stringRewrite
	headers       Headers
}

// TopologyKey - a key represenenting a specific topology
type TopologyKey string

// VersionString - A version of a service 1.0.0
type VersionString string

// ServiceName - A name of a service
type ServiceName string

type serviceMappingTable map[ServiceName]map[VersionString]binding
type topologyTable map[TopologyKey]map[ServiceName]VersionString

type config struct {
	rewrites   []stringRewrite
	mapping    serviceMappingTable
	topologies topologyTable
}

// NewConfig creates a new config object
func NewConfig() *Config {
	return &Config{
		Mapping:    make(map[ServiceName]map[VersionString]EndpointConfig),
		Topologies: make(map[TopologyKey]map[ServiceName]VersionString),
		Rewrites:   make([]RewriteConfigString, 0)}
}

// IdentityRewrite a rewrite rule which is the identity
func IdentityRewrite(input string) string {
	return input
}

// RewriteConfigString a string in the format of:  <find regex>;<replace pattern>
// Eg:   ^bob/(?.*)$;foo/$1
// bob/bazzer -> foo/bazzer
type RewriteConfigString string

func newRegexReplaceRewriteFromRewriteConfigString(rewrite RewriteConfigString) stringRewrite {
	str := string(rewrite)
	p := strings.Split(str, ";")

	if len(p) != 2 {
		log.Fatalf("byway: NewRegexReplaceRewriteFromRewriteConfigString: invalid input: %s", str)
	}

	return newRegexReplaceRewrite(p[0], p[1])
}

func newRegexReplaceRewrite(pattern string, replace string) stringRewrite {
	re := regexp.MustCompile(pattern)

	fn := func(input string) string {
		result := re.ReplaceAllString(input, replace)
		if result != input {
			log.Printf("byway: Rewrite %s -> %s", input, result)
		}
		return result
	}
	return fn
}

func rewriteURL(config *config, input *url.URL) *url.URL {
	matched := make(map[string]bool)
	accumulator := input.String()
	for {
		rewriteResult := accumulator
		for _, rewrite := range config.rewrites {
			rewriteResult = rewrite(accumulator)
			if rewriteResult != accumulator {
				break
			}
		}
		if rewriteResult == accumulator {
			result, err := url.Parse(rewriteResult)
			if err != nil {
				log.Fatal(err)
			}
			return result
		}
		if matched[rewriteResult] {
			log.Fatal("Recursive rewrite detected")

		}
		matched[rewriteResult] = true

		accumulator = rewriteResult
	}
}
func mapEndpointConfig(endpointConfig EndpointConfig) binding {
	return binding{
		host:          endpointConfig.Host,
		scheme:        endpointConfig.Scheme,
		headers:       endpointConfig.Headers,
		pathRewriteFn: IdentityRewrite}
}

func mapConfig(rawConfig *Config) *config {
	newConfig := config{
		mapping:    make(map[ServiceName]map[VersionString]binding),
		topologies: rawConfig.Topologies,
	}

	for _, r := range rawConfig.Rewrites {
		rewrite := newRegexReplaceRewriteFromRewriteConfigString(r)
		newConfig.rewrites = append(newConfig.rewrites, rewrite)
	}

	for k, v := range rawConfig.Mapping {

		version := make(map[VersionString]binding)
		newConfig.mapping[ServiceName(k)] = version
		for k, v := range v {
			version[VersionString(k)] = mapEndpointConfig(v)
		}
	}

	if newConfig.topologies == nil {
		newConfig.topologies = make(map[TopologyKey]map[ServiceName]VersionString)
	}

	return &newConfig
}

func versionify(versionStr string) *version.Version {
	formatted := strings.Replace(versionStr, "-", ".", 3)
	v, err := version.NewVersion(formatted)
	if err != nil {
		return nil
	}
	return v
}

func extractRoutingParameters(req *http.Request) (TopologyKey, *version.Version, *version.Version, ServiceName) {
	log.Print("byway: -- extractRoutingParameters --")
	log.Printf("byway: URL: %s ", req.URL)
	var minVersion *version.Version
	var maxVersion *version.Version
	var serviceName string

	hostComponents := strings.Split(req.URL.Host, ".")
	log.Printf("byway: host components:  %s ", hostComponents)

	i := 0

	topologyKey := TopologyKey(req.Header.Get("x-byway-topology"))
	if strings.HasPrefix(hostComponents[i], "t-") {

		topologyKey = TopologyKey(hostComponents[i])
		topologyKey = topologyKey[2:len(topologyKey)]
		log.Printf("byway: Identified topology from host: %s", topologyKey)
		i++
	}

	minVersion = versionify(req.Header.Get("x-byway-min"))
	if minVersion == nil {
		minVersion = versionify(hostComponents[i])
		if minVersion != nil {
			log.Printf("byway: Identified min version from host: %s", minVersion)
			i++
		} else {
			log.Printf("byway: Could not identify min version")
		}
	} else {
		log.Printf("byway: Found min version from header: %s", minVersion)
	}

	maxVersion = versionify(req.Header.Get("x-byway-max"))
	if maxVersion == nil {
		maxVersion = versionify(hostComponents[i])
		if maxVersion != nil {
			log.Printf("byway: Identified max version from host: %s", maxVersion)
			i++
		} else {
			log.Printf("byway: Could not identify max version")
		}
	} else {
		log.Printf("byway: Found max version from header: %s", maxVersion)
	}

	serviceName = req.Header.Get("x-byway-service")
	if serviceName == "" {
		serviceName = hostComponents[i]
		log.Printf("byway: Identified service from host: %s", serviceName)
	} else {
		log.Printf("byway: Identified service from header: %s", serviceName)
	}

	return topologyKey, minVersion, maxVersion, ServiceName(serviceName)
}

func bulidContraint(minVersion *version.Version, maxVersion *version.Version) version.Constraints {

	if minVersion != nil && maxVersion != nil {
		constraint, _ := version.NewConstraint(fmt.Sprintf(">= %s, <= %s", minVersion, maxVersion))
		return constraint
	} else if minVersion != nil {
		constraint, _ := version.NewConstraint(fmt.Sprintf(">= %s", minVersion))
		return constraint
	} else if maxVersion != nil {
		constraint, _ := version.NewConstraint(fmt.Sprintf("<= %s", maxVersion))
		return constraint
	}
	constraint, _ := version.NewConstraint(">= 0.0.0")
	return constraint
}

func resolveBinding(config *config, topoloyKey TopologyKey, minVersion *version.Version, maxVersion *version.Version, serviceName ServiceName) *binding {

	log.Printf("byway: -- Locating version table: %s --", serviceName)
	vTable := config.mapping[serviceName]
	if vTable == nil {
		log.Printf("byway: Could not locate version table for service. %s", serviceName)
		return nil
	}

	log.Printf("byway: Checking for topology %s", topoloyKey)
	topology := config.topologies[topoloyKey]
	if topology != nil {
		log.Printf("byway: Checking for %s in topology table", string(serviceName))

		specificVersion := topology[serviceName]

		if specificVersion != "" {
			log.Printf("byway: Topology definens specific version: %s:%s", string(serviceName), string(specificVersion))

			binding := vTable[specificVersion]
			return &binding
		}
	}

	log.Println("byway: Building version list ...")
	vList := make([]*version.Version, 0)
	for versionStr := range vTable {

		v, err := version.NewVersion(string(versionStr))
		if err != nil {
			log.Fatalf("byway: Could not parse version: %s, %s", versionStr, err.Error())
		}
		vList = append(vList, v)
	}

	log.Println("byway: Sorting version list ...")

	sort.Sort(version.Collection(vList))
	log.Printf("byway: Version list: %s", vList)

	constraint := bulidContraint(minVersion, maxVersion)
	log.Printf("byway: Version constraint:  %s", constraint)

	for i := len(vList) - 1; i >= 0; i-- {
		v := vList[i]

		if v != nil && constraint.Check(v) {
			log.Printf("byway: Accepted: %s", v)
			binding := vTable[VersionString(v.String())]
			return &binding
		}
		log.Printf("byway: Rejected: %s", v)

	}

	log.Printf("byway: Could not resolve binding for: %s %s  ", serviceName, constraint)

	return nil
}

func newBywayProxy(configChan chan *Config) *httputil.ReverseProxy {
	config := &config{}

	go func() {
		for {
			rawConfig := <-configChan
			config = mapConfig(rawConfig)
		}
	}()

	director := func(req *http.Request) {
		configSnapshot := config
		log.Println("byway: -----------ROUTE BEGIN-----------")

		req.URL.Host = req.Host
		req.URL = rewriteURL(configSnapshot, req.URL)
		req.Host = req.URL.Host

		topologyKey, minVersion, maxVersion, serviceName := extractRoutingParameters(req)
		binding := resolveBinding(configSnapshot, topologyKey, minVersion, maxVersion, serviceName)

		if binding != nil {
			req.URL = rewriteURL(config, req.URL)

			req.Header.Add("X-Forwarded-Host", req.Host)
			if binding.pathRewriteFn != nil {
				req.URL.Path = binding.pathRewriteFn(req.URL.Path)
			}
			req.URL.Scheme = binding.scheme
			req.URL.Host = binding.host
			req.Host = binding.headers["host"]
			if req.Host == "" {
				req.Host = binding.host
			}

			log.Printf("byway: Routing to %s\nHost Header: %s", req.URL, binding.headers["host"])

		}
		log.Println("byway: -----------ROUTE END-----------")
	}

	return &httputil.ReverseProxy{Director: director}
}

// Init run the router
func Init(serviceTable chan *Config, exit chan bool) {
	go func() {
		port := ":1090"
		fmt.Printf("Running on " + port + "!\n")
		proxy := newBywayProxy(serviceTable)

		err := http.ListenAndServe(port, proxy)
		if err != nil {
			log.Fatal(err)
		}
	}()
}
