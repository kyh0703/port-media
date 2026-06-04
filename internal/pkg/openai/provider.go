package openai

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

type Provider struct {
	client *RealtimeClient
}

func NewProvider(client *RealtimeClient) port.RealtimeProvider {
	return &Provider{client: client}
}

func (p *Provider) CreateCall(ctx context.Context, input port.CreateCallInput) (port.CreateCallResult, error) {
	result, err := p.client.CreateCall(ctx, CreateCallInput{
		SDPOffer: input.SDPOffer,
	})
	if err != nil {
		return port.CreateCallResult{}, err
	}
	return port.CreateCallResult{
		SDPAnswer:      result.SDPAnswer,
		ProviderCallID: result.ProviderCallID,
	}, nil
}

func (p *Provider) HangupCall(ctx context.Context, providerCallID string) error {
	return p.client.HangupCall(ctx, providerCallID)
}
