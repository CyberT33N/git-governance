package main

import (
	"context"
	"os"

	"github.com/CyberT33N/git-governance/internal/bootstrap"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/spf13/cobra"
)

var (
	version = "devel"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	command := newCommand()

	if err := command.ExecuteContext(context.Background()); err != nil {
		bootstrap.RenderError(command, err)
		os.Exit(problem.ExitCode(err))
	}
}

func newCommand() *cobra.Command {
	return bootstrap.New(bootstrap.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
}
