package loop

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
)

func appendCoverageFeedback(b *strings.Builder, ctx ExecCtx) {
	if !coverageEnforced(ctx.Settings) {
		return
	}
	if len(ctx.State.LastCoverage) == 0 {
		return
	}
	threshold := float64(ctx.Settings.MinTestCoverage)
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
	b.WriteString(promptregistry.CoverageLowFeedbackHeader)
	for _, file := range lowFiles {
		pct := ctx.State.LastCoverage[file]
		b.WriteString(fmt.Sprintf("- %s: %.1f%% < %.0f%%\n", file, pct, threshold))
	}
	b.WriteString(promptregistry.CoverageLowFeedbackFooter)
}
