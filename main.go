package main

import (
	"fmt"
	"os"

	"github.com/Y4shin/open-caucus/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
