package usersessions

import (
	"encoding/json"
	"github.com/foomo/variant-balancer/config"
	"github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const cookieName = "testCookie"

var sessionCounter = 0

func makeNode(id string) (nodeConfig *config.Node, server *httptest.Server) {
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, cookieErr := r.Cookie(cookieName)
		if cookieErr != nil {
			sessionCounter++
			sessionId := "sess-" + id + strconv.Itoa(sessionCounter)
			cookie := &http.Cookie{
				Name:   cookieName,
				Value:  sessionId,
				Path:   "/",
				Domain: r.URL.Host,
			}
			http.SetCookie(w, cookie)
		}
		w.Write([]byte("hello-" + id))
	}))
	nodeConfig = &config.Node{
		Id:             id,
		Server:         server.URL,
		Cookie:         cookieName,
		MaxConnections: 100,
	}
	return nodeConfig, server
}

func TestUserSessions(t *testing.T) {
	nodeA1, serverA1 := makeNode("a-1")
	nodeA2, serverA2 := makeNode("a-2")
	defer serverA1.Close()
	defer serverA2.Close()

	nodesVariantA := []*config.Node{nodeA1, nodeA2}

	variantConfigA := &config.Variant{
		Id:    "A",
		Share: 90,
		Nodes: nodesVariantA,
	}

	call := func(us *Sessions, v *Variant, clientSessionId string, path string) (writer *httptest.ResponseRecorder, sessionId string, err error) {
		req, _ := http.NewRequest("GET", "http://127.0.0.1"+path, nil)
		writer = httptest.NewRecorder()
		if len(clientSessionId) > 0 {
			cookie := &http.Cookie{
				Name:   cookieName,
				Value:  clientSessionId,
				Path:   "/",
				Domain: req.URL.Host,
			}
			req.AddCookie(cookie)
		}
		sessionId, err = us.serveVariant(v, writer, req, "")
		return writer, sessionId, err
	}

	nodeB1, serverB1 := makeNode("b-1")
	nodeB2, serverB2 := makeNode("b-2")
	defer serverB1.Close()
	defer serverB2.Close()

	nodesVariantB := []*config.Node{nodeB1, nodeB2}

	variantConfigB := &config.Variant{
		Id:    "B",
		Share: 10,
		Nodes: nodesVariantB,
	}

	c := &config.Config{
		Id:       "First",
		Variants: []*config.Variant{variantConfigA, variantConfigB},
	}

	convey.Convey("Given i fire up a usersessions instance", t, func() {
		us := NewSessions(c)
		variantA := us.Variants[variantConfigA.Id]
		variantB := us.Variants[variantConfigB.Id]
		convey.Convey("Then I can get a random variant", func() {
			convey.So(us.getRandomVariant(), convey.ShouldNotBeNil)
		})
		convey.Convey("When I put some traffic on the universe", func() {
			for i := 0; i < 1000; i++ {
				randomVariant := us.getRandomVariant()

				writer, sessionId, err := call(us, randomVariant, "", "/")
				if err != nil {
					t.Fatal(err)
				}
				if len(sessionId) == 0 {
					t.Fatal("empty session wtf", i, writer)
				}
				if writer.Body.Len() == 0 {
					t.Fatal("empty body wtf")
				}
				call(us, randomVariant, sessionId, "/foo")
				call(us, randomVariant, sessionId, "/bar")
				//convey.So(randomVariant, convey.ShouldNotBeNil)
			}
			t.Log("share a", variantA.Share)
			t.Log("share b", variantB.Share)
			jsonDump(t, us.getStatsForVariant(variantA))
			jsonDump(t, us.getStatsForVariant(variantB))
			//t.Log(us.getStatsForVariant(variantB).Pageviews, us.UserSessions)
			convey.Convey("Then we should see some sessions and some load", func() {
				convey.So(len(variantA.SessionIds), convey.ShouldBeGreaterThan, 450)
				convey.So(len(variantB.SessionIds), convey.ShouldBeGreaterThan, 450)
			})
		})
	})
}

func jsonDump(t *testing.T, v interface{}) {
	jsonBytes, err := json.MarshalIndent(v, "", "	")
	t.Log(string(jsonBytes), err)
}
