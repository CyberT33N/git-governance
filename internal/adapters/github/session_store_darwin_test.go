//go:build darwin

package github

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMacOSKeychainStorePreservesRefreshSessionsWithoutSecretArguments(t *testing.T) {
	runner := &fakeMacOSKeychainRunner{values: make(map[string][]byte)}
	store := &macOSKeychainStore{runner: runner}
	session := testStoredSession("github.com", "octocat")
	if err := store.SaveActive(context.Background(), session); err != nil {
		t.Fatalf("SaveActive() error = %v", err)
	}
	if strings.Contains(strings.Join(runner.arguments, " "), session.RefreshToken) {
		t.Fatalf("Keychain command arguments leaked a refresh token: %#v", runner.arguments)
	}
	loaded, err := store.LoadActive(context.Background(), "github.com")
	if err != nil || loaded != session {
		t.Fatalf("LoadActive() = (%#v, %v)", loaded, err)
	}
	if err := store.DeleteActive(context.Background(), "github.com"); err != nil {
		t.Fatalf("DeleteActive() error = %v", err)
	}
	if _, err := store.LoadActive(context.Background(), "github.com"); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("deleted LoadActive() error = %v", err)
	}
}

func TestMacOSKeychainStoreRejectsFailureModes(t *testing.T) {
	runner := &fakeMacOSKeychainRunner{values: make(map[string][]byte)}
	store := &macOSKeychainStore{runner: runner}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.SaveActive(ctx, testStoredSession("github.com", "octocat")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled SaveActive() error = %v", err)
	}
	if err := store.SaveActive(context.Background(), Session{}); err == nil {
		t.Fatal("SaveActive accepted an incomplete session")
	}
	runner.err = errors.New("keychain unavailable")
	if _, err := store.LoadActive(context.Background(), "github.com"); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("lookup failure = %v", err)
	}
	runner.err = nil
	runner.values[runner.key("github.com", macOSKeychainActiveAccount)] = []byte("octocat")
	runner.values[runner.key("github.com", "octocat")] = []byte("{")
	if _, err := store.LoadActive(context.Background(), "github.com"); err == nil {
		t.Fatal("LoadActive accepted malformed Keychain JSON")
	}
	if macOSKeychainService(" GitHub.COM ") != "git-governance.github-app.github.com" {
		t.Fatal("macOS Keychain service was not host-isolated")
	}
}

type fakeMacOSKeychainRunner struct {
	values    map[string][]byte
	err       error
	arguments []string
}

func (runner *fakeMacOSKeychainRunner) run(_ context.Context, input []byte, arguments ...string) ([]byte, error) {
	runner.arguments = append(runner.arguments, strings.Join(arguments, " "))
	if runner.err != nil {
		return nil, runner.err
	}
	service := macOSArgument(arguments, "-s")
	account := macOSArgument(arguments, "-a")
	key := service + "\x00" + account
	switch arguments[0] {
	case "find-generic-password":
		value, found := runner.values[key]
		if !found {
			return nil, errors.New("not found")
		}
		return append([]byte(nil), value...), nil
	case "add-generic-password":
		runner.values[key] = append([]byte(nil), input...)
		return nil, nil
	case "delete-generic-password":
		delete(runner.values, key)
		return nil, nil
	default:
		return nil, errors.New("unexpected Keychain command")
	}
}

func (runner *fakeMacOSKeychainRunner) key(host, account string) string {
	return macOSKeychainService(host) + "\x00" + account
}

func macOSArgument(arguments []string, flag string) string {
	for index := range arguments {
		if arguments[index] == flag && index+1 < len(arguments) {
			return arguments[index+1]
		}
	}
	return ""
}

var _ macOSKeychainRunner = (*fakeMacOSKeychainRunner)(nil)
