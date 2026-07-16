package browser

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSystemOpenerValidatesURLsAndUsesNativeCommands(t *testing.T) {
	defaultOpener := New(Options{})
	if defaultOpener == nil || defaultOpener.operatingSystem == "" || defaultOpener.run == nil {
		t.Fatalf("default browser opener = %#v", defaultOpener)
	}

	for _, testCase := range []struct {
		operatingSystem string
		executable      string
		arguments       []string
	}{
		{
			operatingSystem: "windows",
			executable:      "rundll32.exe",
			arguments:       []string{"url.dll,FileProtocolHandler", "https://github.com/login/device"},
		},
		{
			operatingSystem: "darwin",
			executable:      "open",
			arguments:       []string{"https://github.com/login/device"},
		},
		{
			operatingSystem: "linux",
			executable:      "xdg-open",
			arguments:       []string{"https://github.com/login/device"},
		},
	} {
		testCase := testCase
		t.Run(testCase.operatingSystem, func(t *testing.T) {
			var gotExecutable string
			var gotArguments []string
			opener := New(Options{
				OperatingSystem: testCase.operatingSystem,
				Run: func(_ context.Context, executable string, arguments ...string) error {
					gotExecutable = executable
					gotArguments = append([]string(nil), arguments...)
					return nil
				},
			})
			if err := opener.Open(context.Background(), "https://github.com/login/device"); err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			if gotExecutable != testCase.executable || strings.Join(gotArguments, ",") != strings.Join(testCase.arguments, ",") {
				t.Fatalf("browser command = %q %#v", gotExecutable, gotArguments)
			}
		})
	}

	var nilOpener *SystemOpener
	if err := nilOpener.Open(context.Background(), "https://github.com"); err == nil {
		t.Fatal("nil opener unexpectedly succeeded")
	}
	for _, rawURL := range []string{"", "http://github.com", "https://user@github.com", "https:///missing-host"} {
		if err := New(Options{Run: func(context.Context, string, ...string) error { return nil }}).Open(context.Background(), rawURL); err == nil {
			t.Fatalf("Open(%q) unexpectedly succeeded", rawURL)
		}
	}
	if err := New(Options{
		OperatingSystem: "plan9",
		Run:             func(context.Context, string, ...string) error { return nil },
	}).Open(context.Background(), "https://github.com"); err == nil {
		t.Fatal("unsupported platform unexpectedly opened a browser")
	}
	expected := errors.New("browser unavailable")
	if err := New(Options{
		OperatingSystem: "linux",
		Run: func(context.Context, string, ...string) error {
			return expected
		},
	}).Open(context.Background(), "https://github.com"); !errors.Is(err, expected) {
		t.Fatalf("Open() error = %v, want %v", err, expected)
	}
}

func TestRunCommandExecutesAndReportsFailures(t *testing.T) {
	if err := runCommand(context.Background(), "git", "--version"); err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}
	if err := runCommand(context.Background(), "git-governance-browser-command-that-does-not-exist"); err == nil {
		t.Fatal("runCommand unexpectedly started a missing executable")
	}
}
