package variantproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/foomo/variant-balancer/config"
)

// Node of a variant
type Node struct {
	Server            string
	URL               *url.URL
	SessionCookieName string
	ID                string
	openConnections   int
	maxConnections    int
	Hits              int64
	ReverseProxy      *httputil.ReverseProxy
	channelOpenConn   chan int
	channelCloseConn  chan int
	user              string
	password          string
}

func NewNode(nodeConfig *config.Node) *Node {
	url, err := url.Parse(nodeConfig.Server)
	if err != nil {
		panic(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(url)
	password := ""
	user := ""
	if url.User != nil {
		password, _ = url.User.Password()
		if len(password) > 0 {
			user = url.User.Username()
		}
	}
	n := &Node{
		Server:            nodeConfig.Server,
		URL:               url,
		Hits:              0,
		ID:                nodeConfig.Id,
		ReverseProxy:      reverseProxy,
		SessionCookieName: nodeConfig.Cookie,
		openConnections:   0,
		maxConnections:    nodeConfig.MaxConnections,
		channelOpenConn:   make(chan int),
		channelCloseConn:  make(chan int),
		user:              user,
		password:          password,
	}
	go func() {
		debugConn := func(msg string) {
			if Debug {
				debug(msg, n.ID, "================================> open", n.openConnections, "hits", n.Hits, "load", n.Load())
			}
		}
		for {
			select {
			case <-n.channelCloseConn:
				debugConn("node close conn")
				n.openConnections--
			case <-n.channelOpenConn:
				n.Hits++
				n.openConnections++
				debugConn("node open conn")
			}
		}
	}()
	return n
}

// Load calculate current load
func (n *Node) Load() float64 {
	if n.openConnections > 0 {
		l := float64(n.openConnections) / float64(n.maxConnections)
		return l
	}
	return 0.0
}

func (n *Node) closeConn() {
	n.channelCloseConn <- 1
}

func (n *Node) ServeHTTP(w http.ResponseWriter, incomingRequest *http.Request) {
	n.channelOpenConn <- 1
	defer func() {
		if err := recover(); err != nil {
			n.closeConn()
		}
	}()
	if len(n.user) > 0 && incomingRequest.URL.User == nil {
		incomingRequest.SetBasicAuth(n.user, n.password)
	}
	n.ReverseProxy.ServeHTTP(w, incomingRequest)
	n.closeConn()
}
