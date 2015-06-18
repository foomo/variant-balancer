package variantproxy

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"github.com/foomo/variant-balancer/config"
	"github.com/foomo/variant-balancer/variantproxy/cache"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Proxy struct {
	cache *cache.Cache
	Nodes []*Node
}

var Debug = false

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
		cache: cache.NewCache(time.Hour, time.Minute),
	}
	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, incomingRequest *http.Request) (sessionId string) {
	debug("=======================================================================================")
	node, sessionId := p.ResolveNode(incomingRequest)
	if node == nil {
		panic(errors.New("No node to serve response"))
	}
	debug("serving from", node.Id, "for session", sessionId)
	srw := newSnifferResponseWriter(w, node.SessionCookieName)
	incomingRequest.Host = node.Url.Host

	if p.canBeCached(incomingRequest) {
		// stuff that can be cached does not set cookies
		p.serveFromCacheWithNode(w, incomingRequest, node)
		return sessionId
	} else {
		node.ServeHTTP(srw, incomingRequest)
		if len(srw.SessionId) > 0 {
			return srw.SessionId
		} else {
			return sessionId
		}
	}
}

func serveFromCache(cached *cache.Item, w http.ResponseWriter, req *http.Request) {
	if cached == nil || len(cached.Data) == 0 {
		panic("Cache Item is empty!")
	}

	h := w.Header()
	h.Set("Content-Type", cached.Header.Get("Content-Type"))
	h.Set("Expires", time.Now().Add(time.Hour*24*30).Format(http.TimeFormat))

	encoding := cached.Header.Get("Content-Encoding")
	if len(encoding) > 0 {
		h.Set("Content-Encoding", encoding)
	}

	http.ServeContent(w, req, req.RequestURI, time.Now(), bytes.NewReader(cached.Data))
}

func (p *Proxy) serveFromCacheWithNode(w http.ResponseWriter, req *http.Request, node *Node) {
	debug("serve from cache with node", req.RequestURI)
	id := createHashFromUri(req.RequestURI)

	// check if there's an entry for the requested resource in the cache
	cached := p.cache.Get(id)
	if cached != nil {
		// there is a cache entry
		debug("	Cache hit:", req.RequestURI)
		serveFromCache(cached, w, req)
	} else if p.cache.GetLock(id) {
		// there is none and we got the job
		req.URL.Host = node.Url.Host
		req.URL.Scheme = node.Url.Scheme

		crw := NewCuriousResponseWriter(w)
		node.ServeHTTP(crw, req)
		data := crw.bytes

		// check if resources are javascript or css files, if true compress
		if (strings.HasSuffix(req.RequestURI, ".js") || strings.HasSuffix(req.RequestURI, ".css")) && !strings.Contains(crw.Header().Get("Content-Encoding"), "gzip") {
			debug("	compressing", crw.Header().Get("Content-Encoding"), req.RequestURI)
			data = compress(data)
		}
		// save resources to cache
		if crw.statusCode == 304 {
			debug("	304", "ID:", id, "URI:", req.RequestURI)
			p.cache.Cancel(id)
		} else if len(data) != 0 {
			debug("	saving", "ID:", id, "Size:", len(data), "bytes", "URI:", req.RequestURI)
			p.cache.Save(id, req.RequestURI, data, crw.Header())
		} else {
			// empty Item: dont save!
			debug("	Cancelling saving of ID:", id, "because its empty!")
			p.cache.Cancel(id)
		}
		io.Copy(w, bytes.NewReader(data))
	} else {
		// we have to wait until the running request is complete
		p.cache.WaitFor(id)
		debug("	yay \\o/ it arrived", id, req.URL)
		serveFromCache(p.cache.Get(id), w, req)
	}

}

func createHashFromUri(uri string) string {
	twentyBytes := sha1.Sum([]byte(uri))
	bytes := []byte{}
	return base64.URLEncoding.EncodeToString(append(bytes, twentyBytes[0:20]...))
}

func compress(data []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

func (p *Proxy) canBeCached(incomingRequest *http.Request) bool {
	switch true {
	case strings.HasPrefix(incomingRequest.RequestURI, "/images"):
		fallthrough
	case strings.HasSuffix(incomingRequest.RequestURI, ".txt"):
		fallthrough
	case strings.HasSuffix(incomingRequest.RequestURI, ".png"):
		fallthrough
	case strings.HasSuffix(incomingRequest.RequestURI, ".css"):
		fallthrough
	case strings.HasSuffix(incomingRequest.RequestURI, ".js"):
		fallthrough
	case strings.HasSuffix(incomingRequest.RequestURI, ".jpg"):
		return true
	}
	return false
}

func debug(a ...interface{}) {
	if Debug {
		log.Println(a...)
	}
}
