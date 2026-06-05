package auth

import (
	"context"
	"errors"
	"testing"

	coreport "github.com/kyh0703/portfoilo-media/internal/core/port"
)

type fakeMediaTokenStore struct {
	token coreport.MediaToken
	err   error
}

func (s fakeMediaTokenStore) Get(ctx context.Context, token string) (coreport.MediaToken, error) {
	_ = ctx
	_ = token
	return s.token, s.err
}

func TestMediaTokenVerifierReturnsRedisBackedClaims(t *testing.T) {
	verifier := NewMediaTokenVerifier(fakeMediaTokenStore{
		token: coreport.MediaToken{
			SessionID:      "session-1",
			ConversationID: "conversation-1",
			UserID:         "user-1",
		},
	})

	claims, err := verifier.Verify(context.Background(), "opaque-token")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", claims.SessionID)
	}
	if claims.ConversationID != "conversation-1" {
		t.Fatalf("ConversationID = %q, want conversation-1", claims.ConversationID)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", claims.UserID)
	}
}

func TestMediaTokenVerifierRejectsExpiredOrMissingToken(t *testing.T) {
	verifier := NewMediaTokenVerifier(fakeMediaTokenStore{err: coreport.ErrMediaTokenNotFound})

	if _, err := verifier.Verify(context.Background(), "expired-token"); !errors.Is(err, coreport.ErrInvalidMediaToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidMediaToken", err)
	}
}
