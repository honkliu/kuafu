package auth

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
)

func TestHeaderAuthenticator(t *testing.T) {
	repo := repository.NewMemoryRepository()
	user := &domain.User{Alias: "ada", CreatedAt: time.Now()}
	if err := repo.AddUser(user); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	authenticator := HeaderAuthenticator{Users: repo}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(UserHeader, "ada")

	got, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if got.Alias != "ada" {
		t.Fatalf("unexpected user: %#v", got)
	}
}

func TestHeaderAuthenticatorRejectsMissingUser(t *testing.T) {
	authenticator := HeaderAuthenticator{Users: repository.NewMemoryRepository()}
	req := httptest.NewRequest("GET", "/", nil)
	if _, err := authenticator.Authenticate(req); err == nil {
		t.Fatal("expected missing user error")
	}
}
