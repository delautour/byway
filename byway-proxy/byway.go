package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/amerdrix/byway/core"
	"github.com/amerdrix/byway/redis"
	"gopkg.in/yaml.v2"
)


func mapEndpointConfig(endpointConfig endpointConfig) core.Binding {
	return core.Binding{
		Host:          endpointConfig.Host,
		Scheme:        endpointConfig.Scheme,
		Headers:       endpointConfig.Headers,
		PathRewriteFn: core.IdentityRewrite}
}

func watchConfigFile(channel chan *core.Config) {
	configFile, err := ioutil.ReadFile("./conf.yml")
	if err != nil {
		log.Fatal(err)
	}

	configFromFile := make(map[string]map[string]endpointConfig)
	yaml.Unmarshal(configFile, &configFromFile)

	log.Println("byway: Loading config")

	newConfig := core.NewConfig()
	for serviceName, versionMap := range configFromFile {
		versionTable := make(map[core.VersionString]core.Binding)
		for versionStr, endpointConfig := range versionMap {
			binding := mapEndpointConfig(endpointConfig)

			versionTable[core.VersionString(versionStr)] = binding
		}

		newConfig.Mapping[core.ServiceName(serviceName)] = versionTable
	}
	channel <- newConfig
}

func main() {
	fmt.Println("Welcome to byway!")
	configWriter := make(chan *core.Config)
	configReader := make(chan *core.Config)

	go func() {
		for {
			table := <-configReader

			loaded, _ := yaml.Marshal(table)
			fmt.Printf("byway: config updated\n%s", loaded)

			configWriter <- table
		}
	}()

	core.Init(configWriter)

	bywayRedis.WatchRedis(configReader)

	//watchConfigFile(serviceTableReader)

	exit := make(chan bool)
	<-exit
}
