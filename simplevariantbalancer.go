package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/foomo/variant-balancer/variantbalancer"
)

type simpleHandler struct {
	balancer *variantbalancer.Balancer
}

const routeAPI = "/balancer-api/"

func (s *simpleHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Println("serving", req.URL.Path)
	switch true {
	case strings.HasPrefix(req.URL.Path, routeAPI):
		s.balancer.Service.ServeHTTP(routeAPI, w, req)
	default:
		err := s.balancer.ServeHTTP(w, req)
		if err != nil {
			log.Println("balancer error", err)
		}
	}

}

func main() {
	http.ListenAndServe("0.0.0.0:8080", &simpleHandler{
		balancer: variantbalancer.NewBalancer(),
	})
}
