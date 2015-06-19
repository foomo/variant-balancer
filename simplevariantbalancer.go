package main

import (
	b "github.com/foomo/variant-balancer/variantbalancer"
	"log"
	"net/http"
	"strings"
)

type simpleHandler struct {
	balancer *b.Balancer
}

const routeAPI = "/balancer-api/"

func (s *simpleHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//w.Write([]byte("hello"))
	log.Println("serving", req.URL.Path)
	switch true {
	case strings.HasPrefix(req.URL.Path, routeAPI):
		s.balancer.Service.ServeHTTP(routeAPI, w, req)
	default:
		err := s.balancer.ServeHTTP(w, req, s.getCacheId(req.URL.Path))
		if err != nil {
			log.Println("balancer error", err)
		}
	}

}

func (s *simpleHandler) getCacheId(path string) string {
	// this is obviously a naive implementation
	switch true {
	case strings.HasSuffix(path, ".txt"), strings.HasSuffix(path, ".css"), strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".gif"), strings.HasSuffix(path, ".png"), strings.HasSuffix(path, ".js"):
		return path
	}
	return ""
}

func main() {
	http.ListenAndServe("0.0.0.0:8080", &simpleHandler{
		balancer: b.NewBalancer(),
	})
}
