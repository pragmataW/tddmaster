
package adapters

import (
	statesync "github.com/pragmataW/tddmaster/internal/sync"
)

func init() {
	statesync.RegisterAdapter(&ClaudeCodeAdapter{})
	statesync.RegisterAdapter(&OpenCodeAdapter{})
	statesync.RegisterAdapter(&CodexAdapter{})
}
