package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/amerdrix/byway/core"
	"gopkg.in/yaml.v2"
)

func watchConfigFile(channel chan *core.Config) {
	configFile, err := ioutil.ReadFile("./conf.yml")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("byway: Loading config")
	newConfig := core.NewConfig()

	err = yaml.Unmarshal(configFile, &newConfig)
	if err != nil {
		log.Fatal(err)
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

	//bywayRedis.WatchRedis(configReader)

	watchConfigFile(configReader)

	exit := make(chan bool)
	<-exit
}
