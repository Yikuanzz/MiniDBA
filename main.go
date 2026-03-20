package main

import (
	"embed"
	"flag"
	"log"

	"mini-dba/internal/server"
)

//go:embed web/templates web/static
var webAssets embed.FS

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	srv, err := server.New(*cfgPath, webAssets)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
