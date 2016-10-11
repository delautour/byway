package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
)

type headers map[string]string

type binding struct {
	host    string
	scheme  string
	headers headers
}

func newBinding(template binding) binding {
	r := binding{scheme: "http", headers: headers{}}
	r.host = template.host
	if template.scheme != "" {
		r.scheme = template.scheme
	}
	if template.headers != nil {
		r.headers = template.headers
	}

	return r
}

func localhostBinding(host string) binding {
	return newBinding(binding{host: "localhost:8081", headers: headers{"host": host}})
}

var serviceTable = map[string]map[string]binding{
	"search": {
		"1.0.0": newBinding(binding{host: "www.aol.com"}),
		"2.0.0": newBinding(binding{host: "www.yahoo.com"}),
		"3.0.0": newBinding(binding{host: "www.google.com"}),
		"4.0.0": newBinding(binding{host: "www.bing.com"}),
	},
	"echo": {
		"1.0.0": localhostBinding("1-0-0.echo"),
		"1.0.1": localhostBinding("1-0-1.echo"),
		"1.0.2": localhostBinding("1-0-2.echo"),
	},
}

func versionify(s string) *version.Version {
	formatted := strings.Replace(s, "-", ".", 3)
	v, err := version.NewVersion(formatted)
	if err != nil {
		return nil
	}
	return v
}

func extractRoutingParameters(req *http.Request) (*version.Version, *version.Version, string) {
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

	return minVersion, maxVersion, serviceName
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

func resolveBinding(minVersion *version.Version, maxVersion *version.Version, serviceName string) *binding {

	log.Printf("byway: -- Locating version table: %s --", serviceName)
	vTable := serviceTable[serviceName]
	if vTable == nil {
		log.Printf("byway: Could not locate version table")
		return nil
	}

	log.Println("byway: Building version list ...")
	vList := make([]*version.Version, 0)
	for versionStr := range vTable {
		v, err := version.NewVersion(versionStr)
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
			binding := vTable[v.String()]
			return &binding
		}
		log.Printf("byway: Rejected: %s", v)

	}

	log.Printf("byway: Could not resolve binding for: %s %s  ", serviceName, constraint)

	return nil
}

func newBywayProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		log.Println("byway: -----------ROUTE BEGIN-----------")
		minVersion, maxVersion, serviceName := extractRoutingParameters(req)
		binding := resolveBinding(minVersion, maxVersion, serviceName)

		if binding != nil {
			log.Printf("byway: Routing to %s://%s", binding.scheme, binding.host)
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.URL.Scheme = binding.scheme
			req.URL.Host = binding.host
			req.Host = binding.headers["host"]
		}
		log.Println("byway: -----------ROUTE END-----------")
	}

	return &httputil.ReverseProxy{Director: director}
}

func echo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Echo\n--------------------------------\n")
	fmt.Fprintf(w, "%s%s\n--------------------------------\n", r.Host, r.URL.RequestURI())
	for k, v := range r.Header {
		for _, v := range v {
			fmt.Fprintf(w, "%s: %s\n", k, v)
		}
	}
}

func main() {
	go func() {
		echoMux := http.NewServeMux()
		echoMux.HandleFunc("/", echo)
		http.ListenAndServe(":8081", echoMux)
	}()

	port := ":31337"
	fmt.Printf("Running byway on " + port + "!\n")
	proxy := newBywayProxy()

	http.ListenAndServe(port, proxy)
}
