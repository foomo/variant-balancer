package variantproxy

import (
	"net/http"
)

type CuriousResponseWriter struct {
	responseWriter http.ResponseWriter
	bytes          []byte
	statusCode     int
}

func NewCuriousResponseWriter(rw http.ResponseWriter) *CuriousResponseWriter {
	return &CuriousResponseWriter{
		responseWriter: rw,
		bytes:          []byte{},
		statusCode:     0,
	}
}

func (crw *CuriousResponseWriter) Header() http.Header {
	return crw.responseWriter.Header()
}

func (crw *CuriousResponseWriter) Write(bytes []byte) (int, error) {
	crw.bytes = append(crw.bytes, bytes...)
	return len(bytes), nil
}

func (crw *CuriousResponseWriter) WriteHeader(h int) {
	crw.statusCode = h
	crw.responseWriter.WriteHeader(h)
}
