package variantproxy

import (
	"bytes"
	"compress/gzip"
	"errors"
	"github.com/foomo/variant-balancer/config"
	"github.com/foomo/variant-balancer/variantproxy/cache"
	"io"
	"log"
	"net/http"
	//"strings"
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

func (p *Proxy) ServeHTTPAndCache(w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (sessionId string, cookieName string, err error) {
	node, sessionId, cookieName := p.ResolveNode(incomingRequest)
	if node == nil {
		return "", "", errors.New("No node to serve response")
	}
	debug("serving from", node.Id, "for session", sessionId)
	srw := newSnifferResponseWriter(w, node.SessionCookieName)
	incomingRequest.Host = node.Url.Host

	if len(cacheId) > 0 {
		// stuff that can be cached does not set cookies
		p.serveFromCacheWithNode(w, incomingRequest, node, cacheId)
		return sessionId, cookieName, nil
	} else {
		node.ServeHTTP(srw, incomingRequest)
		if len(srw.SessionId) > 0 {
			return srw.SessionId, srw.cookieName, nil
		} else {
			return sessionId, cookieName, nil
		}
	}
}

func serveFromCache(cached *cache.Item, w http.ResponseWriter, req *http.Request) {
	h := w.Header()
	h.Set("Content-Type", cached.Header.Get("Content-Type"))
	h.Set("Expires", time.Now().Add(time.Hour*24*30).Format(http.TimeFormat))

	encoding := cached.Header.Get("Content-Encoding")
	if len(encoding) > 0 {
		h.Set("Content-Encoding", encoding)
	}

	http.ServeContent(w, req, req.RequestURI, time.Now(), bytes.NewReader(cached.Data))
}

func (p *Proxy) serveFromCacheWithNode(w http.ResponseWriter, req *http.Request, node *Node, cacheId string) {
	debug("serve from cache with node", node.Id, cacheId)
	// check if there's an entry for the requested resource in the cache
	cached := p.cache.Get(cacheId)
	if cached != nil {
		// there is a cache entry
		debug("	Cache hit:", req.RequestURI)
		serveFromCache(cached, w, req)
	} else if p.cache.GetLock(cacheId) {
		// there is none and we got the job
		req.URL.Host = node.Url.Host
		req.URL.Scheme = node.Url.Scheme

		crw := NewCuriousResponseWriter(w)
		node.ServeHTTP(crw, req)
		data := crw.bytes

		// check if resources are javascript or css files, if true compress
		/*
			if (strings.HasSuffix(req.RequestURI, ".js") || strings.HasSuffix(req.RequestURI, ".css")) && !strings.Contains(crw.Header().Get("Content-Encoding"), "gzip") {
				debug("	compressing", crw.Header().Get("Content-Encoding"), req.RequestURI)
				data = compress(data)
			}
		*/
		// save resources to cache
		if crw.statusCode == 304 {
			debug("	304", "ID:", cacheId, "URI:", req.RequestURI)
			p.cache.Cancel(cacheId)
		} else if len(data) != 0 {
			debug("	saving", "ID:", cacheId, "Size:", len(data), "bytes", "URI:", req.RequestURI)
			p.cache.Save(cacheId, req.RequestURI, data, crw.Header())
		} else {
			// empty Item: dont save!
			debug("	Cancelling saving of ID:", cacheId, "because its empty!")
			p.cache.Cancel(cacheId)
		}
		io.Copy(w, bytes.NewReader(data))
	} else {
		// we have to wait until the running request is complete
		p.cache.WaitFor(cacheId)
		debug("	yay \\o/ it arrived", cacheId, req.URL)
		serveFromCache(p.cache.Get(cacheId), w, req)
	}
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
