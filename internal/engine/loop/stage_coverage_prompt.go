package loop

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
)

func appendCoverageRequirement(b *strings.Builder, ctx ExecCtx) {
	if !coverageEnforced(ctx.Settings) {
		return
	}
	if ctx.State.CoverageUnreported {
		b.WriteString(promptregistry.CoverageUnreportedText)
	}
	var files []string
	if ctx.State.Plan != nil {
		files = ctx.State.Plan.TouchedFiles
	}
	if len(files) == 0 {
		files = ctx.State.LastModifiedFiles
	}

	b.WriteString(fmt.Sprintf(promptregistry.CoverageRequirementFmt, ctx.Settings.MinTestCoverage))
	b.WriteString("Touched files to measure:\n")
	for _, f := range files {
		b.WriteString("- ")
		b.WriteString(f)
		b.WriteString("\n")
	}
}
