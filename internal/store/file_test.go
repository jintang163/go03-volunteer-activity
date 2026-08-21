package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go03-volunteer-activity/internal/model"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fs.Store().CreateUser(context.Background(), model.User{
		Username: "bob", DisplayName: "Bob", Role: model.RoleVolunteer, Status: model.UserActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	fs2, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	u, err := fs2.Store().GetUserByUsername(context.Background(), "bob")
	if err != nil || u.DisplayName != "Bob" {
		t.Fatalf("%#v %v", u, err)
	}
}
