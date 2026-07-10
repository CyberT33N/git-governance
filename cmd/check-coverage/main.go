// Command check-coverage enforces complete Go statement coverage.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type commandRunner func(context.Context, string, ...string) ([]byte, error)

var (
	exitProcess = os.Exit
	commandArgs = os.Args
	runCommand  = runGoCommand
)

func main() {
	exitProcess(run(context.Background(), commandArgs[1:], os.Stdout, os.Stderr, runCommand))
}

func run(
	ctx context.Context,
	arguments []string,
	stdout io.Writer,
	stderr io.Writer,
	execute commandRunner,
) int {
	if len(arguments) != 0 {
		fmt.Fprintln(stderr, "usage: check-coverage")
		return 2
	}
	if ctx == nil {
		ctx = context.Background()
	}

	output, err := execute(ctx, "go", "test", "-cover", "./...")
	if len(output) > 0 {
		_, _ = stdout.Write(output)
	}
	if err != nil {
		fmt.Fprintln(stderr, "run Go coverage:", err)
		return 1
	}

	incomplete := incompletePackages(string(output))
	if len(incomplete) > 0 {
		fmt.Fprintln(stderr, "every executable Go package must reach 100.0% statement coverage:")
		for _, line := range incomplete {
			fmt.Fprintln(stderr, line)
		}
		return 1
	}

	fmt.Fprintln(stdout, "All executable Go packages reached 100.0% statement coverage.")
	return 0
}

func runGoCommand(ctx context.Context, executable string, arguments ...string) ([]byte, error) {
	return exec.CommandContext(ctx, executable, arguments...).CombinedOutput()
}

func incompletePackages(output string) []string {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	incomplete := make([]string, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		for index, field := range fields {
			if field != "coverage:" || index+1 >= len(fields) {
				continue
			}
			if fields[index+1] != "100.0%" && fields[index+1] != "[no" {
				incomplete = append(incomplete, line)
			}
		}
	}
	return incomplete
}
