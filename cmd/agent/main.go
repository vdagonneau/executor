package main

import (
	"fmt"

	"github.com/spf13/pflag"
)

func main() {
	is_version := pflag.Bool("version", false, "version")
	pflag.Parse()

	if *is_version {
		fmt.Println("0.0.1")
		return
	}
}
