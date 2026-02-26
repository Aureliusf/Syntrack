package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenAuth_AllowsHeaderToken(t *testing.T) {
	oldRequireAuth := requireAuth
	oldAuthTokens := authTokens
	t.Cleanup(func() {
		requireAuth = oldRequireAuth
		authTokens = oldAuthTokens
	})

	requireAuth = true
	authTokens = []string{"syntrack_token_valid"}

	nextCalled := false
	handler := tokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "example.com:8080"
	req.Header.Set("X-Auth-Token", "syntrack_token_valid")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestTokenAuth_AllowsQueryToken(t *testing.T) {
	oldRequireAuth := requireAuth
	oldAuthTokens := authTokens
	t.Cleanup(func() {
		requireAuth = oldRequireAuth
		authTokens = oldAuthTokens
	})

	requireAuth = true
	authTokens = []string{"syntrack_token_valid"}

	nextCalled := false
	handler := tokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?token=syntrack_token_valid", nil)
	req.Host = "example.com:8080"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestTokenAuth_RejectsWhenTokenMissingOrInvalid(t *testing.T) {
	oldRequireAuth := requireAuth
	oldAuthTokens := authTokens
	t.Cleanup(func() {
		requireAuth = oldRequireAuth
		authTokens = oldAuthTokens
	})

	requireAuth = true
	authTokens = []string{"syntrack_token_valid"}

	handler := tokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	missingReq := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	missingReq.Host = "example.com:8080"
	missingRR := httptest.NewRecorder()
	handler.ServeHTTP(missingRR, missingReq)

	if missingRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for missing token, got %d", missingRR.Code)
	}

	invalidReq := httptest.NewRequest(http.MethodGet, "http://example.com/?token=syntrack_token_invalid", nil)
	invalidReq.Host = "example.com:8080"
	invalidRR := httptest.NewRecorder()
	handler.ServeHTTP(invalidRR, invalidReq)

	if invalidRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for invalid token, got %d", invalidRR.Code)
	}
}
