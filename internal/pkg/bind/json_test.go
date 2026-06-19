package bind

import (
	"net/http/httptest"
	"strings"
	"testing"
)

type jsonRequest struct {
	Name string `json:"name"`
}

func TestJSONBindsRequestBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"kim"}`))

	var req jsonRequest
	if err := JSON(r, &req); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	if req.Name != "kim" {
		t.Fatalf("JSON() Name = %q, want %q", req.Name, "kim")
	}
}

func TestJSONReturnsDecodeError(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{`))

	if err := JSON(r, &jsonRequest{}); err == nil {
		t.Fatal("JSON() error = nil, want error")
	}
}
