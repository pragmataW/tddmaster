package adapter

import (
	"sync"
	"testing"

	"github.com/pragmataW/tddmaster/internal/manifest"
)

type fakeAdapter struct {
	id      manifest.ToolID
	synced  bool
	lastCtx SyncContext
}

func (f *fakeAdapter) ID() manifest.ToolID {
	return f.id
}

func (f *fakeAdapter) Sync(ctx SyncContext) error {
	f.synced = true
	f.lastCtx = ctx
	return nil
}

func TestGet_EmptyRegistryReturnsFalse(t *testing.T) {
	Reset()

	got, ok := Get("nonexistent")
	if ok {
		t.Fatal("expected ok=false for unknown ID in empty registry")
	}
	if got != nil {
		t.Fatal("expected nil adapter for unknown ID in empty registry")
	}
}

func TestRegister_GetReturnsAdapter(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	a := &fakeAdapter{id: "fake"}
	Register(a)

	got, ok := Get("fake")
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if got.ID() != "fake" {
		t.Fatalf("expected ID fake, got %s", got.ID())
	}
}

func TestAllIDs_ReturnsAllRegisteredIDs(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	Register(&fakeAdapter{id: "alpha"})
	Register(&fakeAdapter{id: "beta"})

	ids := AllIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}

	set := make(map[manifest.ToolID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}

	for _, expected := range []manifest.ToolID{"alpha", "beta"} {
		if !set[expected] {
			t.Errorf("AllIDs missing expected ID %q", expected)
		}
	}
}

func TestAllIDs_SortedOrDeterministic(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	Register(&fakeAdapter{id: "zulu"})
	Register(&fakeAdapter{id: "alpha"})
	Register(&fakeAdapter{id: "mike"})

	first := AllIDs()
	second := AllIDs()

	if len(first) != len(second) {
		t.Fatalf("AllIDs not deterministic: got %d then %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("AllIDs not deterministic at index %d: %q vs %q", i, first[i], second[i])
		}
	}
}

func TestRegister_OverwritesSameID(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	first := &fakeAdapter{id: "dup"}
	second := &fakeAdapter{id: "dup"}

	Register(first)
	Register(second)

	got, ok := Get("dup")
	if !ok {
		t.Fatal("expected ok=true after overwrite")
	}
	if got != second {
		t.Fatal("expected second adapter (last-write-wins), got first or unexpected adapter")
	}

	ids := AllIDs()
	count := 0
	for _, id := range ids {
		if id == "dup" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 entry for duplicate ID, got %d", count)
	}
}

func TestReset_ClearsRegistry(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	Register(&fakeAdapter{id: "to-be-cleared"})
	Reset()

	got, ok := Get("to-be-cleared")
	if ok {
		t.Fatal("expected ok=false after Reset")
	}
	if got != nil {
		t.Fatal("expected nil after Reset")
	}
	if len(AllIDs()) != 0 {
		t.Fatalf("expected empty AllIDs after Reset, got %d entries", len(AllIDs()))
	}
}

func TestRegister_Concurrent(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		id := manifest.ToolID("concurrent-" + string(rune('0'+i)))
		go func(toolID manifest.ToolID) {
			defer wg.Done()
			Register(&fakeAdapter{id: toolID})
		}(id)
	}

	wg.Wait()

	ids := AllIDs()
	if len(ids) != n {
		t.Fatalf("expected %d registered adapters after concurrent Register, got %d", n, len(ids))
	}

	set := make(map[manifest.ToolID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}

	for i := 0; i < n; i++ {
		expected := manifest.ToolID("concurrent-" + string(rune('0'+i)))
		if !set[expected] {
			t.Errorf("AllIDs missing concurrently registered ID %q", expected)
		}
	}
}

func TestSyncContext_HasCommandPrefix(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	m := &manifest.Manifest{Command: "tddmaster"}
	ctx := SyncContext{Root: "/tmp/x", Manifest: m, CommandPrefix: "tddmaster"}

	if ctx.Root != "/tmp/x" {
		t.Fatalf("expected Root /tmp/x, got %s", ctx.Root)
	}
	if ctx.CommandPrefix != "tddmaster" {
		t.Fatalf("expected CommandPrefix tddmaster, got %s", ctx.CommandPrefix)
	}
	if ctx.Manifest.Command != "tddmaster" {
		t.Fatalf("expected Manifest.Command tddmaster, got %s", ctx.Manifest.Command)
	}
}

func TestSync_Called(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	a := &fakeAdapter{id: "syncable"}
	Register(a)

	got, ok := Get("syncable")
	if !ok {
		t.Fatal("expected ok=true")
	}

	ctx := SyncContext{Root: "/home/tdd", Manifest: &manifest.Manifest{Command: "tddmaster"}, CommandPrefix: "tddmaster"}
	if err := got.Sync(ctx); err != nil {
		t.Fatalf("unexpected error from Sync: %v", err)
	}

	if !a.synced {
		t.Fatal("expected synced=true after Sync call")
	}
	if a.lastCtx.Root != "/home/tdd" {
		t.Fatalf("expected lastCtx.Root /home/tdd, got %s", a.lastCtx.Root)
	}
}
