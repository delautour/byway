package main

import (
	"fmt"

	"github.com/amerdrix/byway/config"
	"github.com/amerdrix/byway/core"
	"golang.org/x/sys/windows/svc"
)

func main() {
	interactive, _ := svc.IsAnInteractiveSession()

	fmt.Printf("Welcome to byway - windows %b!", interactive)

	config := make(chan *core.Config, 1)

	//bywayConfig.WatchRedis(config)
	bywayConfig.WatchConfigFile(config)

	core.Init(bywayConfig.LogConfig(config))
	exit := make(chan bool)
	<-exit
}
