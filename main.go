package main

import (
	"log"

	"github.com/jbltx/master-server/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatal("An error has occured during execution of the process: %v", err)
	}
}
