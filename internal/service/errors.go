// Package service contains the domain-layer orchestration between repos
// and handlers. It does not import net/http.
package service

import "errors"

var (
	ErrOrgExists         = errors.New("organization already exists")
	ErrUnknownKey        = errors.New("unknown ssh key")
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteExpired     = errors.New("invite expired")
	ErrInviteAlreadyUsed = errors.New("invite already used")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrProjectNotFound   = errors.New("project not found")
	ErrProjectExists     = errors.New("project already exists")
	ErrEnvNotFound       = errors.New("environment not found")
	ErrEnvExists         = errors.New("environment already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserRevoked       = errors.New("user revoked")
	ErrSecretNotFound    = errors.New("secret not found")
)
