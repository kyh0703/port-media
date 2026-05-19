package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

type fakeStatsUsecase struct {
	fakeSessionUsecase
	stats sessiondto.RuntimeStatsResponse
}

func (f *fakeStatsUsecase) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	_ = ctx
	return f.stats, nil
}

func TestMetricsHandlerReturnsRuntimeStats(t *testing.T) {
	usecase := &fakeStatsUsecase{
		stats: sessiondto.RuntimeStatsResponse{
			Rooms:           1,
			Sessions:        1,
			Participants:    2,
			Tracks:          2,
			ByStatus:        map[string]int{"active": 1},
			ByConnection:    map[string]int{"connected": 1},
			ByMedia:         map[string]int{"active": 1},
			ByRole:          map[string]int{"client": 1, "openai_agent": 1},
			ByAudioMode:     map[string]int{"publisher": 1},
			ByRealtimeEvent: map[string]int{"response.done": 1},
		},
	}
	handler := NewMetricsHandler(usecase)

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler...)
	}

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body struct {
		StatusCode int                             `json:"statusCode"`
		Message    string                          `json:"message"`
		Data       sessiondto.RuntimeStatsResponse `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.StatusCode != http.StatusOK {
		t.Fatalf("statusCode = %d, want %d", body.StatusCode, http.StatusOK)
	}
	if body.Message != "OK" {
		t.Fatalf("message = %q, want OK", body.Message)
	}
	if body.Data.Participants != 2 {
		t.Fatalf("participants = %d, want 2", body.Data.Participants)
	}
	if body.Data.ByRole["openai_agent"] != 1 {
		t.Fatalf("openai_agent count = %d, want 1", body.Data.ByRole["openai_agent"])
	}
	if body.Data.ByAudioMode["publisher"] != 1 {
		t.Fatalf("publisher count = %d, want 1", body.Data.ByAudioMode["publisher"])
	}
	if body.Data.ByRealtimeEvent["response.done"] != 1 {
		t.Fatalf("response.done count = %d, want 1", body.Data.ByRealtimeEvent["response.done"])
	}
}
