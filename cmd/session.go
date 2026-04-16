
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pragmataW/tddmaster/internal/output"
	"github.com/pragmataW/tddmaster/internal/state"
)

func newSessionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
		Long:  "Manage tddmaster sessions (start, stop, list, clean).",
		RunE:  runSession,
	}
}

func runSession(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		printErr(fmt.Sprintf("Usage: %s session <start | stop | list | clean>", output.CmdPrefix()))
		return nil
	}

	root, err := resolveRoot()
	if err != nil {
		return err
	}

	switch args[0] {
	case "start":
		return sessionStart(root, args[1:])
	case "stop":
		return sessionStop(root, args[1:])
	case "list":
		return sessionList(root)
	case "clean":
		return sessionClean(root)
	default:
		printErr(fmt.Sprintf("Usage: %s session <start | stop | list | clean>", output.CmdPrefix()))
		return nil
	}
}

func sessionStart(root string, args []string) error {
	var specName string
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			specName = a
		}
	}

	sessionID, err := state.GenerateSessionId()
	if err != nil {
		return err
	}

	var specPtr *string
	if specName != "" {
		specPtr = &specName
	}

	tool := "unknown"
	if t := os.Getenv("AGENT_TOOL"); t != "" {
		tool = t
	}

	session := state.Session{
		ID:           sessionID,
		Spec:         specPtr,
		Mode:         "spec",
		PID:          os.Getpid(),
		StartedAt:    "",
		LastActiveAt: "",
		Tool:         tool,
	}

	if err := state.CreateSession(root, session); err != nil {
		return err
	}

	fmt.Printf("TDDMASTER_SESSION=%s\n", sessionID)
	printErr(fmt.Sprintf("Session started: %s", sessionID))
	return nil
}

func sessionStop(root string, args []string) error {
	sessionID := os.Getenv("TDDMASTER_SESSION")
	if len(args) > 0 {
		sessionID = args[0]
	}
	if sessionID == "" {
		return fmt.Errorf("no session ID provided and TDDMASTER_SESSION not set")
	}

	ok, err := state.DeleteSession(root, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		printErr(fmt.Sprintf("Session %s not found.", sessionID))
	} else {
		printErr(fmt.Sprintf("Session %s stopped.", sessionID))
	}
	return nil
}

func sessionList(root string) error {
	sessions, err := state.ListSessions(root)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		printErr("No active sessions.")
		return nil
	}

	printErr("Sessions:")
	for _, s := range sessions {
		stale := ""
		if state.IsSessionStale(s) {
			stale = " (stale)"
		}
		spec := "none"
		if s.Spec != nil {
			spec = *s.Spec
		}
		phase := "unknown"
		if s.Phase != nil {
			phase = *s.Phase
		}
		printErr(fmt.Sprintf("  %s  spec=%s  phase=%s  tool=%s%s", s.ID, spec, phase, s.Tool, stale))
	}
	return nil
}

func sessionClean(root string) error {
	removed, err := state.GcStaleSessions(root)
	if err != nil {
		return err
	}

	if len(removed) == 0 {
		printErr("No stale sessions to clean.")
	} else {
		printErr(fmt.Sprintf("Cleaned %d stale session(s).", len(removed)))
	}
	return nil
}
