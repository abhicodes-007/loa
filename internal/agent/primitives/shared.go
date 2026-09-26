package primitives

import (
	"errors"
	"fmt"
	"strings"

	"github.com/laughingmandev/loa/internal/state"
	"github.com/laughingmandev/loa/internal/tools"
)

const ToolKindEnum = `read_file|read_range|search_path|search_text|find_symbol|list_symbols|read_ast_node|write_file|patch_file|patch_ast_node|create_directory|delete_file|delete_directory|execute_process|execute_shell|git_status|git_diff|read_tool_output|analyze_large_file|search_task_steps|read_task_step`

func NeedReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func FillToolDescription(reason string, r *tools.Request) {
	if r == nil || strings.TrimSpace(r.Description) != "" {
		return
	}
	r.Description = strings.TrimSpace(reason)
}

func ToolActionRepairHint(action string) error {
	kind := state.ToolKind(action)
	if tools.ValidKind(kind) {
		return fmt.Errorf("action %q is a tool kind, not an action; set action=\"tool\", set tool.kind=%q, and preserve the existing tool.input", action, action)
	}
	return nil
}

func ValidateToolRequest(r *tools.Request) error {
	if r == nil {
		return errors.New("tool required")
	}
	if !tools.ValidKind(r.Kind) {
		return fmt.Errorf("invalid tool kind %q", r.Kind)
	}
	if len(r.Input) == 0 {
		return errors.New("tool input object is required")
	}
	return nil
}
