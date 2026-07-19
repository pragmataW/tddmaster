package loop

import (
	"fmt"
	"strings"
)

func appendCoverageRequirement(b *strings.Builder, ctx ExecCtx) {
	if !coverageEnforced(ctx.Settings) {
		return
	}
	if ctx.State.CoverageUnreported {
		b.WriteString("\nThe previous verification reported no coverage measurements. " +
			"You MUST run the coverage tool now and return a non-empty " + fileCoverageReportShape + ". " +
			"An empty report blocks the cycle and will be rejected.\n")
	}
	var files []string
	if ctx.State.Plan != nil {
		files = ctx.State.Plan.TouchedFiles
	}
	if len(files) == 0 {
		files = ctx.State.LastModifiedFiles
	}

	b.WriteString(fmt.Sprintf(
		"\nCoverage requirement: measure test coverage for each touched file using the project's language-appropriate coverage tool. "+
			"Each file must reach %d%% coverage. "+
			"Report results as "+fileCoverageReportShape+". "+
			"For each file below the threshold, propose new tests.\n"+
			"Coverage measurement is performed exclusively by you, the verifier sub-agent. "+
			"The orchestrator must delegate this to you and must not run any coverage tooling itself.\n",
		ctx.Settings.MinTestCoverage,
	))
	b.WriteString("Touched files to measure:\n")
	for _, f := range files {
		b.WriteString("- ")
		b.WriteString(f)
		b.WriteString("\n")
	}
}
