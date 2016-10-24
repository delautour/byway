package main

import (
	"fmt"

	"github.com/amerdrix/byway/config"
	"github.com/amerdrix/byway/core"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

type bywayService struct{}

func (m *bywayService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			process.Exit()
			break loop
		default:
			debug.Log.Error(1, fmt.Sprintf("unexpected control request #%d", c))
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}

func main() {
	interactive, _ := svc.IsAnInteractiveSession()
	if !interactive {
		svc.Run("Byway", &bywayService{})
	}

	fmt.Printf("Welcome to byway - windows %b!", interactive)

	config := make(chan *core.Config, 1)

	//bywayConfig.WatchRedis(config)
	bywayConfig.WatchConfigFile(config)

	core.Init(bywayConfig.LogConfig(config))
	exit := make(chan bool)
	<-exit
}
