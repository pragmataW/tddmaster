package loop

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
)

func appendCoverageFeedback(b *strings.Builder, ctx ExecCtx) {
	lowFiles := lowCoverageStateFiles(ctx.State, ctx.Settings)
	if len(lowFiles) == 0 {
		return
	}
	threshold := float64(ctx.Settings.MinTestCoverage)
	section(b, promptregistry.SectionCoverage)
	b.WriteString(promptregistry.CoverageLowFeedbackHeader)
	for _, file := range lowFiles {
		pct := ctx.State.LastCoverage[file]
		fmt.Fprintf(b, "- %s: %.1f%% < %.0f%%\n", file, pct, threshold)
	}
	b.WriteString(promptregistry.CoverageLowFeedbackFooter)
}
