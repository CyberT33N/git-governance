package gitcli

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
)

const defaultMaxOutputBytes = 1 << 20

type processResult struct {
	stdout    string
	stderr    string
	exitCode  int
	err       error
	truncated bool
}

type processRunner interface {
	run(ctx context.Context, directory string, stdin io.Reader, arguments ...string) processResult
}

type environmentProcessRunner interface {
	runWithEnvironment(
		ctx context.Context,
		directory string,
		stdin io.Reader,
		environment []string,
		arguments ...string,
	) processResult
}

type execRunner struct {
	binary         string
	maxOutputBytes int
}

func (runner execRunner) run(ctx context.Context, directory string, stdin io.Reader, arguments ...string) processResult {
	return runner.runWithEnvironment(ctx, directory, stdin, nil, arguments...)
}

func (runner execRunner) runWithEnvironment(
	ctx context.Context,
	directory string,
	stdin io.Reader,
	environment []string,
	arguments ...string,
) processResult {
	command := exec.CommandContext(ctx, runner.binary, arguments...)
	command.Dir = directory
	command.Stdin = stdin
	if len(environment) > 0 {
		command.Env = append(os.Environ(), environment...)
	}

	stdout := newBoundedBuffer(runner.maxOutputBytes)
	stderr := newBoundedBuffer(runner.maxOutputBytes)
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()
	exitCode := 0
	if err != nil {
		exitCode = -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
	}

	return processResult{
		stdout:    stdout.String(),
		stderr:    stderr.String(),
		exitCode:  exitCode,
		err:       err,
		truncated: stdout.Truncated() || stderr.Truncated(),
	}
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	if limit <= 0 {
		limit = defaultMaxOutputBytes
	}
	return &boundedBuffer{limit: limit}
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining <= 0 {
		buffer.truncated = true
		return len(value), nil
	}
	if len(value) > remaining {
		_, _ = buffer.buffer.Write(value[:remaining])
		buffer.truncated = true
		return len(value), nil
	}
	_, _ = buffer.buffer.Write(value)
	return len(value), nil
}

func (buffer *boundedBuffer) String() string {
	return buffer.buffer.String()
}

func (buffer *boundedBuffer) Truncated() bool {
	return buffer.truncated
}
