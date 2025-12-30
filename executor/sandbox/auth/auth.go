package auth

import "context"

type Identity struct {
	ID string
}

type Authenticator interface {
	Authenticate(ctx context.Context, metadata map[string]string) (Identity, error)
}
