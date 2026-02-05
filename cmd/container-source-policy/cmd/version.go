package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/wharflab/container-source-policy/internal/version"
)

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print the version information",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Printf("container-source-policy %s\n", version.Version())
			return nil
		},
	}
}
