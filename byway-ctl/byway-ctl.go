package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"encoding/json"

	"github.com/amerdrix/byway/config"
	"github.com/amerdrix/byway/core"
)

func cors(inner func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
		inner(w, r)
	}
}

func createRewrite(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {

	} else if r.Method == http.MethodPost {
		r.ParseForm()
		rewrite := string(r.Form["rewrite"][0])
		log.Println(rewrite)
		err := bywayConfig.CreateRewrite(core.RewriteConfigString(rewrite))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		fmt.Fprint(w, "ok")
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func createService(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {

	} else if r.Method == http.MethodPost {
		r.ParseForm()
		name := string(r.Form["name"][0])
		log.Println(name)
		err := bywayConfig.CreateService(core.ServiceName(name))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		fmt.Fprint(w, "ok")
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func createBinding(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {

	} else if r.Method == http.MethodPost {
		r.ParseForm()
		name := string(r.Form["service_name"][0])
		version := string(r.Form["version"][0])

		endpoint := core.EndpointConfig{
			Host:    r.Form["host"][0],
			Scheme:  r.Form["scheme"][0],
			Headers: make(map[string]string),
		}

		log.Println(name)
		err := bywayConfig.CreateBinding(core.ServiceName(name), core.VersionString(version), &endpoint)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		fmt.Fprint(w, "ok")
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func addServiceToTopology(w http.ResponseWriter, r *http.Request) {
	log.Println("addServiceToTopology ------------")
	if r.Method == http.MethodOptions {

	} else if r.Method == http.MethodPost {
		r.ParseForm()
		name := string(r.Form["service_name"][0])
		version := string(r.Form["service_version"][0])
		key := string(r.Form["topology_key"][0])

		log.Println(name)
		err := bywayConfig.AddServiceToTopology(core.TopologyKey(key), core.ServiceName(name), core.VersionString(version))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		fmt.Fprint(w, "ok")
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func deleteRewrite(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {

	} else if r.Method == http.MethodPost {

		err := r.ParseForm()
		log.Println(r.Form)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
			return
		}
		index, err := strconv.Atoi(r.FormValue("index"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
			return
		}
		err = bywayConfig.RemoveRewrite(int64(index), core.RewriteConfigString(r.FormValue("rewrite")))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprint(w, "delete ok")
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func serve(configChan chan *core.Config) func(http.ResponseWriter, *http.Request) {
	config := core.NewConfig()
	go func() {
		for {
			config = <-configChan
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		config := config

		js, err := json.Marshal(config)
		if err != nil {
			fmt.Fprintln(w, err)
		}

		fmt.Fprintln(w, string(js))
	}
}

func main() {
	port := ":1091"
	fmt.Printf("Running manage on port %s\n", port)

	config := make(chan *core.Config)
	exit := make(chan bool)
	bywayConfig.WatchRedis(config, exit)

	http.HandleFunc("/", cors(serve(config)))
	http.HandleFunc("/rewrite", cors(createRewrite))
	http.HandleFunc("/deleteRewrite", cors(deleteRewrite))

	http.HandleFunc("/createService", cors(createService))
	http.HandleFunc("/createBinding", cors(createBinding))
	http.HandleFunc("/addServiceToTopology", cors(addServiceToTopology))

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
	exit <- true
	fmt.Printf("Goodbye")

}
