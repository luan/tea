package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/luan/tea/handlers"
	"github.com/mitchellh/go-homedir"
	"github.com/pivotal-golang/lager"
)

var secret = flag.String(
	"secret",
	"",
	"secret for accessing API",
)

func main() {
	cf_lager.AddFlags(flag.CommandLine)
	logger, _ := cf_lager.New("tea")

	home, err := homedir.Dir()
	if err != nil {
		logger.Fatal("cannot-get-user-info", err)
	}
	sshPath := path.Join(home, ".ssh")

	err = os.MkdirAll(sshPath, 0700)
	if err != nil {
		logger.Fatal("cannot-create-ssh-directory", err)
	}

	logger.Info("starting", lager.Data{"listen_port": 8080})
	http.Handle("/shell", handlers.NewShellHandler(logger))
	http.Handle("/ssh", handlers.NewSSHHandler(logger))
	http.Handle("/add-key/"+*secret, handlers.NewAddKeyHandler(sshPath, logger))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
