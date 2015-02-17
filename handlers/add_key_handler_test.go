package handlers_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"

	. "github.com/luan/tea/handlers"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddKey", func() {
	var (
		logger           lager.Logger
		responseRecorder *httptest.ResponseRecorder
		sshPath          string
		handler          *AddKeyHandler
	)

	BeforeEach(func() {
		logger = lager.NewLogger("test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		responseRecorder = httptest.NewRecorder()
		sshPath, _ = ioutil.TempDir("", ".ssh-"+strconv.Itoa(GinkgoParallelNode()))
		handler = NewAddKeyHandler(sshPath, logger)
	})

	Describe("ServeHTTP", func() {
		Context("when everything succeeds", func() {
			JustBeforeEach(func() {
				handler.ServeHTTP(responseRecorder, newTestRequest("my-key"))
			})

			It("responds with 201 CREATED", func() {
				Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
			})

			It("responds with an empty body", func() {
				Expect(responseRecorder.Body.String()).To(Equal(""))
			})

			It("adds the key to authorized keys", func() {
				keys, err := authorizedKeys(sshPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(keys).To(ContainSubstring("my-key\n"))
			})
		})

		Context("when there's a malformed last line", func() {
			JustBeforeEach(func() {
				ioutil.WriteFile(path.Join(sshPath, "authorized_keys"), []byte("first-line"), 0600)
				handler.ServeHTTP(responseRecorder, newTestRequest("some-other-key"))
			})

			It("adds the key to authorized keys", func() {
				keys, err := authorizedKeys(sshPath)
				Expect(err).NotTo(HaveOccurred())
				Expect(keys).To(ContainSubstring("first-line\nsome-other-key\n"))
			})
		})
	})
})

func authorizedKeys(p string) (string, error) {
	contents, err := ioutil.ReadFile(path.Join(p, "/authorized_keys"))
	return string(contents), err
}
