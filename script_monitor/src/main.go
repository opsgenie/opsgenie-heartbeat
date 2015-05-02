package main

import (
	"os"
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var TIMEOUT = time.Second * 30
var apiUrl = "https://api.opsgenie.com"

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Version = "1.0"
	app.Usage = "Send hartbeats to OpsGenie"
	app.Author = "OpsGenie"
	app.Flags = SharedFlags
	app.Commands = Commands
	app.Run(os.Args)
}

var logAndExit = func(msg string) {
	log.Fatal(msg)
}
