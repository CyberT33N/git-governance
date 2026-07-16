//go:build windows

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	sessionStoreSchemaVersion = 1
	dpapiEntropy              = "git-governance/github-app-session-store/v1"
)

type sessionTemporaryFile interface {
	Name() string
	Chmod(os.FileMode) error
	Write([]byte) (int, error)
	Close() error
}

var (
	userConfigDirectory = os.UserConfigDir
	readSessionFile     = os.ReadFile
	removeSessionFile   = os.Remove
	makeSessionDir      = os.MkdirAll
	createSessionTemp   = func(directory, pattern string) (sessionTemporaryFile, error) {
		return os.CreateTemp(directory, pattern)
	}
	renameSessionFile  = os.Rename
	cryptProtectData   = windows.CryptProtectData
	cryptUnprotectData = windows.CryptUnprotectData
	freeLocalMemory    = windows.LocalFree
)

type sessionDocument struct {
	SchemaVersion int                `json:"schemaVersion"`
	ActiveByHost  map[string]string  `json:"activeByHost"`
	Sessions      map[string]Session `json:"sessions"`
}

type dpapiSessionStore struct {
	path func() (string, error)
}

func newPlatformSessionStore() SessionStore {
	return &dpapiSessionStore{path: defaultDPAPISessionStorePath}
}

func newDPAPISessionStore(path string) *dpapiSessionStore {
	return &dpapiSessionStore{
		path: func() (string, error) {
			return path, nil
		},
	}
}

func (store *dpapiSessionStore) LoadActive(ctx context.Context, host string) (Session, error) {
	if err := contextFailure(ctx); err != nil {
		return Session{}, err
	}
	document, err := store.load()
	if err != nil {
		return Session{}, err
	}
	account, found := document.ActiveByHost[normalizeHost(host)]
	if !found || strings.TrimSpace(account) == "" {
		return Session{}, errSessionNotFound
	}
	session, found := document.Sessions[sessionKey(host, account)]
	if !found {
		return Session{}, errors.New("protected GitHub App session index is inconsistent")
	}
	return session, nil
}

func (store *dpapiSessionStore) SaveActive(ctx context.Context, session Session) error {
	if err := contextFailure(ctx); err != nil {
		return err
	}
	if err := validateStoredSession(session); err != nil {
		return err
	}
	document, err := store.load()
	if err != nil {
		return err
	}
	document.Sessions[sessionKey(session.Host, session.Account)] = session
	document.ActiveByHost[normalizeHost(session.Host)] = session.Account
	return store.save(document)
}

func (store *dpapiSessionStore) DeleteActive(ctx context.Context, host string) error {
	if err := contextFailure(ctx); err != nil {
		return err
	}
	document, err := store.load()
	if err != nil {
		return err
	}
	normalizedHost := normalizeHost(host)
	account, found := document.ActiveByHost[normalizedHost]
	if !found || strings.TrimSpace(account) == "" {
		return errSessionNotFound
	}
	delete(document.ActiveByHost, normalizedHost)
	delete(document.Sessions, sessionKey(host, account))
	return store.save(document)
}

func (store *dpapiSessionStore) load() (sessionDocument, error) {
	path, err := store.path()
	if err != nil {
		return sessionDocument{}, err
	}
	encrypted, err := readSessionFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return emptySessionDocument(), nil
	}
	if err != nil {
		return sessionDocument{}, fmt.Errorf("read protected GitHub App session: %w", err)
	}
	plain, err := unprotectDPAPI(encrypted)
	if err != nil {
		return sessionDocument{}, fmt.Errorf("decrypt protected GitHub App session: %w", err)
	}
	var document sessionDocument
	if err := json.Unmarshal(plain, &document); err != nil {
		return sessionDocument{}, errors.New("protected GitHub App session has an invalid format")
	}
	if document.SchemaVersion != sessionStoreSchemaVersion {
		return sessionDocument{}, errors.New("protected GitHub App session has an unsupported schema version")
	}
	if document.ActiveByHost == nil {
		document.ActiveByHost = make(map[string]string)
	}
	if document.Sessions == nil {
		document.Sessions = make(map[string]Session)
	}
	return document, nil
}

func (store *dpapiSessionStore) save(document sessionDocument) error {
	path, err := store.path()
	if err != nil {
		return err
	}
	if len(document.Sessions) == 0 {
		if err := removeSessionFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove protected GitHub App session: %w", err)
		}
		return nil
	}
	document.SchemaVersion = sessionStoreSchemaVersion
	// sessionDocument contains only JSON-encodable map, string, and time values.
	encoded, _ := json.Marshal(document)
	encrypted, err := protectDPAPI(encoded)
	if err != nil {
		return fmt.Errorf("encrypt protected GitHub App session: %w", err)
	}
	directory := filepath.Dir(path)
	if err := makeSessionDir(directory, 0o700); err != nil {
		return fmt.Errorf("create protected GitHub App session directory: %w", err)
	}
	temporary, err := createSessionTemp(directory, ".github-app-session-*")
	if err != nil {
		return fmt.Errorf("create protected GitHub App session file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer removeSessionFile(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect GitHub App session file: %w", err)
	}
	if _, err := temporary.Write(encrypted); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write protected GitHub App session: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close protected GitHub App session: %w", err)
	}
	if err := renameSessionFile(temporaryPath, path); err != nil {
		return fmt.Errorf("replace protected GitHub App session: %w", err)
	}
	return nil
}

func defaultDPAPISessionStorePath() (string, error) {
	directory, err := userConfigDirectory()
	if err != nil {
		return "", fmt.Errorf("resolve user configuration directory: %w", err)
	}
	return filepath.Join(directory, "git-governance", "github-app-sessions.dpapi"), nil
}

func emptySessionDocument() sessionDocument {
	return sessionDocument{
		SchemaVersion: sessionStoreSchemaVersion,
		ActiveByHost:  make(map[string]string),
		Sessions:      make(map[string]Session),
	}
}

func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

func contextFailure(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func protectDPAPI(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, errors.New("cannot protect an empty session")
	}
	entropy := []byte(dpapiEntropy)
	input := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	optionalEntropy := windows.DataBlob{Size: uint32(len(entropy)), Data: &entropy[0]}
	var output windows.DataBlob
	if err := cryptProtectData(
		&input,
		nil,
		&optionalEntropy,
		0,
		nil,
		windows.CRYPTPROTECT_UI_FORBIDDEN,
		&output,
	); err != nil {
		return nil, err
	}
	defer freeDPAPIBlob(&output)
	protected := append([]byte(nil), unsafe.Slice(output.Data, output.Size)...)
	runtime.KeepAlive(plain)
	runtime.KeepAlive(entropy)
	return protected, nil
}

func unprotectDPAPI(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, errors.New("protected GitHub App session is empty")
	}
	entropy := []byte(dpapiEntropy)
	input := windows.DataBlob{Size: uint32(len(encrypted)), Data: &encrypted[0]}
	optionalEntropy := windows.DataBlob{Size: uint32(len(entropy)), Data: &entropy[0]}
	var output windows.DataBlob
	if err := cryptUnprotectData(
		&input,
		nil,
		&optionalEntropy,
		0,
		nil,
		windows.CRYPTPROTECT_UI_FORBIDDEN,
		&output,
	); err != nil {
		return nil, err
	}
	defer freeDPAPIBlob(&output)
	plain := append([]byte(nil), unsafe.Slice(output.Data, output.Size)...)
	runtime.KeepAlive(encrypted)
	runtime.KeepAlive(entropy)
	return plain, nil
}

func freeDPAPIBlob(blob *windows.DataBlob) {
	if blob == nil || blob.Data == nil {
		return
	}
	_, _ = freeLocalMemory(windows.Handle(uintptr(unsafe.Pointer(blob.Data))))
	blob.Data = nil
	blob.Size = 0
}
