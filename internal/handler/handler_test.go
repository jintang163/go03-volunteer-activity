package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go03-volunteer-activity/internal/auth"
	"go03-volunteer-activity/internal/model"
	"go03-volunteer-activity/internal/service"
	"go03-volunteer-activity/internal/store"
)

func TestHealthAndLogin(t *testing.T) {
	st := store.NewMemoryStore(nil, nil)
	hsh := auth.NewPasswordHasher()
	sm := auth.NewSessionManager(time.Hour)
	svc := service.NewServices(st, hsh, sm, nil, 3)
	salt, hash, it, _ := hsh.Hash("alice123")
	_, err := st.CreateUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), model.User{
		Username: "alice", DisplayName: "Alice", Role: model.RoleVolunteer, Status: model.UserActive,
		PasswordSalt: salt, PasswordHash: hash, Iterations: it,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := New(svc, st, sm, nil)
	mux := http.NewServeMux()
	h.Routes(mux)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != 200 {
		t.Fatalf("health=%d", rr.Code)
	}

	body, _ := json.Marshal(model.LoginRequest{Username: "alice", Password: "alice123"})
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("login=%d %s", rr.Code, rr.Body.String())
	}
	var out model.LoginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil || out.Token == "" {
		t.Fatalf("token %#v %v", out, err)
	}
}
