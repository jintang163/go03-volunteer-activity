package auth

import (
	"testing"
	"time"

	"go03-volunteer-activity/internal/model"
)

func TestHashVerify(t *testing.T) {
	h := NewPasswordHasher()
	salt, hash, it, err := h.Hash("secret1")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Verify("secret1", salt, hash, it) {
		t.Fatal("should match")
	}
	if h.Verify("secret2", salt, hash, it) {
		t.Fatal("should not match")
	}
}

func TestSessionLifecycle(t *testing.T) {
	sm := NewSessionManager(time.Hour)
	token, err := sm.Create(model.User{ID: "u_1", Username: "a", Role: model.RoleVolunteer})
	if err != nil || token == "" {
		t.Fatalf("create %v %s", err, token)
	}
	sess, err := sm.Get(token)
	if err != nil || sess.UserID != "u_1" {
		t.Fatalf("get %#v %v", sess, err)
	}
	sm.Invalidate(token)
	if _, err := sm.Get(token); err == nil {
		t.Fatal("expected invalid")
	}
}
