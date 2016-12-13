package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/dbenque/toKube/deployer"
)

func main() {
	flag.Parse()
	deployer.AutoDeploy()
	fmt.Println("I am waiting for signal")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	fmt.Println("Got signal:", s)
}
