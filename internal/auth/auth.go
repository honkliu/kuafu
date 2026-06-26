package auth

import (
	"net/http"
	"strings"

	"github.com/microsoft/kuafu/internal/domain"
	kuafuerrors "github.com/microsoft/kuafu/internal/errors"
)

const UserHeader = "X-Kuafu-User"

type Authenticator interface {
	Authenticate(*http.Request) (*domain.User, error)
}

type HeaderAuthenticator struct {
	Users UserStore
}

type UserStore interface {
	GetUser(string) (*domain.User, error)
}

func (a HeaderAuthenticator) Authenticate(r *http.Request) (*domain.User, error) {
	alias := strings.TrimSpace(r.Header.Get(UserHeader))
	if alias == "" {
		return nil, kuafuerrors.New(kuafuerrors.KindPermanent, "missing X-Kuafu-User header")
	}
	user, err := a.Users.GetUser(alias)
	if err != nil {
		return nil, kuafuerrors.Wrap(kuafuerrors.KindPermanent, "unknown user", err)
	}
	return user, nil
}

type AnonymousAuthenticator struct{}

func (AnonymousAuthenticator) Authenticate(r *http.Request) (*domain.User, error) {
	return &domain.User{Alias: "anonymous"}, nil
}
