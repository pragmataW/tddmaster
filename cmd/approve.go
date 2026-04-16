
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	specp "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Approve the current state",
		RunE:  runApprove,
	}
	cmd.Flags().String("spec", "", "Spec name")
	return cmd
}

func runApprove(cmd *cobra.Command, args []string) error {
	specFlag, _ := cmd.Flags().GetString("spec")
	specArgs := args
	if specFlag != "" {
		specArgs = append([]string{"--spec=" + specFlag}, args...)
	}
	return runApproveWithArgs(specArgs)
}

func runApproveWithArgs(args []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	specResult := state.RequireSpecFlag(args)
	if !specResult.OK {
		return fmt.Errorf("%s", specResult.Error)
	}

	st, err := state.ResolveState(root, &specResult.Spec)
	if err != nil {
		return fmt.Errorf("%s", err.Error())
	}

	config, _ := state.ReadManifest(root)

	// Spec directory check
	if st.Spec != nil && !specDirExists(root, *st.Spec) {
		return fmt.Errorf("active spec '%s' directory not found. Run `%s` to return to idle", *st.Spec, output.Cmd("reset"))
	}

	// Delegation gate
	pendingDelegations := state.GetPendingDelegations(st)
	if len(pendingDelegations) > 0 &&
		(st.Phase == state.PhaseSpecProposal || st.Phase == state.PhaseDiscoveryRefinement) {
		return fmt.Errorf("cannot approve — %d pending delegation(s). All delegations must be answered before approval",
			len(pendingDelegations))
	}

	user, _ := state.ResolveUser(root)
	userInfo := &state.UserInfo{Name: user.Name, Email: user.Email}

	if st.Phase == state.PhaseSpecProposal {
		// Compute state transition first; derivative writes (spec.md, status
		// fields) happen after the primary state is durable on disk.
		newState, err := state.ApproveSpec(st)
		if err != nil {
			return err
		}
		reasonStr := "approved"
		newState = state.RecordTransition(newState, state.PhaseSpecProposal, state.PhaseSpecApproved, userInfo, &reasonStr)

		if err := state.WriteStateAndSpec(root, newState); err != nil {
			return err
		}

		// Post-commit: regenerate spec.md if needed, then update status fields.
		if newState.Spec != nil {
			if st.Classification == nil {
				allConcerns, _ := state.ListConcerns(root)
				activeConcerns := filterConcerns(allConcerns, config.Concerns)
				if _, genErr := specp.GenerateSpec(root, &newState, activeConcerns); genErr != nil {
					printErr(fmt.Sprintf("warning: approve succeeded but spec.md generation failed: %v", genErr))
				}
			}
			if err := specp.UpdateSpecStatus(root, *newState.Spec, "approved"); err != nil {
				printErr(fmt.Sprintf("warning: spec.md status update failed: %v", err))
			}
			if err := specp.UpdateProgressStatus(root, *newState.Spec, "approved"); err != nil {
				printErr(fmt.Sprintf("warning: progress.json status update failed: %v", err))
			}
		}

		printErr("Spec approved. Phase: SPEC_APPROVED")
		printErr(fmt.Sprintf("When ready, run %s to begin execution.", output.Cmd(`next --answer="start"`)))

	} else if st.Phase == state.PhaseDiscoveryRefinement {
		newState, err := state.ApproveDiscoveryReview(st)
		if err != nil {
			return err
		}
		reasonStr := "approved"
		newState = state.RecordTransition(newState, state.PhaseDiscoveryRefinement, state.PhaseSpecProposal, userInfo, &reasonStr)

		if err := state.WriteStateAndSpec(root, newState); err != nil {
			return err
		}

		printErr("Discovery answers approved. Phase: SPEC_PROPOSAL")
		printErr(fmt.Sprintf("Review the spec and run %s again to approve.", output.Cmd("approve")))

	} else {
		return fmt.Errorf("cannot approve in phase: %s", st.Phase)
	}

	_ = strings.TrimSpace // avoid unused import
	return nil
}
