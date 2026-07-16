// Package browser opens an explicit user-initiated HTTPS URL in the native
// desktop browser. It is used only by authentication commands, never by
// background publication flows.
package browser

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// Opener is the outbound platform boundary for an explicit browser launch.
type Opener interface {
	Open(context.Context, string) error
}

// Options provides test seams for native browser launching.
type Options struct {
	OperatingSystem string
	Run             func(context.Context, string, ...string) error
}

// SystemOpener starts the platform-native browser opener.
type SystemOpener struct {
	operatingSystem string
	run             func(context.Context, string, ...string) error
}

// New creates a browser opener with safe HTTPS-only URL validation.
func New(options Options) *SystemOpener {
	operatingSystem := options.OperatingSystem
	if operatingSystem == "" {
		operatingSystem = runtime.GOOS
	}
	run := options.Run
	if run == nil {
		run = runCommand
	}
	return &SystemOpener{
		operatingSystem: operatingSystem,
		run:             run,
	}
}

// Open launches the given HTTPS URL through the OS shell integration.
func (opener *SystemOpener) Open(ctx context.Context, rawURL string) error {
	if opener == nil || opener.run == nil {
		return errors.New("browser opener is not configured")
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return errors.New("browser URL must be an HTTPS URL without credentials")
	}
	switch opener.operatingSystem {
	case "windows":
		return opener.run(ctx, "rundll32.exe", "url.dll,FileProtocolHandler", parsed.String())
	case "darwin":
		return opener.run(ctx, "open", parsed.String())
	case "linux":
		return opener.run(ctx, "xdg-open", parsed.String())
	default:
		return errors.New("browser opening is not supported on this platform")
	}
}

func runCommand(ctx context.Context, executable string, arguments ...string) error {
	return exec.CommandContext(ctx, executable, arguments...).Run()
}

var _ Opener = (*SystemOpener)(nil)
