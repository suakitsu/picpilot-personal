package main

import (
	"os"

	"github.com/suakitsu/picpilot-platform/core/app"
	"github.com/suakitsu/picpilot-platform/core/config"
)

func main(){os.Exit(app.Run(config.Personal,os.Args[1:],os.Stdout,os.Stderr))}
