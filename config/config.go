package bywayConfig

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/amerdrix/byway/core"
)

// LogConfig intercepts a chan and logs it
func LogConfig(input chan *core.Config) chan *core.Config {
	configWriter := make(chan *core.Config, 1)
	go func() {
		for {
			table := <-input

			loaded, _ := yaml.Marshal(table)
			fmt.Printf("byway: config updated\n%s", loaded)

			configWriter <- table
		}
	}()
	return configWriter
}
