package loop

import (
	"fmt"
	"sort"
	"strings"
)

func appendCoverageFeedback(b *strings.Builder, ctx ExecCtx) {
	if !coverageEnforced(ctx.Settings) {
		return
	}
	if len(ctx.State.LastCoverage) == 0 {
		return
	}
	threshold := ctx.Settings.MinTestCoverage
	var lowFiles []string
	for file, pct := range ctx.State.LastCoverage {
		if pct < threshold {
			lowFiles = append(lowFiles, file)
		}
	}
	if len(lowFiles) == 0 {
		return
	}
	sort.Strings(lowFiles)
	b.WriteString("\nThe following files have low test coverage and need additional tests:\n")
	for _, file := range lowFiles {
		pct := ctx.State.LastCoverage[file]
		b.WriteString(fmt.Sprintf("- %s: %d%% < %d%%\n", file, pct, threshold))
	}
	b.WriteString("Add tests to bring these files above the coverage threshold.\n")
}
