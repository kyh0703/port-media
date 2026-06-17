package port

type ProviderEventNormalizer interface {
	Normalize(payload string) ConversationSignal
}

type ConversationSignal struct {
	Type                   ConversationEventType
	ProviderEventType      ProviderEventType
	ProviderEventID        string
	ProviderItemID         string
	PreviousProviderItemID string
	ProviderRespID         string
	Payload                string
	Publishable            bool
}
