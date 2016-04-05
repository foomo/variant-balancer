package variantproxy

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/foomo/variant-balancer/config"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func makeSessionID() (string, error) {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// TODO: verify the two lines implement RFC 4122 correctly
	uuid[8] = 0x80 // variant bits see page 5
	uuid[4] = 0x40 // version 4 Pseudo Random, see page 7

	return "sess-" + hex.EncodeToString(uuid), nil
}

var hello = []byte("hello")

func TestUtils(t *testing.T) {
	// this one is a little weak, but hey it runs the code ;)
	c := compress([]byte("asc"))
	assert.True(t, len(c) > 0)
}

func getTestStuff(serverSleepTime time.Duration, maxConnections int) (ts *httptest.Server, node *Node) {
	return getTestServerAndNode("foo", "test", maxConnections, serverSleepTime)
}

func getTestServerAndNode(id string, cookieName string, maxConnections int, serverSleepTime time.Duration) (ts *httptest.Server, node *Node) {
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(serverSleepTime)
		_, cookieErr := r.Cookie(cookieName)
		if cookieErr != nil {
			sessionID, _ := makeSessionID()
			cookie := &http.Cookie{
				Name:   cookieName,
				Value:  sessionID,
				Path:   "/",
				Domain: r.URL.Host,
			}
			http.SetCookie(w, cookie)
		}
		w.Write(hello)
	}))
	nodeConfig := &config.Node{
		Id:             id,
		Server:         ts.URL,
		MaxConnections: maxConnections,
		Cookie:         cookieName,
	}
	return ts, NewNode(nodeConfig)
}

func TestNode(t *testing.T) {
	//Debug = true
	var wg sync.WaitGroup

	// long server sleep time needed in order to be able to observe load
	const maxConnections = 2
	ts, n := getTestStuff(time.Millisecond*100, maxConnections)
	defer ts.Close()

	call := func(n *Node, path string) {
		wg.Add(1)
		go func() {
			req, _ := http.NewRequest("GET", ts.URL+path, nil)
			writer := httptest.NewRecorder()
			n.ServeHTTP(writer, req)
			wg.Done()
		}()
	}

	convey.Convey("Given we start a node", t, func() {

		convey.Convey("So there should be no load", func() {
			convey.So(n.Load(), convey.ShouldEqual, 0.0)
		})

		convey.Convey("If we perform 4 parallel calls the load should go up to 2.0", func() {
			numCalls := float64(4)
			for i := float64(0); i < numCalls; i += 1.0 {
				call(n, "/")
			}

			// make sure that the calls have reached the server
			time.Sleep(time.Millisecond * 50)
			load := numCalls / float64(maxConnections)
			convey.So(n.Load(), convey.ShouldEqual, load)

			convey.Convey("once the requests were answered the load should be back to 0.0", func() {
				// make sure all calls are done and accounted for
				wg.Wait()
				time.Sleep(time.Millisecond * 10)
				convey.So(n.Load(), convey.ShouldEqual, 0.0)
			})

		})
	})

}

func TestCuriousResponseWriter(t *testing.T) {

	ts, n := getTestStuff(time.Millisecond, 2)
	defer ts.Close()
	req, _ := http.NewRequest("GET", ts.URL+"/", nil)
	curiousCat := NewCuriousResponseWriter(httptest.NewRecorder())
	n.ServeHTTP(curiousCat, req)

	convey.Convey("Given I use a curious response writer", t, func() {
		convey.Convey("i should be able to read the full server body and the statusCode", func() {
			convey.So(bytes.Compare(curiousCat.bytes, hello) == 0, convey.ShouldBeTrue)
			convey.So(curiousCat.statusCode, convey.ShouldEqual, http.StatusOK)
		})
	})
}

func TestNodeResolution(t *testing.T) {

	//Debug = true

	var wg sync.WaitGroup

	call := func(proxy *Proxy, cookieName string, sessionId string, path string) (string, error) {
		req, _ := http.NewRequest("GET", "http://127.0.0.1"+path, nil)
		if len(sessionId) > 0 {
			cookie := &http.Cookie{
				Name:   cookieName,
				Value:  sessionId,
				Path:   "/",
				Domain: req.URL.Host,
			}
			req.AddCookie(cookie)
		}
		writer := httptest.NewRecorder()
		extractedSessionID, proxyCookieName, err := proxy.Serve(writer, req)
		if err != nil {
			return "", err
		}
		if proxyCookieName != cookieName {
			return "", fmt.Errorf("cookieName %q did not match server cookie name %q", cookieName, proxyCookieName)
		}
		return extractedSessionID, nil
	}

	callConcurrently := func(concurrency int, proxy *Proxy, cookieName string, sessionId string) error {
		var e error
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			time.Sleep(time.Millisecond * 1)
			go func(ii int) {
				newSessionID, err := call(proxy, cookieName, sessionId, "/")
				if err != nil {
					e = err
				}
				if newSessionID != sessionId {
					t.Log("an existing session got fucked up: old session id:", sessionId, "=> new session id:", newSessionID, "for cookie:", cookieName)
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		return e
	}

	convey.Convey("Given we fire up a few servers with nodes in front of them", t, func() {

		cookieNameA := "foomoSessionA"
		cookieNameB := "foomoSessionB"

		const serverSleepTime = time.Millisecond * 10
		// session group a
		serverAA, nodeAA := getTestServerAndNode("aa", cookieNameA, 1, serverSleepTime)
		serverAB, nodeAB := getTestServerAndNode("ab", cookieNameA, 1, serverSleepTime)
		serverAC, nodeAC := getTestServerAndNode("ac", cookieNameA, 1, serverSleepTime)
		defer serverAA.Close()
		defer serverAB.Close()
		defer serverAC.Close()

		// session group b
		serverBA, nodeBA := getTestServerAndNode("ba", cookieNameB, 2, serverSleepTime)
		serverBB, nodeBB := getTestServerAndNode("bb", cookieNameB, 2, serverSleepTime)
		serverBC, nodeBC := getTestServerAndNode("bc", cookieNameB, 2, serverSleepTime)
		defer serverBA.Close()
		defer serverBB.Close()
		defer serverBC.Close()

		hitsInSessionGroup := func(nodes []*Node) func() int64 {
			return func() int64 {
				hits := int64(0)
				for _, n := range nodes {
					hits += n.Hits
				}
				return hits
			}
		}

		hitsInSessionGroupA := hitsInSessionGroup([]*Node{nodeAA, nodeAB, nodeAC})
		hitsInSessionGroupB := hitsInSessionGroup([]*Node{nodeBA, nodeBB, nodeBC})

		proxy := newProxy([]*Node{nodeAA, nodeAB, nodeAC, nodeBA, nodeBB, nodeBC})

		expectedHitsSessionGroupA := 0
		const expectedHitsSessionGroupB = 0

		convey.Convey("when we call the proxy, it calls the server and extracts a session id", func() {
			sessionID, err := call(proxy, cookieNameA, "", "/")
			if err != nil {
				t.Fatal(err)
			}
			expectedHitsSessionGroupA++
			convey.So(sessionID, convey.ShouldNotBeEmpty)

			convey.Convey("once we have a session and call with it concurrently", func() {
				const concurrency = 100
				err = callConcurrently(concurrency, proxy, cookieNameA, sessionID)
				if err != nil {
					t.Fatal(err)
				}
				expectedHitsSessionGroupA += concurrency

				convey.Convey("all hits have gone to session group a", func() {
					convey.So(hitsInSessionGroupA(), convey.ShouldEqual, expectedHitsSessionGroupA)

				})

				convey.Convey("no hits have gone to session group b", func() {
					convey.So(hitsInSessionGroupB(), convey.ShouldEqual, expectedHitsSessionGroupB)

				})

				convey.Convey("traffic will be distributed evenly in the session group A", func() {
					t.Log(
						"nodeAA.Hits:", nodeAA.Hits,
						"nodeAB.Hits:", nodeAB.Hits,
						"nodeAC.Hits:", nodeAC.Hits,
					)
					convey.So(nodeAA.Hits, convey.ShouldBeGreaterThan, 0)
					convey.So(nodeAB.Hits, convey.ShouldBeGreaterThan, 0)
					convey.So(nodeAC.Hits, convey.ShouldBeGreaterThan, 0)
					convey.So(nodeAA.Hits+nodeAB.Hits+nodeAC.Hits, convey.ShouldEqual, concurrency+1)
				})

			})

		})
	})

}
