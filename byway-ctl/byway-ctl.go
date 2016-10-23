package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"encoding/json"

	"github.com/amerdrix/byway/core"
	"github.com/amerdrix/byway/redis"
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
		log.Println(r.Form)
		rewrite := string(r.Form["rewrite"][0])
		log.Println(rewrite)
		err := bywayRedis.CreateRewrite(core.RewriteConfigString(rewrite))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		fmt.Fprint(w, "ok")

	} else if r.Method == http.MethodDelete {
		log.Println(r.Form)
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err)
		}

		fmt.Fprint(w, "delete ok")
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
		err = bywayRedis.RemoveRewrite(int64(index), core.RewriteConfigString(r.FormValue("rewrite")))
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
	port := ":1081"
	fmt.Printf("Running manage on port %s\n", port)

	config := make(chan *core.Config)
	bywayRedis.WatchRedis(config)

	http.HandleFunc("/", cors(serve(config)))
	http.HandleFunc("/rewrite", cors(createRewrite))
	http.HandleFunc("/deleteRewrite", cors(deleteRewrite))
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Goodbye")

}
