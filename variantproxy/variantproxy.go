package variantproxy

import (
	"bytes"
	"compress/gzip"
	"errors"
	"log"
	"net/http"

	"github.com/foomo/variant-balancer/config"
	//"strings"
	"github.com/foomo/variant-balancer/context"
)

// Proxy a proxy for a variant
type Proxy struct {
	Nodes []*Node
}

// Debug enable debugging for this package
var Debug = false

// NewProxy constructor
func NewProxy(c []*config.Node) *Proxy {
	nodes := []*Node{}
	for _, nodeConfig := range c {
		nodes = append(nodes, NewNode(nodeConfig))
	}
	return newProxy(nodes)
}

func newProxy(nodes []*Node) *Proxy {
	p := &Proxy{
		Nodes: nodes,
	}
	return p
}

// Serve serve a http request
func (p *Proxy) Serve(w http.ResponseWriter, incomingRequest *http.Request) (sessionID string, cookieName string, err error) {
	node, cookieName, sessionID := p.ResolveNode(incomingRequest)

	if node == nil {
		return "", "", errors.New("No node to serve response")
	}
	ctx := context.Get(incomingRequest)
	ctx.NodeID = node.ID
	ctx.SessionID = sessionID

	debug("serving from", node.ID, "for session", sessionID)
	srw := newSnifferResponseWriter(w, node.SessionCookieName)
	node.ServeHTTP(srw, incomingRequest)
	if len(srw.SessionId) > 0 {
		return srw.SessionId, srw.cookieName, nil
	}
	return sessionID, cookieName, nil
}

func compress(data []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

func debug(a ...interface{}) {
	if Debug {
		log.Println(a...)
	}
}
