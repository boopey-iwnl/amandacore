package authn

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"amandacore/services/internal/store"
)

func TestLoginAttemptsAreRateLimited(t *testing.T) {
	fileStore := newAuthTestStore(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, fileStore)

	for attempt := 0; attempt < authMutationLimit; attempt++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"limited","password":"wrong"}`))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "192.0.2.10:54000"

		mux.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d expected unauthorized before limit, got %d body=%s", attempt, response.Code, response.Body.String())
		}
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"limited","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "192.0.2.10:54000"
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate-limited login, got %d body=%s", response.Code, response.Body.String())
	}
}

func TestRegisterRejectsMalformedJSON(t *testing.T) {
	fileStore := newAuthTestStore(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, fileStore)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/accounts/register", bytes.NewBufferString(`{"username":`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected malformed request rejection, got %d", response.Code)
	}
}

func newAuthTestStore(t *testing.T) *store.FileStore {
	t.Helper()
	fileStore, err := store.NewFileStore(filepath.Join(t.TempDir(), "platform-state.json"), "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	return fileStore
}
