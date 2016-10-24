package main

import (
	"fmt"

	"github.com/amerdrix/byway/config"
	"github.com/amerdrix/byway/core"
)

func main() {
	fmt.Println("Welcome to byway darwin!")

	config := make(chan *core.Config, 1)

	//bywayConfig.WatchRedis(config)
	bywayConfig.WatchConfigFile(config)

	core.Init(bywayConfig.LogConfig(config))
	exit := make(chan bool)
	<-exit
}
