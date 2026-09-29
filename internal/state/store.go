package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"runtime/debug"
)

const (
	RuntimeDirname          = ".loa"
	SessionsDirname         = "sessions"
	DebugDirname            = "debug"
	SessionRegistryFilename = "sessions.json"
)

type SessionInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SessionStatus struct {
	ActiveID      string        `json:"active_id,omitempty"`
	ActiveTitle   string        `json:"active_title,omitempty"`
	NeedsCreation bool          `json:"needs_creation"`
	Sessions      []SessionInfo `json:"sessions"`
}

type PersistenceStatus struct {
	Dirty          bool       `json:"dirty"`
	LastSavedAt    *time.Time `json:"last_saved_at,omitempty"`
	LastSaveReason string     `json:"last_save_reason,omitempty"`
	SaveSequence   uint64     `json:"save_sequence"`
}

type sessionRegistry struct {
	Version             int           `json:"version"`
	LastActiveSessionID string        `json:"last_active_session_id,omitempty"`
	Sessions            []SessionInfo `json:"sessions"`
}

type Store struct {
	mu             sync.RWMutex
	root           string
	s              *AgentState
	registry       sessionRegistry
	dirty          bool
	lastSavedAt    *time.Time
	lastSaveReason string
	saveSequence   uint64
	PlanMutationLogger func(stack string, before, after []byte)
}

func NewStore(root string, s *AgentState) *Store {
	if s == nil {
		s = NewAgentState()
	}
	st := &Store{root: root, s: s, registry: sessionRegistry{Version: 1}}
	if s.SessionID != "" {
		st.registry.LastActiveSessionID = s.SessionID
		st.registry.Sessions = []SessionInfo{{ID: s.SessionID, Title: s.SessionTitle, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}}
		if !s.UpdatedAt.IsZero() {
			t := s.UpdatedAt
			st.lastSavedAt = &t
		}
	}
	return st
}

func Load(root string) (*Store, error) {
	st := NewStore(root, NewAgentState())
	reg, err := loadRegistry(root)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	st.registry = reg
	if len(reg.Sessions) == 0 {
		return st, nil
	}

	activeID := reg.LastActiveSessionID
	if !sessionExists(reg.Sessions, activeID) {
		latest := append([]SessionInfo(nil), reg.Sessions...)
		sort.Slice(latest, func(i, j int) bool { return latest[i].UpdatedAt.After(latest[j].UpdatedAt) })
		activeID = latest[0].ID
	}
	s, err := loadSessionState(root, activeID)
	if err != nil {
		return nil, err
	}
	
	// If a task was running when the server unexpectedly shutdown, 
	// safely reset its state to paused so the UI doesn't brick.
	if s.ActiveTask != nil && s.ActiveTask.Status == TaskRunning {
		s.ActiveTask.Status = TaskPaused
	}
	
	st.s = s
	st.registry.LastActiveSessionID = activeID
	if !s.UpdatedAt.IsZero() {
		t := s.UpdatedAt
		st.lastSavedAt = &t
	}
	return st, nil
}

func (st *Store) HasActiveSession() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s != nil && st.s.SessionID != ""
}

func (st *Store) CreateSession(title string) (SessionInfo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return SessionInfo{}, errors.New("session title is required")
	}
	if len(title) > 120 {
		return SessionInfo{}, errors.New("session title must be 120 characters or fewer")
	}
	if st.HasActiveSession() {
		st.mu.RLock()
		dirty := st.dirty
		st.mu.RUnlock()
		if dirty {
			if err := st.Save("autosave"); err != nil {
				return SessionInfo{}, fmt.Errorf("save current session before creating a new one: %w", err)
			}
		}
	}
	id, err := newSessionID()
	if err != nil {
		return SessionInfo{}, err
	}
	now := time.Now()
	info := SessionInfo{ID: id, Title: title, CreatedAt: now, UpdatedAt: now}
	s := NewAgentState()
	s.SessionID = id
	s.SessionTitle = title
	s.CreatedAt = now
	s.UpdatedAt = now

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.root == "" {
		return SessionInfo{}, errors.New("store has no project root")
	}
	if err := ensureRuntimeDirs(st.root); err != nil {
		return SessionInfo{}, err
	}
	if err := writeStateFile(sessionPath(st.root, id), s); err != nil {
		return SessionInfo{}, err
	}
	oldRegistry := st.registry
	st.registry.Version = 1
	st.registry.Sessions = append(st.registry.Sessions, info)
	st.registry.LastActiveSessionID = id
	if err := writeRegistry(st.root, st.registry); err != nil {
		st.registry = oldRegistry
		_ = os.Remove(sessionPath(st.root, id))
		return SessionInfo{}, err
	}
	st.s = s
	st.dirty = false
	st.lastSavedAt = timePtr(now)
	st.lastSaveReason = "session_create"
	st.saveSequence++
	return info, nil
}

func (st *Store) SwitchSession(id string) (bool, SessionInfo, error) {
	savedCurrent := false
	id = strings.TrimSpace(id)
	if id == "" {
		return false, SessionInfo{}, errors.New("session id is required")
	}
	if st.HasActiveSession() {
		st.mu.RLock()
		currentID := st.s.SessionID
		dirty := st.dirty
		st.mu.RUnlock()
		if currentID == id {
			info, _ := st.sessionInfo(id)
			return false, info, nil
		}
		if dirty {
			if err := st.Save("autosave"); err != nil {
				return false, SessionInfo{}, fmt.Errorf("save current session before switch: %w", err)
			}
			savedCurrent = true
		}
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	info, ok := findSession(st.registry.Sessions, id)
	if !ok {
		return false, SessionInfo{}, errors.New("session not found")
	}
	s, err := loadSessionState(st.root, id)
	if err != nil {
		return false, SessionInfo{}, err
	}
	oldActive := st.registry.LastActiveSessionID
	st.registry.LastActiveSessionID = id
	if err := writeRegistry(st.root, st.registry); err != nil {
		st.registry.LastActiveSessionID = oldActive
		return false, SessionInfo{}, err
	}
	st.s = s
	st.dirty = false
	if s.UpdatedAt.IsZero() {
		st.lastSavedAt = nil
	} else {
		st.lastSavedAt = timePtr(s.UpdatedAt)
	}
	st.lastSaveReason = "session_switch"
	st.saveSequence++
	return savedCurrent, info, nil
}

func (st *Store) Save(reason string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.root == "" {
		return errors.New("store has no project root")
	}
	if st.s == nil || st.s.SessionID == "" {
		return errors.New("no active session")
	}
	if err := ensureRuntimeDirs(st.root); err != nil {
		return err
	}
	now := time.Now()
	st.s.UpdatedAt = now
	if err := writeStateFile(sessionPath(st.root, st.s.SessionID), st.s); err != nil {
		return err
	}
	st.updateRegistryFromStateLocked()
	if err := writeRegistry(st.root, st.registry); err != nil {
		return err
	}
	st.dirty = false
	st.lastSavedAt = timePtr(now)
	st.lastSaveReason = strings.TrimSpace(reason)
	st.saveSequence++
	return nil
}

func (st *Store) MarshalIndent() ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return json.MarshalIndent(st.s, "", "  ")
}

func (st *Store) Snapshot() AgentState {
	st.mu.RLock()
	defer st.mu.RUnlock()
	b, _ := json.Marshal(st.s)
	var cp AgentState
	_ = json.Unmarshal(b, &cp)
	return cp
}

func (st *Store) SessionStatus() SessionStatus {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := SessionStatus{NeedsCreation: st.s == nil || st.s.SessionID == "", Sessions: append([]SessionInfo{}, st.registry.Sessions...)}
	if st.s != nil {
		out.ActiveID = st.s.SessionID
		out.ActiveTitle = st.s.SessionTitle
	}
	sort.Slice(out.Sessions, func(i, j int) bool {
		if out.Sessions[i].UpdatedAt.Equal(out.Sessions[j].UpdatedAt) {
			return out.Sessions[i].ID > out.Sessions[j].ID
		}
		return out.Sessions[i].UpdatedAt.After(out.Sessions[j].UpdatedAt)
	})
	return out
}

func (st *Store) PersistenceStatus() PersistenceStatus {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := PersistenceStatus{Dirty: st.dirty, LastSaveReason: st.lastSaveReason, SaveSequence: st.saveSequence}
	if st.lastSavedAt != nil {
		t := *st.lastSavedAt
		out.LastSavedAt = &t
	}
	return out
}

func (st *Store) NextID() uint64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	id := st.s.ID()
	st.markDirtyLocked()
	return id
}

func (st *Store) AddMessage(m Message) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.Messages = append(st.s.Messages, m)
	st.markDirtyLocked()
}

func (st *Store) SetMessageIntents(id uint64, intents []Intent) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.s.Messages {
		if st.s.Messages[i].ID == id {
			st.s.Messages[i].Intents = intents
			st.markDirtyLocked()
			return
		}
	}
}


func (st *Store) AddLog(e ExecutionLogEntry) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.ExecutionLog = append(st.s.ExecutionLog, e)
	st.markDirtyLocked()
}

func (st *Store) AddToolCall(c ToolCall) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.ToolCalls = append(st.s.ToolCalls, c)
	st.markDirtyLocked()
}

func (st *Store) SetToolApproval(id uint64, approved bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.s.ToolCalls {
		if st.s.ToolCalls[i].ID == id {
			st.s.ToolCalls[i].Approved = &approved
			st.markDirtyLocked()
			return
		}
	}
}

func (st *Store) AddToolResult(r ToolResult) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.ToolResults = append(st.s.ToolResults, r)
	st.markDirtyLocked()
}

func (st *Store) SetActiveTask(t *TaskState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.ActiveTask = t
	st.markDirtyLocked()
}
func (st *Store) UpdateActiveTask(fn func(*TaskState)) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask != nil {
		var beforePlan []byte
		if st.PlanMutationLogger != nil {
			beforePlan, _ = json.Marshal(st.s.ActiveTask.Plan)
		}
		fn(st.s.ActiveTask)
		if st.PlanMutationLogger != nil {
			afterPlan, _ := json.Marshal(st.s.ActiveTask.Plan)
			if string(beforePlan) != string(afterPlan) {
				go st.PlanMutationLogger(string(debug.Stack()), beforePlan, afterPlan)
			}
		}
		st.markDirtyLocked()
	}
}
func (st *Store) CompleteActiveTask(status TaskStatus) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil {
		return
	}
	now := time.Now()
	st.s.ActiveTask.Status = status
	st.s.ActiveTask.CompletedAt = &now
	st.s.CompletedTasks = append(st.s.CompletedTasks, *st.s.ActiveTask)
	st.s.ActiveTask = nil
	st.markDirtyLocked()
}
func (st *Store) PauseActiveTask() {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil {
		return
	}
	st.s.ActiveTask.Status = TaskPaused
	st.markDirtyLocked()
}
func (st *Store) PauseActiveTaskWithError(reason string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil {
		return
	}
	
	// Revert the currently executing step to pending so it cleanly retries
	for i := range st.s.ActiveTask.Plan.Steps {
		if st.s.ActiveTask.Plan.Steps[i].Status == StepRunning {
			st.s.ActiveTask.Plan.Steps[i].Status = StepPending
			break
		}
	}
	
	st.s.ActiveTask.Status = TaskPaused
	st.markDirtyLocked()
}
func (st *Store) SetActiveTaskExtras(extras []string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil {
		return
	}
	st.s.ActiveTask.Extras = extras
	st.markDirtyLocked()
}
func (st *Store) PurgeActiveTaskExtra(index int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil || index < 0 || index >= len(st.s.ActiveTask.Extras) {
		return
	}
	st.s.ActiveTask.Extras = append(st.s.ActiveTask.Extras[:index], st.s.ActiveTask.Extras[index+1:]...)
	st.markDirtyLocked()
}
func (st *Store) ResumeActiveTask() {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil {
		return
	}
	st.s.ActiveTask.Status = TaskRunning
	st.markDirtyLocked()
}
func (st *Store) FailActiveTask(reason string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.s.ActiveTask == nil {
		return
	}

	var failedStepID *uint64
	for i := range st.s.ActiveTask.Plan.Steps {
		if st.s.ActiveTask.Plan.Steps[i].Status == StepRunning {
			id := st.s.ActiveTask.Plan.Steps[i].ID
			failedStepID = &id
			st.s.ActiveTask.Plan.Steps[i].Status = StepFailed
			break
		}
	}

	var lastSuccessfulStepID *uint64
	for i := len(st.s.ActiveTask.StepResults) - 1; i >= 0; i-- {
		if st.s.ActiveTask.StepResults[i].Success {
			id := st.s.ActiveTask.StepResults[i].StepID
			lastSuccessfulStepID = &id
			break
		}
	}

	st.s.ActiveTask.Status = TaskFailed
	st.s.ActiveTask.Failure = &TaskFailure{
		FailedStepID:         failedStepID,
		LastSuccessfulStepID: lastSuccessfulStepID,
		Reason:               reason,
		Recoverable:          true,
		FailedAt:             time.Now(),
	}
	st.markDirtyLocked()
}
func (st *Store) SetPendingQuestion(q *PendingQuestion) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.PendingQuestion = q
	st.markDirtyLocked()
}

func (st *Store) AddUpload(u UploadState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s.Uploads = append(st.s.Uploads, u)
	st.markDirtyLocked()
}

func (st *Store) UpdateUpload(id string, fn func(*UploadState)) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.s.Uploads {
		if st.s.Uploads[i].ID == id {
			fn(&st.s.Uploads[i])
			st.markDirtyLocked()
			return
		}
	}
}

func (st *Store) DeleteUpload(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.s.Uploads {
		if st.s.Uploads[i].ID == id {
			st.s.Uploads = append(st.s.Uploads[:i], st.s.Uploads[i+1:]...)
			st.markDirtyLocked()
			return
		}
	}
}

func (st *Store) AddActiveAttachment(a AttachmentState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, existing := range st.s.ActiveAttachments {
		if existing.ID == a.ID {
			return // Already attached
		}
	}
	st.s.ActiveAttachments = append(st.s.ActiveAttachments, a)
	st.markDirtyLocked()
}

func (st *Store) RemoveActiveAttachment(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.s.ActiveAttachments {
		if st.s.ActiveAttachments[i].ID == id {
			st.s.ActiveAttachments = append(st.s.ActiveAttachments[:i], st.s.ActiveAttachments[i+1:]...)
			st.markDirtyLocked()
			return
		}
	}
}

func (st *Store) markDirtyLocked() {
	if st.s != nil {
		st.s.Sequence++
		if st.s.SessionID != "" {
			st.dirty = true
		}
	}
}

func (st *Store) updateRegistryFromStateLocked() {
	info := SessionInfo{ID: st.s.SessionID, Title: st.s.SessionTitle, CreatedAt: st.s.CreatedAt, UpdatedAt: st.s.UpdatedAt}
	for i := range st.registry.Sessions {
		if st.registry.Sessions[i].ID == info.ID {
			st.registry.Sessions[i] = info
			st.registry.LastActiveSessionID = info.ID
			return
		}
	}
	st.registry.Sessions = append(st.registry.Sessions, info)
	st.registry.LastActiveSessionID = info.ID
}

func (st *Store) sessionInfo(id string) (SessionInfo, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return findSession(st.registry.Sessions, id)
}

func RuntimeDir(root string) string  { return filepath.Join(root, RuntimeDirname) }
func DebugDir(root string) string    { return filepath.Join(RuntimeDir(root), DebugDirname) }
func SessionsDir(root string) string { return filepath.Join(RuntimeDir(root), SessionsDirname) }

func ensureRuntimeDirs(root string) error {
	if err := os.MkdirAll(SessionsDir(root), 0o700); err != nil {
		return err
	}
	return os.MkdirAll(DebugDir(root), 0o700)
}

func registryPath(root string) string {
	return filepath.Join(RuntimeDir(root), SessionRegistryFilename)
}
func sessionPath(root, id string) string {
	return filepath.Join(SessionsDir(root), id+".json")
}

func loadRegistry(root string) (sessionRegistry, error) {
	b, err := os.ReadFile(registryPath(root))
	if err != nil {
		return sessionRegistry{}, err
	}
	var reg sessionRegistry
	if err := json.Unmarshal(b, &reg); err != nil {
		return sessionRegistry{}, fmt.Errorf("decode %s: %w", registryPath(root), err)
	}
	if reg.Version == 0 {
		reg.Version = 1
	}
	return reg, nil
}

func writeRegistry(root string, reg sessionRegistry) error {
	if err := ensureRuntimeDirs(root); err != nil {
		return err
	}
	b, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(registryPath(root), b, 0o600)
}

func loadSessionState(root, id string) (*AgentState, error) {
	path := sessionPath(root, id)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s AgentState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if s.SessionID == "" || s.SessionID != id {
		return nil, fmt.Errorf("session file %s has invalid session id", path)
	}
	var maxID uint64
	for _, m := range s.Messages {
		if m.ID > maxID {
			maxID = m.ID
		}
	}
	for _, tc := range s.ToolCalls {
		if tc.ID > maxID {
			maxID = tc.ID
		}
	}
	
	checkTask := func(t *TaskState) {
		if t == nil {
			return
		}
		if t.ID > maxID {
			maxID = t.ID
		}
		for _, step := range t.Plan.Steps {
			if step.ID > maxID {
				maxID = step.ID
			}
		}
	}
	
	for i := range s.CompletedTasks {
		checkTask(&s.CompletedTasks[i])
	}
	checkTask(s.ActiveTask)

	if s.NextID <= maxID {
		s.NextID = maxID + 1
	}
	if s.NextID == 0 {
		s.NextID = 1
	}
	if s.Version == 0 {
		s.Version = 2
	}
	return &s, nil
}

func writeStateFile(path string, s *AgentState) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, b, 0o600)
}

func atomicWriteFile(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".loa-tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err = f.Chmod(mode); err != nil {
		return err
	}
	if _, err = f.Write(b); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func newSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sessionExists(sessions []SessionInfo, id string) bool {
	_, ok := findSession(sessions, id)
	return ok
}

func findSession(sessions []SessionInfo, id string) (SessionInfo, bool) {
	for _, s := range sessions {
		if s.ID == id {
			return s, true
		}
	}
	return SessionInfo{}, false
}

func timePtr(t time.Time) *time.Time {
	x := t
	return &x
}
