package middleware

import (
	"bytes"
	"io/ioutil"
	"net/http"

	attest "github.com/takimoto3/app-attest"
	"github.com/takimoto3/app-attest-middleware/logger"
)

func AppAttestAssert(logger logger.Logger, appID string, plugin AssertionPlugin) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.SetContext(r.Context())
			var requestBody []byte
			if r.Body != nil {
				requestBody, _ = ioutil.ReadAll(r.Body)
				r.Body.Close()
				defer func() { r.Body = ioutil.NopCloser(bytes.NewBuffer(requestBody)) }()
			}
			r, assertion, challenge, err := plugin.ParseRequest(r, requestBody)
			if err != nil {
				logger.Errorf("%s", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			pubkey, counter, err := plugin.GetPublicKeyAndCounter(r)
			if err != nil {
				logger.Errorf("%s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if pubkey == nil {
				plugin.RedirectToAttestation(w, r)
				return
			}
			assginedChallenge, err := plugin.GetAssignedChallenge(r)
			if err != nil {
				logger.Errorf("%s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if assginedChallenge == "" {
				if err := plugin.ResponseNewChallenge(w, r); err != nil {
					logger.Errorf("%s", err)
					w.WriteHeader(http.StatusInternalServerError)
				}
				return
			}
			service := attest.AssertionService{
				AppID:     appID,
				Challenge: assginedChallenge,
				PublicKey: pubkey,
				Counter:   counter,
			}
			cnt, err := service.Verify(assertion, challenge, requestBody)
			if err != nil {
				logger.Errorf("%s", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err = plugin.StoreNewCounter(r, cnt); err != nil {
				logger.Errorf("%s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
