package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithoutSessionsRequiresNamedCreation(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	status := st.SessionStatus()
	if !status.NeedsCreation || status.ActiveID != "" || len(status.Sessions) != 0 {
		t.Fatalf("unexpected empty-session status: %+v", status)
	}
	if _, err := st.CreateSession("   "); err == nil {
		t.Fatal("blank session title should be rejected")
	}
}

func TestSaveLoadNamedSession(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir, NewAgentState())
	info, err := st.CreateSession("Test session")
	if err != nil {
		t.Fatal(err)
	}
	st.AddMessage(Message{ID: st.NextID(), Source: MessageUser, Text: "hello"})
	if !st.PersistenceStatus().Dirty {
		t.Fatal("state mutation should mark session dirty")
	}
	if err := st.Save("manual"); err != nil {
		t.Fatal(err)
	}
	if st.PersistenceStatus().Dirty {
		t.Fatal("successful save should clear dirty state")
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Snapshot().Messages; len(got) != 1 || got[0].Text != "hello" {
		t.Fatalf("unexpected: %#v", got)
	}
	status := loaded.SessionStatus()
	if status.ActiveID != info.ID || status.ActiveTitle != "Test session" || status.NeedsCreation {
		t.Fatalf("unexpected session status: %+v", status)
	}
	if _, err := os.Stat(filepath.Join(SessionsDir(dir), info.ID+".json")); err != nil {
		t.Fatalf("expected session state file: %v", err)
	}
}

func TestSwitchSessionAutosavesDirtyCurrent(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir, NewAgentState())
	first, err := st.CreateSession("First")
	if err != nil {
		t.Fatal(err)
	}
	st.AddMessage(Message{ID: st.NextID(), Source: MessageUser, Text: "persist me"})
	second, err := st.CreateSession("Second")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatal("session ids must be unique")
	}
	// Make the second session dirty, then switch back; SwitchSession must save it.
	st.AddMessage(Message{ID: st.NextID(), Source: MessageUser, Text: "second dirty"})
	saved, info, err := st.SwitchSession(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !saved || info.ID != first.ID {
		t.Fatalf("unexpected switch result: saved=%v info=%+v", saved, info)
	}
	if got := st.Snapshot().Messages; len(got) != 1 || got[0].Text != "persist me" {
		t.Fatalf("first session was not restored: %+v", got)
	}
	if _, _, err := st.SwitchSession(second.ID); err != nil {
		t.Fatal(err)
	}
	if got := st.Snapshot().Messages; len(got) != 1 || got[0].Text != "second dirty" {
		t.Fatalf("dirty second session was not autosaved: %+v", got)
	}
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.SessionStatus().ActiveID; got != second.ID {
		t.Fatalf("last active session was not restored: got %q want %q", got, second.ID)
	}
}

func TestFailActiveTaskPreservesRecoverableState(t *testing.T) {
	st := NewStore(t.TempDir(), NewAgentState())
	step1 := PlanStep{ID: 10, Status: StepCompleted}
	step2 := PlanStep{ID: 11, Status: StepRunning}
	st.SetActiveTask(&TaskState{
		ID:     1,
		Goal:   "finish implementation",
		Status: TaskRunning,
		Plan:   Plan{Version: 1, Steps: []PlanStep{step1, step2}},
		StepResults: []StepResult{
			{StepID: 10, Success: true, Summary: "first step done"},
		},
	})

	st.FailActiveTask("structured response invalid")
	snap := st.Snapshot()
	if snap.ActiveTask == nil {
		t.Fatal("failed task should remain active for recovery")
	}
	if snap.ActiveTask.Status != TaskFailed || snap.ActiveTask.Failure == nil || !snap.ActiveTask.Failure.Recoverable {
		t.Fatalf("unexpected failure state: %+v", snap.ActiveTask)
	}
	if snap.ActiveTask.Failure.FailedStepID == nil || *snap.ActiveTask.Failure.FailedStepID != 11 {
		t.Fatalf("failed step not recorded: %+v", snap.ActiveTask.Failure)
	}
	if snap.ActiveTask.Failure.LastSuccessfulStepID == nil || *snap.ActiveTask.Failure.LastSuccessfulStepID != 10 {
		t.Fatalf("last successful step not recorded: %+v", snap.ActiveTask.Failure)
	}
	if snap.ActiveTask.Plan.Steps[1].Status != StepFailed {
		t.Fatalf("running step should be marked failed: %+v", snap.ActiveTask.Plan.Steps[1])
	}
}
