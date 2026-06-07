package loop

type RuleSet struct {
	stages []Stage
}

func newRuleSetWith(stages []Stage) *RuleSet {
	return &RuleSet{stages: stages}
}

func newRuleSet() *RuleSet {
	return newRuleSetWith(allStages())
}

func (rs *RuleSet) Next(ctx ExecCtx) (Stage, bool) {
	for _, s := range rs.stages {
		if s.Applies(ctx) {
			return s, true
		}
	}
	return nil, false
}
