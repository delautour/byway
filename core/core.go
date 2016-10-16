package core

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

// Headers - a list of headers to set
type Headers map[string]string

// Rewrite - a path rewrite func
type Rewrite func(string) string

// Binding - a endpoint binding
type Binding struct {
	Host          string
	Scheme        string
	PathRewriteFn Rewrite `yaml:"-"`
	Headers       Headers
}

// VersionString - A version of a service 1.0.0
type VersionString string

// ServiceName - A name of a service
type ServiceName string

// ServiceTable - A list map of service / version bindings
type ServiceTable map[ServiceName]map[VersionString]Binding

// IdentityRewrite a rewrite rule which is the identity
func IdentityRewrite(input string) string {
	return input
}

// NewRegexReplaceRewrite returns a new rewrite rule based on a regular expression
func NewRegexReplaceRewrite(pattern string, replace string) Rewrite {
	re := regexp.MustCompile(pattern)

	fn := func(input string) string {
		result := re.ReplaceAllString(input, replace)
		log.Printf("byway: Rewrite %s -> %s", input, result)
		return result
	}
	return fn
}

func versionify(versionStr string) *version.Version {
	formatted := strings.Replace(versionStr, "-", ".", 3)
	v, err := version.NewVersion(formatted)
	if err != nil {
		return nil
	}
	return v
}

func extractRoutingParameters(req *http.Request) (*version.Version, *version.Version, ServiceName) {
	log.Print("byway: -- extractRoutingParameters --")
	var minVersion *version.Version
	var maxVersion *version.Version
	var serviceName string

	hostComponents := strings.Split(req.Host, ".")

	i := 0

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
		log.Printf("byway: Found min version from host: %s", minVersion)
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

	return minVersion, maxVersion, ServiceName(serviceName)
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

func resolveBinding(serviceTable ServiceTable, minVersion *version.Version, maxVersion *version.Version, serviceName ServiceName) *Binding {

	log.Printf("byway: -- Locating version table: %s --", serviceName)
	vTable := serviceTable[serviceName]
	if vTable == nil {
		log.Printf("byway: Could not locate version table")
		return nil
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

func newBywayProxy(serviceTableChan chan ServiceTable) *httputil.ReverseProxy {
	serviceTable := ServiceTable{}

	go func() {
		for {
			serviceTable = <-serviceTableChan
		}
	}()

	director := func(req *http.Request) {
		log.Println("byway: -----------ROUTE BEGIN-----------")
		minVersion, maxVersion, serviceName := extractRoutingParameters(req)
		binding := resolveBinding(serviceTable, minVersion, maxVersion, serviceName)

		if binding != nil {
			log.Printf("byway: Routing to %s://%s\nHost Header: %s", binding.Scheme, binding.Host, binding.Headers["host"])
			req.Header.Add("X-Forwarded-Host", req.Host)
			if binding.PathRewriteFn != nil {
				req.URL.Path = binding.PathRewriteFn(req.URL.Path)
			}
			req.URL.Scheme = binding.Scheme
			req.URL.Host = binding.Host
			req.Host = binding.Headers["host"]
			if req.Host == "" {
				req.Host = binding.Host
			}

		}
		log.Println("byway: -----------ROUTE END-----------")
	}

	return &httputil.ReverseProxy{Director: director}
}

// Init run the router
func Init(serviceTable chan ServiceTable) {
	go func() {
		port := ":31337"
		fmt.Printf("Running on " + port + "!\n")
		proxy := newBywayProxy(serviceTable)

		http.ListenAndServe(port, proxy)
	}()
}
