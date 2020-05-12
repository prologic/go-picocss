package main

import (
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	bind    string
	debug   bool
	version bool
)

func init() {
	flag.BoolVarP(&version, "version", "v", false, "display version information")
	flag.BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	flag.StringVarP(&bind, "bind", "b", "0.0.0.0:8000", "[int]:<port> to bind to")
}

func main() {
	flag.Parse()

	if version {
		fmt.Printf("go-picocss v%s", FullVersion())
		os.Exit(0)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.Infof("%s listening on http://%s", path.Base(os.Args[0]), bind)
	NewServer(bind).ListenAndServe()
}
