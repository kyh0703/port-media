package openai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
)

func TestRealtimeClientCreatesCallFromSDP(t *testing.T) {
	var receivedSession map[string]any
	var receivedSDP string
	var receivedAuth string
	offerSDP := "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/realtime/calls" {
			t.Fatalf("path = %q, want /v1/realtime/calls", r.URL.Path)
		}
		receivedAuth = r.Header.Get("Authorization")

		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		receivedSDP = r.FormValue("sdp")
		if err := json.Unmarshal([]byte(r.FormValue("session")), &receivedSession); err != nil {
			t.Fatalf("decode session form field: %v", err)
		}

		w.Header().Set("Location", "/v1/realtime/calls/rtc_123")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "answer-sdp")
	}))
	defer server.Close()

	client := NewRealtimeClient(&configs.Config{
		OpenAI: configs.OpenAIConfig{
			RealtimeBaseURL: server.URL,
			RealtimeModel:   "gpt-realtime-2",
			APIKey:          "test-key",
		},
	})

	result, err := client.CreateCall(t.Context(), CreateCallInput{SDPOffer: offerSDP})
	if err != nil {
		t.Fatalf("CreateCall() error = %v", err)
	}

	if result.SDPAnswer != "answer-sdp" {
		t.Fatalf("SDPAnswer = %q, want answer-sdp", result.SDPAnswer)
	}
	if result.ProviderCallID != "rtc_123" {
		t.Fatalf("ProviderCallID = %q, want rtc_123", result.ProviderCallID)
	}
	if receivedAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want Bearer test-key", receivedAuth)
	}
	if receivedSDP != offerSDP {
		t.Fatalf("sdp = %q, want exact SDP %q", receivedSDP, offerSDP)
	}
	if receivedSession["type"] != "realtime" {
		t.Fatalf("session.type = %v, want realtime", receivedSession["type"])
	}
	if receivedSession["model"] != "gpt-realtime-2" {
		t.Fatalf("session.model = %v, want gpt-realtime-2", receivedSession["model"])
	}
}

func TestRealtimeClientRejectsEmptySDP(t *testing.T) {
	client := NewRealtimeClient(&configs.Config{})

	_, err := client.CreateCall(t.Context(), CreateCallInput{SDPOffer: strings.TrimSpace("  ")})
	if err == nil {
		t.Fatal("CreateCall() error = nil, want error")
	}
}

func TestRealtimeClientHangsUpCall(t *testing.T) {
	var method string
	var requestPath string
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		requestPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewRealtimeClient(&configs.Config{
		OpenAI: configs.OpenAIConfig{
			RealtimeBaseURL: server.URL,
			APIKey:          "test-key",
		},
	})

	if err := client.HangupCall(t.Context(), "rtc_123"); err != nil {
		t.Fatalf("HangupCall() error = %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if requestPath != "/v1/realtime/calls/rtc_123/hangup" {
		t.Fatalf("path = %q, want /v1/realtime/calls/rtc_123/hangup", requestPath)
	}
	if receivedAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want Bearer test-key", receivedAuth)
	}
}
