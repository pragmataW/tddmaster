package promptregistry

type InstructionKey string

type AgentRegistryKey string

const (
	AgentExecutor   AgentRegistryKey = "tddmaster-executor"
	AgentVerifier   AgentRegistryKey = "tddmaster-verifier"
	AgentPlanner    AgentRegistryKey = "tddmaster-planner"
	AgentTestWriter AgentRegistryKey = "tddmaster-test-writer"
)

var AllAgentKeys = []AgentRegistryKey{
	AgentExecutor,
	AgentVerifier,
	AgentPlanner,
	AgentTestWriter,
}

const (
	KeySettings         InstructionKey = "spec-settings:configure"
	KeyListenFirst      InstructionKey = "discovery:listen-first"
	KeyModeSelection    InstructionKey = "discovery:mode-selection"
	KeyPremiseChallenge InstructionKey = "discovery:premise-challenge"
	KeySpecTaskGen      InstructionKey = "spec-proposal:task-gen"
	KeySelfReview       InstructionKey = "spec-proposal:self-review"
	KeyRefinePrompt     InstructionKey = "refinement:refine-prompt"
)

const (
	KeyExecRed          InstructionKey = "execution:red"
	KeyExecGreen        InstructionKey = "execution:green"
	KeyExecRefactor     InstructionKey = "execution:refactor"
	KeyExecRefactorApply InstructionKey = "execution:refactor-apply"
	KeyExecRefactorSkipVerify InstructionKey = "execution:refactor-skip-verify"
	KeyExecExecutor     InstructionKey = "execution:executor"
	KeyExecExecutorSkipVerify InstructionKey = "execution:executor-skip-verify"
	KeyExecVerifier     InstructionKey = "execution:verifier"
	KeyExecGate         InstructionKey = "execution:gate"
	KeyExecVerifyFailed InstructionKey = "execution:verify-failed"
)

func KeyDiscoveryQuestion(id string) InstructionKey {
	return InstructionKey("discovery:question:" + id)
}
