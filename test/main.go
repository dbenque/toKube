package main

import (
	"flag"
	"fmt"
	"net/http"

	"os"

	"github.com/dbenque/toKube/deployer"
)

func main() {

	flag.Parse()
	deployer.AutoDeploy()

	fmt.Printf("Environment:\n%#v\n", os.Environ())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	http.ListenAndServe(":80", nil)
}
