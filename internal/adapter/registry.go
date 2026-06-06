package adapter

import (
	"sort"
	"sync"

	"github.com/pragmataW/tddmaster/internal/manifest"
)

var (
	mu       sync.Mutex
	registry = make(map[manifest.ToolID]ToolAdapter)
)

func Register(a ToolAdapter) {
	mu.Lock()
	defer mu.Unlock()
	registry[a.ID()] = a
}

func Get(id manifest.ToolID) (ToolAdapter, bool) {
	mu.Lock()
	defer mu.Unlock()
	a, ok := registry[id]
	return a, ok
}

func AllIDs() []manifest.ToolID {
	mu.Lock()
	defer mu.Unlock()
	strs := make([]string, 0, len(registry))
	for id := range registry {
		strs = append(strs, string(id))
	}
	sort.Strings(strs)
	ids := make([]manifest.ToolID, len(strs))
	for i, s := range strs {
		ids[i] = manifest.ToolID(s)
	}
	return ids
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[manifest.ToolID]ToolAdapter)
}
