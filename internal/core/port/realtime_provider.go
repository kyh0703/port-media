package port

import "context"

type RealtimeCallCreator interface {
	CreateCall(ctx context.Context, input CreateCallInput) (CreateCallResult, error)
}

type RealtimeCallHanger interface {
	HangupCall(ctx context.Context, providerCallID string) error
}

type RealtimeProvider interface {
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
