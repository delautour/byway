package bywayConfig

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"

	"github.com/amerdrix/byway/core"
)

// WatchConfigFile watches config file for proxy config
func WatchConfigFile(channel chan *core.Config, exit chan bool) {
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

	log.Println("byway: Config loaded")

	channel <- newConfig
}
