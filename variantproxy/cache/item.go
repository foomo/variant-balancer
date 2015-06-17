package cache

import (
	"net/http"
)

type Item struct {
	Id     string
	Uri    string
	Data   []byte
	Header http.Header
}
