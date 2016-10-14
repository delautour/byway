package main

import (
	"fmt"
	"log"
	"net/http"
)

func echo(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s%s", r.Host, r.URL.RequestURI())
	fmt.Fprintf(w, "Echo\n--------------------------------\n")
	fmt.Fprintf(w, "%s%s\n--------------------------------\n", r.Host, r.URL.RequestURI())
	for k, v := range r.Header {
		for _, v := range v {
			fmt.Fprintf(w, "%s: %s\n", k, v)
		}
	}
}

func main() {
	port := ":8081"
	fmt.Printf("Echo running on port %s\n", port)
	echoMux := http.NewServeMux()
	echoMux.HandleFunc("/", echo)
	http.ListenAndServe(port, echoMux)
}
