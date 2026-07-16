//go:build !windows && !darwin && !linux

package github

import (
	"context"
	"errors"
)

type unavailableSessionStore struct{}

func newPlatformSessionStore() SessionStore {
	return unavailableSessionStore{}
}

func (unavailableSessionStore) LoadActive(context.Context, string) (Session, error) {
	return Session{}, errors.New("no supported native GitHub App secret store is available on this platform")
}

func (unavailableSessionStore) SaveActive(context.Context, Session) error {
	return errors.New("no supported native GitHub App secret store is available on this platform")
}

func (unavailableSessionStore) DeleteActive(context.Context, string) error {
	return errors.New("no supported native GitHub App secret store is available on this platform")
}
