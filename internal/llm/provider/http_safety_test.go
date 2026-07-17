package provider

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecodeLimitedJSON(t *testing.T) {
	var decoded map[string]string
	if err := decodeLimitedJSON(strings.NewReader(`{"value":"ok"}`), &decoded); err != nil {
		t.Fatalf("decodeLimitedJSON() error = %v", err)
	}
	if decoded["value"] != "ok" {
		t.Fatalf("decoded value = %q, want ok", decoded["value"])
	}
}

func TestDecodeLimitedJSONRejectsOversizedBody(t *testing.T) {
	body := bytes.Repeat([]byte("x"), maxProviderResponseBytes+1)
	var decoded any
	if err := decodeLimitedJSON(bytes.NewReader(body), &decoded); err == nil {
		t.Fatal("decodeLimitedJSON() accepted an oversized response")
	}
}
