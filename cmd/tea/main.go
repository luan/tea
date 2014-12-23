package main

import (
	"log"
	"net/http"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/luan/tea/handlers"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := cf_lager.New("tea")
	logger.Info("starting", lager.Data{"listen_port": 8080})
	http.Handle("/shell", handlers.NewShellHandler(logger))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
