//go:build linux

package github

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLinuxSecretServiceStorePreservesRefreshSessionsWithoutSecretArguments(t *testing.T) {
	runner := &fakeLinuxSecretTool{values: make(map[string][]byte)}
	store := &linuxSecretServiceStore{runner: runner}
	session := testStoredSession("github.com", "octocat")
	if err := store.SaveActive(context.Background(), session); err != nil {
		t.Fatalf("SaveActive() error = %v", err)
	}
	if strings.Contains(strings.Join(runner.arguments, " "), session.RefreshToken) {
		t.Fatalf("Secret Service command arguments leaked a refresh token: %#v", runner.arguments)
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

func TestLinuxSecretServiceStoreRejectsFailureModes(t *testing.T) {
	runner := &fakeLinuxSecretTool{values: make(map[string][]byte)}
	store := &linuxSecretServiceStore{runner: runner}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.SaveActive(ctx, testStoredSession("github.com", "octocat")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled SaveActive() error = %v", err)
	}
	if err := store.SaveActive(context.Background(), Session{}); err == nil {
		t.Fatal("SaveActive accepted an incomplete session")
	}
	runner.err = errors.New("Secret Service unavailable")
	if _, err := store.LoadActive(context.Background(), "github.com"); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("lookup failure = %v", err)
	}
	runner.err = nil
	runner.values[runner.key("github.com", linuxSecretActiveAccount)] = []byte("octocat")
	runner.values[runner.key("github.com", "octocat")] = []byte("{")
	if _, err := store.LoadActive(context.Background(), "github.com"); err == nil {
		t.Fatal("LoadActive accepted malformed Secret Service JSON")
	}
	if linuxSecretHost(" GitHub.COM ") != "github.com" {
		t.Fatal("Secret Service host key was not normalized")
	}
}

type fakeLinuxSecretTool struct {
	values    map[string][]byte
	err       error
	arguments []string
}

func (runner *fakeLinuxSecretTool) run(_ context.Context, input []byte, arguments ...string) ([]byte, error) {
	runner.arguments = append(runner.arguments, strings.Join(arguments, " "))
	if runner.err != nil {
		return nil, runner.err
	}
	host := linuxSecretArgument(arguments, "host")
	account := linuxSecretArgument(arguments, "account")
	key := runner.key(host, account)
	switch arguments[0] {
	case "lookup":
		value, found := runner.values[key]
		if !found {
			return nil, errors.New("not found")
		}
		return append([]byte(nil), value...), nil
	case "store":
		runner.values[key] = append([]byte(nil), input...)
		return nil, nil
	case "clear":
		delete(runner.values, key)
		return nil, nil
	default:
		return nil, errors.New("unexpected Secret Service command")
	}
}

func (runner *fakeLinuxSecretTool) key(host, account string) string {
	return linuxSecretHost(host) + "\x00" + account
}

func linuxSecretArgument(arguments []string, name string) string {
	for index := range arguments {
		if arguments[index] == name && index+1 < len(arguments) {
			return arguments[index+1]
		}
	}
	return ""
}

var _ linuxSecretToolRunner = (*fakeLinuxSecretTool)(nil)
