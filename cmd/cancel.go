package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/pragmataW/tddmaster/internal/lifecycle"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
	"github.com/spf13/cobra"
)

func readerIsTTY(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

var cancelConfirm = func(slug string, in io.Reader, out io.Writer) (bool, error) {
	if !readerIsTTY(in) {
		return false, fmt.Errorf("no TTY detected: pass --force to skip confirmation")
	}

	var typed string
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Type %q to confirm cancellation", slug)).
				Description("This will permanently delete the spec directory.").
				Value(&typed),
		),
	).WithTheme(theme.Theme()).WithInput(in).WithOutput(out)

	if err := confirmForm.Run(); err != nil {
		return false, err
	}

	return typed == slug, nil
}

func newCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <slug>",
		Short: "Cancel a spec and delete its directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			force, _ := cmd.Flags().GetBool("force")

			root, err := resolveRoot(cmd)
			if err != nil {
				return fmt.Errorf("resolve root: %w", err)
			}

			if !spec.ValidSlug(slug) {
				return fmt.Errorf("invalid slug %q", slug)
			}
			if !spec.Exists(root, slug) {
				return fmt.Errorf("spec %q does not exist", slug)
			}

			out := cmd.OutOrStdout()

			if !force {
				confirmed, err := cancelConfirm(slug, cmd.InOrStdin(), out)
				if err != nil {
					if errors.Is(err, huh.ErrUserAborted) {
						fmt.Fprintln(out, "cancel aborted.")
						return nil
					}
					return err
				}

				if !confirmed {
					fmt.Fprintln(out, "cancel aborted: slug confirmation did not match.")
					return nil
				}
			}

			if err := lifecycle.Cancel(root, slug); err != nil {
				return fmt.Errorf("cancel spec: %w", err)
			}

			fmt.Fprintf(out, "cancelled spec %s\n", slug)
			return nil
		},
	}
	cmd.Flags().String("root", "", "Root directory (default: cwd)")
	cmd.Flags().Bool("force", false, "Skip confirmation and delete immediately")
	return cmd
}
