package handlers

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/pivotal-golang/lager"
)

type AddKeyHandler struct {
	sshPath string
	logger  lager.Logger
}

func NewAddKeyHandler(sshPath string, logger lager.Logger) *AddKeyHandler {
	return &AddKeyHandler{
		sshPath: sshPath,
		logger:  logger,
	}
}

func (ak *AddKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := ak.logger.Session("add-key")
	f, err := os.OpenFile(path.Join(ak.sshPath, "authorized_keys"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Error("cannot-open-authorized-keys-file", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("cannot-ready-body", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	wrappedContent := "\n" + string(content) + "\n"
	log.Info("key-added", lager.Data{"key": string(content)})
	f.Write([]byte(wrappedContent))
	w.WriteHeader(http.StatusCreated)
}
