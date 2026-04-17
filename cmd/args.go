package cmd

import (
	"fmt"
	"strings"
)

// nonFlagArgs returns positional args after removing CLI flags like --spec=foo.
func nonFlagArgs(args []string) []string {
	var positional []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		positional = append(positional, a)
	}
	return positional
}

// stripExactArg removes an exact flag token from args and reports whether it was found.
func stripExactArg(args []string, target string) ([]string, bool) {
	found := false
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == target {
			found = true
			continue
		}
		out = append(out, a)
	}
	return out, found
}

func rejectPositionalArgs(command string, args []string, hint string) error {
	positional := nonFlagArgs(args)
	if len(positional) == 0 {
		return nil
	}
	return fmt.Errorf("unexpected positional arguments for `%s`: %s. %s",
		command, strings.Join(positional, ", "), hint)
}
