package context

import (
	"testing"
	"net/http/httptest"
)

func TestContextInitialization(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := Initialize(req)
	ctx.VariantID = "variant"

	context := Get(req)

	if context.VariantID != "variant" {
		t.Error("variant must be initialized")
	}
}
