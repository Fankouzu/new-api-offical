package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestRegisterRequestParsesFirstTouchAttribution(t *testing.T) {
	raw := `{
		"username":"private-user",
		"password":"private-password",
		"email":"private@example.com",
		"aff":"ABCD",
		"attribution":{
			"client_id":"111.222",
			"page_location":"https://lizh.ai/?utm_source=plati",
			"page_referrer":"https://plati.market/",
			"source":"plati",
			"medium":"marketplace",
			"campaign":"launch",
			"term":"chatgpt",
			"content":"card-a",
			"gclid":"gclid-value",
			"fbclid":"fbclid-value",
			"ttclid":"ttclid-value",
			"yclid":"yclid-value",
			"first_visit_at":"2026-06-12T10:00:00.000Z"
		}
	}`

	var req RegisterRequest
	if err := common.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("failed to parse register request: %v", err)
	}
	if req.Attribution.ClientID != "111.222" {
		t.Fatalf("client id = %q, want first-touch client id", req.Attribution.ClientID)
	}
	if req.Attribution.Source != "plati" || req.Attribution.Medium != "marketplace" {
		t.Fatalf("utm attribution missing: %#v", req.Attribution)
	}
	if req.Aff != "ABCD" {
		t.Fatalf("aff alias was not parsed: %#v", req)
	}

	params := req.Attribution
	if strings.Contains(params.ClientID, "private") ||
		strings.Contains(params.PageLocation, "private@example.com") {
		t.Fatalf("attribution should not contain account private data: %#v", params)
	}
}
