package usersessions

import (
	"github.com/foomo/variant-balancer/config"
	vp "github.com/foomo/variant-balancer/variantproxy"
	"net/http"
)

type Variant struct {
	Id         string
	SessionIds []string
	Share      float64
	Proxy      *vp.Proxy
}

func NewVariant(c *config.Variant) *Variant {
	v := &Variant{
		Share: float64(c.Share) / 100.0,
		Proxy: vp.NewProxy(c.Nodes),
		Id:    c.Id,
	}
	return v
}

func (v *Variant) Serve(w http.ResponseWriter, incomingRequest *http.Request, cacheId string) (sessionId string, err error) {
	return v.Proxy.ServeHTTPAndCache(w, incomingRequest, cacheId)
}
