package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ammend/shelllane/internal/app"
)

var version = "dev"

func main() {
	app.Version = version
	if err := app.New().Run(os.Args); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "shelllane:", err)
		}
		os.Exit(1)
	}
}
