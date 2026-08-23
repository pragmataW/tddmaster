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
	var files []string
	if ctx.State.Plan != nil {
		files = measurableFiles(ctx.State.Plan.TouchedFiles)
	}
	if len(files) == 0 {
		files = measurableFiles(implementationFiles(ctx.State))
	}

	section(b, promptregistry.SectionCoverage)
	if ctx.State.CoverageUnreported {
		b.WriteString(promptregistry.CoverageUnreportedText)
	}
	fmt.Fprintf(b, promptregistry.CoverageRequirementFmt, ctx.Settings.MinTestCoverage)
	b.WriteString("Touched files to measure:\n")
	writeList(b, files)
}
