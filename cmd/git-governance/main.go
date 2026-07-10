package main

import (
	"context"
	"os"

	"github.com/CyberT33N/git-governance/internal/bootstrap"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/spf13/cobra"
)

var (
	version      = "devel"
	commit       = "unknown"
	date         = "unknown"
	exitProcess  = os.Exit
	buildCommand = newCommand
)

func main() {
	exitProcess(execute(context.Background(), buildCommand()))
}

func execute(ctx context.Context, command *cobra.Command) int {
	if err := command.ExecuteContext(ctx); err != nil {
		bootstrap.RenderError(command, err)
		return problem.ExitCode(err)
	}
	return problem.ExitSuccess
}

func newCommand() *cobra.Command {
	return bootstrap.New(bootstrap.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
}
