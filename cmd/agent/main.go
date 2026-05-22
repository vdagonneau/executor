package main

import (
	"fmt"

	"github.com/spf13/pflag"
)

var commitHash string

func main() {
	is_version := pflag.Bool("version", false, "version")
	pflag.Parse()

	if *is_version {
		fmt.Print(commitHash)
		return
	}
}
