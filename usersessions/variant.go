package usersessions

import (
	"net/http"

	"github.com/foomo/variant-balancer/config"
	vp "github.com/foomo/variant-balancer/variantproxy"
)

const (
	VariantHeaderKey = "Server-Variant"
)

type Variant struct {
	Id string
	//SessionIds []string
	Share float64
	Proxy *vp.Proxy
}

func NewVariant(c *config.Variant) *Variant {
	v := &Variant{
		Share: float64(c.Share) / 100.0,
		Proxy: vp.NewProxy(c.Nodes),
		Id:    c.Id,
	}
	return v
}

// Serve serve a http request
func (v *Variant) Serve(w http.ResponseWriter, incomingRequest *http.Request) (sessionID string, cookieName string, err error) {
	w.Header().Add(VariantHeaderKey, v.Id)
	return v.Proxy.Serve(w, incomingRequest)
}
