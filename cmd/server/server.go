package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func main() {
	host := flag.String("host", "", "")

	flag.Parse()

	r := mux.NewRouter()

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if r.Host == *host {
			rw.Write([]byte("welcome"))
			return
		}

		subdomain := strings.Split(r.Host, ".")[0]

		rw.Write([]byte("tunnel: " + subdomain))
	})

	r.HandleFunc("/start", func(rw http.ResponseWriter, r *http.Request) {
		subdomain := r.URL.Query().Get("subdomain")
		if subdomain == "" {
			subdomain, _ = gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 10)
		}

		rw.Write([]byte(fmt.Sprintf("Starting: %s", subdomain)))
	}).Methods("POST").Host(*host)

	log.Println("starting...")

	http.ListenAndServe(":80", r)
}
