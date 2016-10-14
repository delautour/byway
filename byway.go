package main

import "github.com/amerdrix/byway/core"

func newBinding(template core.Binding) core.Binding {
	identity := func(i string) string {
		return i
	}
	r := core.Binding{Scheme: "http", Headers: core.Headers{}, PathRewrite: identity}
	r.Host = template.Host
	if template.Scheme != "" {
		r.Scheme = template.Scheme
	}
	if template.Headers != nil {
		r.Headers = template.Headers
	}
	if template.PathRewrite != nil {
		r.PathRewrite = template.PathRewrite
	}

	return r
}

func localhostBinding(host string) core.Binding {
	return newBinding(core.Binding{Host: "localhost:8081", Headers: core.Headers{"host": host}})
}

var serviceTable = core.ServiceTable{
	"search": {
		"1.0.0": newBinding(core.Binding{Host: "www.aol.com"}),
		"2.0.0": newBinding(core.Binding{Host: "www.yahoo.com"}),
		"3.0.0": newBinding(core.Binding{Host: "www.google.com"}),
		"4.0.0": newBinding(core.Binding{Host: "www.bing.com"}),
	},
	"echo": {
		"1.0.0": localhostBinding("1-0-0.echo.example.com"),
		"1.0.1": localhostBinding("1-0-1.echo.example.com"),
		"1.0.2": localhostBinding("1-0-2.echo.example.com"),
	},
}

func main() {
	core.Init(serviceTable)
}
