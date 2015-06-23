package variantbalancer

import (
	"encoding/json"
	"github.com/foomo/variant-balancer/config"
	us "github.com/foomo/variant-balancer/usersessions"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	balancer *Balancer
}

type Status struct {
	UserSessions []*us.SessionsStatus `json:"userSessions"`
}

func jsonReply(w http.ResponseWriter, data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "", "	")
	if err != nil {
		panic(err)
	}
	w.Write(jsonBytes)
}

func (s *Service) measureAndReply(w http.ResponseWriter, job func()) {
	start := time.Now().Unix()
	job()
	jsonReply(w, struct {
		Time    int64
		Success string
	}{
		Time:    time.Now().Unix() - start,
		Success: "ok",
	})
}

func (s *Service) ServeHTTP(routeAPI string, w http.ResponseWriter, incomingRequest *http.Request) {
	cmd := strings.TrimPrefix(incomingRequest.URL.Path, routeAPI)
	switch cmd {
	case "run":
		jsonBytes, _ := ioutil.ReadAll(incomingRequest.Body)
		c := new(config.Config)
		err := json.Unmarshal(jsonBytes, c)
		if err != nil {
			panic(err)
		}
		s.balancer.RunSession(c, false)
		jsonReply(w, c)
	case "status":
		jsonReply(w, Status{
			UserSessions: s.balancer.GetUserSessionsStatus(),
		})
	default:
		jsonReply(w, map[string]interface{}{
			"routes": []string{"status", "run"},
		})
	}
	incomingRequest.Body.Close()
}
