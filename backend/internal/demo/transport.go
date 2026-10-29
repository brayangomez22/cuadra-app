package demo

import (
	"net/http"
	"net/http/httptest"
)

// handlerDoer sends the generated client's requests straight to the API's
// handler, in the same process: no port, no network, the same contract
// validation and authorization as any client.
type handlerDoer struct{ h http.Handler }

// Do serves req with the handler and returns its response.
func (d handlerDoer) Do(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	// A server never hands a handler a nil body.
	if req.Body == nil {
		req.Body = http.NoBody
	}
	if req.RemoteAddr == "" {
		req.RemoteAddr = "127.0.0.1:0"
	}
	d.h.ServeHTTP(rec, req)
	return rec.Result(), nil
}
