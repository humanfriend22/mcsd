package main

import (
	"log"

	_ "mcsd/web"

	"mcsd/cli"
	"mcsd/core"
)

func main() {
	if err := core.InitSDManager(); err != nil {
		log.Fatal(err)
	}
	defer core.ShutdownSDManager()
	cli.Execute()
}
