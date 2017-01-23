package main

import (
	"flag"
	"net/http"

	"github.com/dbenque/toKube/deployer"
)

func main() {

	flag.Parse()
	deployer.AutoDeploy()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	http.ListenAndServe(":80", nil)
}
