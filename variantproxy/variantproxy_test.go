package variantproxy

import (
	"bytes"
	"github.com/foomo/variant-balancer/config"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

var hello = []byte("hello")

func TestUtils(t *testing.T) {
	assert.True(t, len(createHashFromUri("/foo")) > 0)
	// this one is a little weak, but hey it runs the code ;)
	c := compress([]byte("asc"))
	assert.True(t, len(c) > 0)
}

func getTestStuff() (ts *httptest.Server, nodeConfig *config.Node) {
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * 100)
		w.Write(hello)
	}))
	nodeConfig = &config.Node{
		Id:             "foo",
		Server:         ts.URL,
		MaxConnections: 2,
		Cookie:         "test",
	}
	return ts, nodeConfig
}

func TestNode(t *testing.T) {

	var wg sync.WaitGroup

	ts, nodeConfig := getTestStuff()
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

		n := NewNode(nodeConfig)

		convey.Convey("So there should be no load", func() {
			convey.So(n.Load(), convey.ShouldEqual, 0.0)
		})

		convey.Convey("If we perform 4 parallel calls the load should go up to 2.0", func() {

			call(n, "/")
			call(n, "/")
			call(n, "/")
			call(n, "/")

			time.Sleep(time.Millisecond * 10)
			convey.So(n.Load(), convey.ShouldEqual, 2.0)

			convey.Convey("once the requests were answered the load should be back to 0.0", func() {
				wg.Wait()
				convey.So(n.Load(), convey.ShouldEqual, 0.0)
			})

		})
	})

}

func TestCuriousResponseWriter(t *testing.T) {

	ts, nodeConfig := getTestStuff()
	defer ts.Close()
	n := NewNode(nodeConfig)
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
