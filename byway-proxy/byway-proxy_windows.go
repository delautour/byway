package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/amerdrix/byway/config"
	"github.com/amerdrix/byway/core"

	"golang.org/x/sys/windows/svc"

	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

var elog debug.Log

type bywayService struct{}

func (m *bywayService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	config := make(chan *core.Config, 1)
	exit := make(chan bool)

	// // elog.Error(0, "Config")
	bywayConfig.WatchRedis(config, exit)
	// bywayConfig.WatchConfigFile(config, exit)

	// // elog.Error(0, "Init")
	core.Init(bywayConfig.LogConfig(config), exit)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	go func() {
		<-exit
	}()

loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			exit <- true
			break loop
		default:

			elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

func installService(name, desc string) error {
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: name, Description: desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}
	return nil
}

func removeService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}
func main() {
	svcName := "Byway Proxy"
	interactive, _ := svc.IsAnInteractiveSession()
	if !interactive {
		svc.Run(svcName, &bywayService{})
	} else {
		log.Printf("%s", os.Args)
		if len(os.Args) == 2 {
			var err error
			cmd := strings.ToLower(os.Args[1])
			switch cmd {

			case "install":
				err = installService(svcName, "Byway proxy service")
			case "remove":
				err = removeService(svcName)
			default:
				log.Fatalf("unknown command: %s", cmd)
			}

			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("We will run byway here!")
		}

	}
}
