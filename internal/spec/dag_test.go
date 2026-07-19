package spec

import (
	"strings"
	"testing"
)

func task(id string, done bool, deps ...string) Task {
	return Task{ID: id, Title: id, Done: done, DependsOn: deps}
}

func TestValidateDAG(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []Task
		wantErr string
	}{
		{name: "empty list", tasks: nil},
		{name: "no dependencies", tasks: []Task{task("task-1", false), task("task-2", false)}},
		{name: "valid chain", tasks: []Task{task("task-1", false), task("task-2", false, "task-1"), task("task-3", false, "task-2")}},
		{name: "diamond", tasks: []Task{
			task("task-1", false),
			task("task-2", false, "task-1"),
			task("task-3", false, "task-1"),
			task("task-4", false, "task-2", "task-3"),
		}},
		{
			name:    "self dependency",
			tasks:   []Task{task("task-1", false, "task-1")},
			wantErr: "task task-1 cannot depend on itself",
		},
		{
			name:    "unknown dependency",
			tasks:   []Task{task("task-1", false, "task-9")},
			wantErr: "task task-1 depends on unknown task id: task-9",
		},
		{
			name: "two-node cycle",
			tasks: []Task{
				task("task-1", false, "task-2"),
				task("task-2", false, "task-1"),
			},
			wantErr: "dependency cycle detected",
		},
		{
			name: "three-node cycle",
			tasks: []Task{
				task("task-1", false, "task-3"),
				task("task-2", false, "task-1"),
				task("task-3", false, "task-2"),
			},
			wantErr: "dependency cycle detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDAG(tt.tasks)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateDAG() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateDAG() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateDAG() = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateDAG_CycleListsMembers(t *testing.T) {
	tasks := []Task{
		task("task-1", false, "task-3"),
		task("task-2", false, "task-1"),
		task("task-3", false, "task-2"),
	}
	err := ValidateDAG(tasks)
	if err == nil {
		t.Fatal("want cycle error, got nil")
	}
	for _, id := range []string{"task-1", "task-2", "task-3"} {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("cycle error %q missing member %s", err.Error(), id)
		}
	}
}

func TestReadyTaskIndices(t *testing.T) {
	tests := []struct {
		name  string
		tasks []Task
		want  []int
	}{
		{name: "empty", tasks: nil, want: nil},
		{
			name:  "all independent all ready",
			tasks: []Task{task("task-1", false), task("task-2", false), task("task-3", false)},
			want:  []int{0, 1, 2},
		},
		{
			name: "dependent not ready",
			tasks: []Task{
				task("task-1", false),
				task("task-2", false),
				task("task-3", false, "task-1", "task-2"),
			},
			want: []int{0, 1},
		},
		{
			name: "dependency done makes ready",
			tasks: []Task{
				task("task-1", true),
				task("task-2", true),
				task("task-3", false, "task-1", "task-2"),
			},
			want: []int{2},
		},
		{
			name: "partially done deps keep waiting",
			tasks: []Task{
				task("task-1", true),
				task("task-2", false),
				task("task-3", false, "task-1", "task-2"),
			},
			want: []int{1},
		},
		{
			name: "diamond middle stage",
			tasks: []Task{
				task("task-1", true),
				task("task-2", false, "task-1"),
				task("task-3", false, "task-1"),
				task("task-4", false, "task-2", "task-3"),
			},
			want: []int{1, 2},
		},
		{
			name: "blocked task excluded",
			tasks: []Task{
				{ID: "task-1", Blocked: true},
				task("task-2", false),
			},
			want: []int{1},
		},
		{
			name: "transitively blocked excluded",
			tasks: []Task{
				{ID: "task-1", Blocked: true},
				task("task-2", false, "task-1"),
				task("task-3", false, "task-2"),
				task("task-4", false),
			},
			want: []int{3},
		},
		{
			name: "done tasks excluded",
			tasks: []Task{
				task("task-1", true),
				task("task-2", true),
			},
			want: nil,
		},
		{
			name: "long chain sequential fallback",
			tasks: []Task{
				task("task-1", true),
				task("task-2", false, "task-1"),
				task("task-3", false, "task-2"),
				task("task-4", false, "task-3"),
			},
			want: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReadyTaskIndices(tt.tasks)
			if len(got) != len(tt.want) {
				t.Fatalf("ReadyTaskIndices() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ReadyTaskIndices() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBlockedSet(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Blocked: true},
		task("task-2", false, "task-1"),
		task("task-3", false, "task-2"),
		task("task-4", false),
		{ID: "task-5", Done: true, Blocked: true},
		task("task-6", false, "task-5"),
	}
	got := BlockedSet(tasks)
	for _, id := range []string{"task-1", "task-2", "task-3"} {
		if !got[id] {
			t.Errorf("BlockedSet missing %s", id)
		}
	}
	for _, id := range []string{"task-4", "task-5", "task-6"} {
		if got[id] {
			t.Errorf("BlockedSet should not contain %s", id)
		}
	}
}

func TestDependentsOf(t *testing.T) {
	tasks := []Task{
		task("task-1", false),
		task("task-2", false, "task-1"),
		task("task-3", false, "task-1", "task-2"),
	}
	got := DependentsOf(tasks, "task-1")
	if len(got) != 2 || got[0] != "task-2" || got[1] != "task-3" {
		t.Fatalf("DependentsOf(task-1) = %v, want [task-2 task-3]", got)
	}
	if deps := DependentsOf(tasks, "task-3"); len(deps) != 0 {
		t.Fatalf("DependentsOf(task-3) = %v, want empty", deps)
	}
}

func TestLintDependencies(t *testing.T) {
	tests := []struct {
		name       string
		tasks      []Task
		categories []string
	}{
		{name: "clean", tasks: []Task{task("task-1", false), task("task-2", false, "task-1")}, categories: nil},
		{name: "self", tasks: []Task{task("task-1", false, "task-1")}, categories: []string{"dep-self"}},
		{name: "unknown", tasks: []Task{task("task-1", false, "task-9")}, categories: []string{"dep-unknown"}},
		{
			name: "cycle",
			tasks: []Task{
				task("task-1", false, "task-2"),
				task("task-2", false, "task-1"),
			},
			categories: []string{"dep-cycle"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LintDependencies(tt.tasks)
			if len(got) != len(tt.categories) {
				t.Fatalf("LintDependencies() = %+v, want %d findings", got, len(tt.categories))
			}
			for i, cat := range tt.categories {
				if got[i].Category != cat {
					t.Errorf("finding %d category = %s, want %s", i, got[i].Category, cat)
				}
				if got[i].Severity != "block" {
					t.Errorf("finding %d severity = %s, want block", i, got[i].Severity)
				}
			}
		})
	}
}

func TestLintDependencies_NodePointingIntoCycle_NoSpuriousCycle(t *testing.T) {
	tasks := []Task{
		{ID: "task-a", DependsOn: []string{"task-b"}},
		{ID: "task-b", DependsOn: []string{"task-a"}},
		{ID: "task-c", DependsOn: []string{"task-a"}},
	}
	findings := LintDependencies(tasks)
	var cycles []Finding
	for _, f := range findings {
		if f.Category == "dep-cycle" {
			cycles = append(cycles, f)
		}
	}
	if len(cycles) != 1 {
		t.Fatalf("expected exactly one cycle finding, got %d: %+v", len(cycles), cycles)
	}
	if strings.Contains(cycles[0].Detail, "task-c") {
		t.Fatalf("task-c is not part of a cycle, got %q", cycles[0].Detail)
	}
}

func TestCollectDAGIssues_SharedNodeCycles_AllReported(t *testing.T) {
	tasks := []Task{
		task("task-1", false, "task-2"),
		task("task-2", false, "task-1", "task-3"),
		task("task-3", false, "task-2"),
	}
	issues := collectDAGIssues(tasks)
	var cycles []DAGError
	for _, issue := range issues {
		if issue.Kind == DAGErrorCycle {
			cycles = append(cycles, issue)
		}
	}
	if len(cycles) != 2 {
		t.Fatalf("expected both cycles reported in one pass, got %d: %+v", len(cycles), cycles)
	}
	joined := ""
	for _, c := range cycles {
		joined += strings.Join(c.Cycle, " -> ") + "; "
	}
	if !strings.Contains(joined, "task-3") {
		t.Fatalf("expected the task-2<->task-3 cycle to be reported, got %q", joined)
	}
}
