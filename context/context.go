package context

/*
	Resources
	- https://joeshaw.org/revisiting-context-and-http-handler-for-go-17/
 */

import (
	"net/http"
	"context"
	"github.com/pkg/errors"
)

type key int

const (
	contextKey key = iota
)

var (
	ErrContextNotInitialized = errors.New("context was not initialized")
)

//Balancer request context
type Context struct {
	SessionID string
	VariantID string
	NodeID    string
}

func Initialize(r *http.Request) *Context {
	ctx := &Context{}
	*r = *r.WithContext(context.WithValue(r.Context(), contextKey, ctx))
	return ctx
}

func Get(r *http.Request) (ctx *Context) {
	if ctx, ok := r.Context().Value(contextKey).(*Context); ok {
		return ctx
	} else {
		return nil
	}
}
