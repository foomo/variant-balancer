package variantproxy

import (
	"net/http"
)

type SnifferResponseWriter struct {
	responseWriter http.ResponseWriter
	cookieName     string
	SessionId      string
	done           bool
}

func newSnifferResponseWriter(responseWriter http.ResponseWriter, cookieName string) *SnifferResponseWriter {
	return &SnifferResponseWriter{
		responseWriter: responseWriter,
		cookieName:     cookieName,
		done:           false,
	}
}

func (srw *SnifferResponseWriter) Header() http.Header {
	return srw.responseWriter.Header()
}
func (srw *SnifferResponseWriter) Write(bytes []byte) (int, error) {
	srw.returnSessionId()
	return srw.responseWriter.Write(bytes)
}
func (srw *SnifferResponseWriter) WriteHeader(code int) {
	srw.returnSessionId()
	srw.responseWriter.WriteHeader(code)
}

func (srw *SnifferResponseWriter) returnSessionId() {
	if !srw.done {
		srw.done = true
		cookies := readSetCookies(srw.responseWriter.Header())
		sessionId := ""
		for _, cookie := range cookies {
			if cookie.Name == srw.cookieName {
				sessionId = cookie.Value
				break
			}
		}
		//srw.sessionIdChannel <- sessionId
		if len(sessionId) > 0 {
			debug("sniffer response writer extracted session", sessionId, "for", srw.cookieName)
		} else {
			debug("sniffer response writer no session found")
		}

		srw.SessionId = sessionId
	}
}
