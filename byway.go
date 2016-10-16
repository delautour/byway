package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"encoding/json"

	"github.com/amerdrix/byway/core"
	"gopkg.in/redis.v5"
	"gopkg.in/yaml.v2"
)

type endpointConfig struct {
	Host    string
	Scheme  string
	Rewrite string
	Headers map[string]string
}

func mapEndpointConfig(endpointConfig endpointConfig) core.Binding {
	return core.Binding{
		Host:          endpointConfig.Host,
		Scheme:        endpointConfig.Scheme,
		Headers:       endpointConfig.Headers,
		PathRewriteFn: core.IdentityRewrite}

}

func watchConfigFile(channel chan core.ServiceTable) {
	configFile, err := ioutil.ReadFile("./conf.yml")
	if err != nil {
		log.Fatal(err)
	}

	config := make(map[string]map[string]endpointConfig)
	yaml.Unmarshal(configFile, &config)

	log.Println("byway: Loading config")

	newTable := make(core.ServiceTable)
	for serviceName, versionMap := range config {
		versionTable := make(map[core.VersionString]core.Binding)
		for versionStr, endpointConfig := range versionMap {
			binding := mapEndpointConfig(endpointConfig)

			versionTable[core.VersionString(versionStr)] = binding
		}

		newTable[core.ServiceName(serviceName)] = versionTable
	}
	channel <- newTable
}

func readRedisConfig(redis *redis.Client) core.ServiceTable {
	indexName := "byway.service_index"
	members := redis.SMembers(indexName)
	if members.Err() != nil {
		log.Fatalf("byway: redis: %s", members.Err())
	}

	log.Printf("byway: redis: %s\n", members)

	table := core.ServiceTable{}

	for _, serviceName := range members.Val() {
		versionTable := make(map[core.VersionString]core.Binding)

		vtable := redis.HGetAll("byway.service." + serviceName)
		if vtable.Err() != nil {
			log.Fatalf("byway: redis: %s", vtable.Err())
		}
		log.Printf("byway: redis: %s\n", vtable)

		for serviceVersion, endpoint := range vtable.Val() {
			log.Printf("byway: redis: %s:%s \n", serviceVersion, endpoint)

			ep := endpointConfig{}

			err := json.Unmarshal([]byte(endpoint), &ep)
			if err != nil {
				log.Fatalf("byway: redis: %s", err)
			}

			versionTable[core.VersionString(serviceVersion)] = mapEndpointConfig(ep)
		}

		table[core.ServiceName(serviceName)] = versionTable
	}

	return table
}

func watchRedis(channel chan core.ServiceTable) {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := client.Ping().Result()
	if err != nil {
		log.Fatalf("byway: redis: %s\n", err)
	}
	log.Printf("byway: redis: %s", pong)

	//subscription, err := client.Subscribe("byway_config")

	// if err != nil {
	// 	log.Fatalf("byway: redis: %s\n", err)
	// }
	//subscription.
	channel <- readRedisConfig(client)
	//subscription.

}

func main() {
	fmt.Println("Welcome to byway")
	serviceTableWriter := make(chan core.ServiceTable)
	serviceTableReader := make(chan core.ServiceTable)

	go func() {
		for {
			table := <-serviceTableReader

			loaded, _ := yaml.Marshal(table)
			log.Printf("byway: config updated\n%s", loaded)

			serviceTableWriter <- table
		}
	}()

	core.Init(serviceTableWriter)

	watchRedis(serviceTableReader)
	//watchConfigFile(serviceTableReader)

	exit := make(chan bool)
	<-exit
}
