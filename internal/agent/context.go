package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/laughingmandev/loa/internal/config"
	"github.com/laughingmandev/loa/internal/state"
)

func buildContext(root string, cfg config.Config, snap state.AgentState, step *state.PlanStep, extras []string, includeInstructions bool) string {
	var mandatory strings.Builder
	fmt.Fprintf(&mandatory, "CURRENT TIME: %s\nPROJECT ROOT: %s\n", time.Now().Format(time.RFC3339), root)
	if includeInstructions && strings.TrimSpace(cfg.ProjectInstructions) != "" {
		fmt.Fprintf(&mandatory, "\nPERSISTENT PROJECT INSTRUCTIONS:\n%s\n", cfg.ProjectInstructions)
	}
	if snap.PendingQuestion != nil {
		b, _ := json.MarshalIndent(snap.PendingQuestion, "", "  ")
		mandatory.WriteString("\nPENDING CLARIFICATION:\n")
		mandatory.Write(b)
		mandatory.WriteByte('\n')
	}
	if snap.ActiveTask != nil {
		b, _ := json.MarshalIndent(map[string]any{
			"id":                  snap.ActiveTask.ID,
			"goal":                snap.ActiveTask.Goal,
			"may_modify":          snap.ActiveTask.MayModify,
			"status":              snap.ActiveTask.Status,
			"acceptance_criteria": snap.ActiveTask.AcceptanceCriteria,
			"acceptance_checks":   snap.ActiveTask.AcceptanceChecks,
			"failure":             snap.ActiveTask.Failure,
			"plan":                snap.ActiveTask.Plan,
			"task_context":        snap.ActiveTask.Context,
			"user_guidance":       snap.ActiveTask.Guidance,
			"prerequisites":       getPrerequisites(step, snap.ActiveTask.StepResults),
			"recent_breadcrumbs":  getBreadcrumbs(snap.ActiveTask.StepResults, cfg.ContextRecentBreadcrumbs),
		}, "", "  ")
		mandatory.WriteString("\nACTIVE TASK STATE:\n")
		mandatory.Write(b)
		mandatory.WriteByte('\n')
	}
	if step != nil {
		b, _ := json.MarshalIndent(step, "", "  ")
		mandatory.WriteString("\nCURRENT STEP:\n")
		mandatory.Write(b)
		mandatory.WriteByte('\n')
	}

	if len(snap.ActiveAttachments) > 0 {
		mandatory.WriteString("\nUSER ATTACHMENTS (Read-Only):\n")
		for _, att := range snap.ActiveAttachments {
			readOnly := false
			if att.Type == state.AttachmentTypeUpload {
				for _, up := range snap.Uploads {
					if up.ID == att.ID {
						readOnly = up.ReadOnly
						break
					}
				}
			} else {
				readOnly = true
			}
			if readOnly {
				fmt.Fprintf(&mandatory, "- [READ ONLY] %s\n", att.VirtualPath)
			} else {
				fmt.Fprintf(&mandatory, "- %s\n", att.VirtualPath)
			}
		}
		mandatory.WriteByte('\n')
	}

	// Approximate token accounting.
	buffer := cfg.ContextBudget / 20
	if buffer < cfg.ContextMinimumBuffer {
		buffer = cfg.ContextMinimumBuffer
	}
	safeBudget := cfg.ContextBudget - buffer
	if safeBudget < cfg.ContextMinimumSafeBudget {
		safeBudget = cfg.ContextMinimumSafeBudget // absolute floor
	}
	maxRunes := (safeBudget - cfg.OutputReserve) * 4
	if maxRunes < cfg.ContextCodeBudgetMaxFloor {
		maxRunes = cfg.ContextCodeBudgetMaxFloor
	}
	baseRunes := []rune(mandatory.String())
	if len(baseRunes) >= maxRunes {
		return string(baseRunes[:maxRunes])
	}
	remaining := maxRunes - len(baseRunes)

	recent := tailMessages(snap.Messages, cfg.RecentMessages)
	var recentParts []string
	for i := len(recent) - 1; i >= 0; i-- {
		line := fmt.Sprintf("[%s] %s: %s", recent[i].CreatedAt.Format(time.RFC3339), recent[i].Source, recent[i].Text)
		lineRunes := []rune(line)
		if len(lineRunes)+1 > remaining {
			break
		}
		recentParts = append(recentParts, line)
		remaining -= len(lineRunes) + 1
	}
	reverse(recentParts)

	codeCap := cfg.CodeBudget * 4
	if codeCap <= 0 {
		codeCap = 32000
	}
	var extraParts []string
	usedCode := 0
	for i := len(extras) - 1; i >= 0 && remaining > 0; i-- {
		xRunes := []rune(extras[i])
		if len(xRunes) > codeCap-usedCode {
			limit := codeCap - usedCode
			if limit < 0 {
				limit = 0
			}
			xRunes = xRunes[:limit]
		}
		if len(xRunes) == 0 {
			continue
		}
		if len(xRunes)+1 > remaining {
			xRunes = xRunes[:remaining-1]
		}
		extraParts = append(extraParts, string(xRunes))
		remaining -= len(xRunes) + 1
		usedCode += len(xRunes)
	}
	reverse(extraParts)

	var out strings.Builder
	out.WriteString(string(baseRunes))
	if len(recentParts) > 0 {
		out.WriteString("\nRECENT CONVERSATION:\n")
		out.WriteString(strings.Join(recentParts, "\n"))
		out.WriteByte('\n')
	}
	if len(extraParts) > 0 {
		out.WriteString("\nRETRIEVED / TOOL CONTEXT:\n")
		out.WriteString(strings.Join(extraParts, "\n\n"))
		out.WriteByte('\n')
	}
	return out.String()
}

func tailMessages(in []state.Message, n int) []state.Message {
	if n <= 0 || len(in) <= n {
		return in
	}
	return in[len(in)-n:]
}
func tailSteps(in []state.StepResult, n int) []state.StepResult {
	if n <= 0 || len(in) <= n {
		return in
	}
	return in[len(in)-n:]
}
func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func getPrerequisites(step *state.PlanStep, results []state.StepResult) []state.StepResult {
	if step == nil || len(step.DependsOn) == 0 {
		return nil
	}
	var prereqs []state.StepResult
	for _, depID := range step.DependsOn {
		for _, r := range results {
			if r.StepID == depID {
				prereqs = append(prereqs, r)
				break
			}
		}
	}
	return prereqs
}

func getBreadcrumbs(results []state.StepResult, n int) []string {
	tail := tailSteps(results, n)
	var crumbs []string
	for _, r := range tail {
		status := "Success"
		if !r.Success {
			status = "Failed"
		}
		crumbs = append(crumbs, fmt.Sprintf("[Step %d] %s: %s", r.StepID, status, r.Summary))
	}
	return crumbs
}
