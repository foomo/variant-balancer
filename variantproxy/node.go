package variantproxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

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

var CloseIdleProxyTransportConnectionsAfter = time.Second * 60

func NewNode(nodeConfig *config.Node) *Node {
	url, err := url.Parse(nodeConfig.Server)
	if err != nil {
		panic(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(url)
	if nodeConfig.InsecureSkipVerify {
		// unfourtunately there is no method to construct a default transport in the net/http package
		// there this is a copy of http.DefaultTransport
		myDefaultTransportInstance := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		// our magnificent change
		myDefaultTransportInstance.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		reverseProxy.Transport = myDefaultTransportInstance
	}
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
	debugConn := func(msg string) {
		if Debug {
			debug(msg, n.ID, "================================> open", n.openConnections, "hits", n.Hits, "load", n.Load())
		}
	}

	go func() {
		for {
			select {
			case <-time.After(CloseIdleProxyTransportConnectionsAfter):
				// idle connection maintenance
				// this should become obsolete:
				// https://github.com/golang/go/issues/6785 and others ...
				if n.ReverseProxy.Transport != nil {
					proxyTransport := n.ReverseProxy.Transport.(*http.Transport)
					if proxyTransport != nil {
						debugConn("closing idle connections")
						proxyTransport.CloseIdleConnections()
					} else {
						debugConn("can not close idle connections")
					}
				} else {
					debugConn("no proxy transport yet")
				}
			}
		}
	}()
	go func() {
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
