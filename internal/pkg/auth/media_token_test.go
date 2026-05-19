package auth

import (
	"context"
	"errors"
	"testing"
)

type fakeMediaTokenStore struct {
	token MediaToken
	err   error
}

func (s fakeMediaTokenStore) Get(ctx context.Context, token string) (MediaToken, error) {
	_ = ctx
	_ = token
	return s.token, s.err
}

func TestMediaTokenVerifierReturnsRedisBackedClaims(t *testing.T) {
	verifier := NewMediaTokenVerifier(fakeMediaTokenStore{
		token: MediaToken{
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
	verifier := NewMediaTokenVerifier(fakeMediaTokenStore{err: ErrMediaTokenNotFound})

	if _, err := verifier.Verify(context.Background(), "expired-token"); !errors.Is(err, ErrInvalidMediaToken) {
		t.Fatalf("Verify() error = %v, want ErrInvalidMediaToken", err)
	}
}
