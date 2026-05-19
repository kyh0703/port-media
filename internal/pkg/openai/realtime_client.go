package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/kyh0703/portfoilo-media/configs"
)

type RealtimeClient struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

type RealtimeCallCreator interface {
	CreateCall(ctx context.Context, input CreateCallInput) (CreateCallResult, error)
}

type RealtimeCallHanger interface {
	HangupCall(ctx context.Context, providerCallID string) error
}

type RealtimeCallManager interface {
	RealtimeCallCreator
	RealtimeCallHanger
}

type CreateCallInput struct {
	SDPOffer string
}

type CreateCallResult struct {
	SDPAnswer      string
	ProviderCallID string
}

func NewRealtimeClient(cfg *configs.Config) *RealtimeClient {
	return &RealtimeClient{
		baseURL:    strings.TrimRight(cfg.OpenAI.RealtimeBaseURL, "/"),
		model:      cfg.OpenAI.RealtimeModel,
		apiKey:     cfg.OpenAI.APIKey,
		httpClient: http.DefaultClient,
	}
}

func (c *RealtimeClient) CreateCall(ctx context.Context, input CreateCallInput) (CreateCallResult, error) {
	sdpOffer := strings.TrimSpace(input.SDPOffer)
	if sdpOffer == "" {
		return CreateCallResult{}, fmt.Errorf("create realtime call: empty SDP offer")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("sdp", sdpOffer); err != nil {
		return CreateCallResult{}, fmt.Errorf("write sdp field: %w", err)
	}

	session, err := json.Marshal(map[string]any{
		"type":  "realtime",
		"model": c.model,
	})
	if err != nil {
		return CreateCallResult{}, fmt.Errorf("encode session: %w", err)
	}
	if err := writer.WriteField("session", string(session)); err != nil {
		return CreateCallResult{}, fmt.Errorf("write session field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return CreateCallResult{}, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.callsURL(), body)
	if err != nil {
		return CreateCallResult{}, fmt.Errorf("build realtime call request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return CreateCallResult{}, fmt.Errorf("send realtime call request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	answer, err := io.ReadAll(res.Body)
	if err != nil {
		return CreateCallResult{}, fmt.Errorf("read realtime call response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return CreateCallResult{}, fmt.Errorf("create realtime call failed status=%d body=%s", res.StatusCode, string(answer))
	}

	return CreateCallResult{
		SDPAnswer:      string(answer),
		ProviderCallID: providerCallID(res.Header.Get("Location")),
	}, nil
}

func (c *RealtimeClient) HangupCall(ctx context.Context, providerCallID string) error {
	callID := strings.TrimSpace(providerCallID)
	if callID == "" {
		return fmt.Errorf("hangup realtime call: empty provider call id")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.callHangupURL(callID), nil)
	if err != nil {
		return fmt.Errorf("build realtime hangup request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send realtime hangup request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("hangup realtime call failed status=%d body=%s", res.StatusCode, string(body))
	}

	return nil
}

func (c *RealtimeClient) callsURL() string {
	if c.baseURL == "" {
		return "https://api.openai.com/v1/realtime/calls"
	}

	return c.baseURL + "/v1/realtime/calls"
}

func (c *RealtimeClient) callHangupURL(providerCallID string) string {
	return c.callsURL() + "/" + url.PathEscape(providerCallID) + "/hangup"
}

func providerCallID(location string) string {
	if location == "" {
		return ""
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return path.Base(location)
	}

	return path.Base(parsed.Path)
}
