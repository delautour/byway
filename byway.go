package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/amerdrix/byway/core"
	"gopkg.in/yaml.v2"
)

func newBinding(template core.Binding) core.Binding {
	identity := func(i string) string {
		return i
	}
	r := core.Binding{Scheme: "http", Headers: core.Headers{}, PathRewriteFn: identity}
	r.Host = template.Host
	if template.Scheme != "" {
		r.Scheme = template.Scheme
	}
	if template.Headers != nil {
		r.Headers = template.Headers
	}
	if template.PathRewriteFn != nil {
		r.PathRewriteFn = template.PathRewriteFn
	}

	return r
}

func localhostBinding(host string) core.Binding {
	return newBinding(core.Binding{Host: "localhost:8081", Headers: core.Headers{"host": host}})
}

func watchConfig(currentTable **core.ServiceTable) {
	config, err := ioutil.ReadFile("./conf.yml")
	if err != nil {
		log.Fatal(err)
	}

	newTable := make(core.ServiceTable)
	yaml.Unmarshal(config, &newTable)

	log.Printf("byway: host -%s\n", newTable["echo"]["1.0.0"].Headers["host"])

	*currentTable = &newTable

}

func main() {
	fmt.Println("Welcome to byway")

	defaultTable := core.ServiceTable{}
	currentTable := &defaultTable
	watchConfig(&currentTable)

	core.Init(&currentTable)

	exit := make(chan bool)
	<-exit
}
