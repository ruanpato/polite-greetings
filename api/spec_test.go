package api

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestSpecIsValid checks the contract against the OpenAPI 3.1 rules, including
// that its examples match their schemas.
func TestSpecIsValid(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(Spec)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !doc.IsOpenAPI31OrLater() {
		t.Fatalf("openapi = %q, want 3.1.x", doc.OpenAPI)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
