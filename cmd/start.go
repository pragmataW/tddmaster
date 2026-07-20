package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/spf13/cobra"
)

func resolveRoot(cmd *cobra.Command) (string, error) {
	r, _ := cmd.Flags().GetString("root")
	if r != "" {
		return r, nil
	}
	return os.Getwd()
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <slug>",
		Short: "Start a new spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			root, err := resolveRoot(cmd)
			if err != nil {
				return errs.Wrap(errs.KeyResolveRoot, err)
			}
			res, err := spec.Start(root, slug, time.Now().UTC())
			if err != nil {
				return errs.Wrap(errs.KeyStartSpec, err)
			}
			out := cmd.OutOrStdout()
			if res.AlreadyExists {
				fmt.Fprintf(out, "spec %s already exists\n", slug)
				return nil
			}
			fmt.Fprintf(out, "started spec %s\n", slug)
			for _, f := range res.FilesWritten {
				fmt.Fprintf(out, "  %s\n", f)
			}
			fmt.Fprintf(out, "run: tddmaster next %s\n", slug)
			return nil
		},
	}
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
	return cmd
}
