package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/laughingmandev/loa/internal/config"
	"github.com/laughingmandev/loa/internal/state"
)

type Message struct{ Role, Content string }

type Attempt struct {
	PromptKind string `json:"prompt_kind"`
	Prompt     string `json:"prompt"`
	Raw        string `json:"raw"`
	Error      string `json:"error,omitempty"`
}

type Client interface {
	Models(context.Context) ([]string, error)
	ChatJSON(context.Context, string, string, string, any, func() error) ([]Attempt, error)
	ChatText(context.Context, string, string, string) (string, error)
	Embed(context.Context, string) ([]float32, error)
	Stats() state.RuntimeStats
}

type OpenAICompatible struct {
	Config func() config.Config
	HTTP   *http.Client
	inf    atomic.Uint64
	emb    atomic.Uint64
}

func New(cfg func() config.Config) *OpenAICompatible {
	return &OpenAICompatible{Config: cfg, HTTP: &http.Client{Timeout: 10 * time.Minute}}
}

func (c *OpenAICompatible) Stats() state.RuntimeStats {
	return state.RuntimeStats{Inferences: c.inf.Load(), Embeddings: c.emb.Load()}
}

func (c *OpenAICompatible) Models(ctx context.Context) ([]string, error) {
	cfg := c.Config()
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, endpoint(cfg.OllamaURL, "/v1/models"), nil, &resp, time.Duration(cfg.LLMPollingTimeoutSeconds)*time.Second); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		if m.ID != "" {
			out = append(out, m.ID)
		}
	}
	return out, nil
}

func (c *OpenAICompatible) ChatText(ctx context.Context, model, system, user string) (string, error) {
	return c.chat(ctx, model, system, user, false)
}

func (c *OpenAICompatible) ChatJSON(ctx context.Context, model, system, user string, out any, validate func() error) ([]Attempt, error) {
	cfg := c.Config()
	max := cfg.JSONRepairAttempts
	if max < 1 {
		max = 1
	}
	attempts := make([]Attempt, 0, max)
	prompt := user
	kind := "original"
	for i := 0; i < max; i++ {
		raw, err := c.chat(ctx, model, system, prompt, true)
		att := Attempt{PromptKind: kind, Prompt: prompt, Raw: raw}
		if err != nil {
			att.Error = err.Error()
			attempts = append(attempts, att)
			if strings.Contains(err.Error(), "model output was truncated") {
				prompt = repairPrompt(user, raw, att.Error)
				kind = "repair"
				continue
			}
			return attempts, err
		}
		zeroOutput(out)
		if err := decodeStrict(raw, out); err == nil {
			if validate == nil {
				attempts = append(attempts, att)
				return attempts, nil
			}
			if verr := validate(); verr == nil {
				attempts = append(attempts, att)
				return attempts, nil
			} else {
				att.Error = verr.Error()
			}
		} else {
			att.Error = err.Error()
		}
		attempts = append(attempts, att)
		prompt = repairPrompt(user, raw, att.Error)
		kind = "repair"
	}
	return attempts, fmt.Errorf("structured response invalid after %d attempts", max)
}

func zeroOutput(out any) {
	v := reflect.ValueOf(out)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		v.Elem().Set(reflect.Zero(v.Elem().Type()))
	}
}

func decodeStrict(raw string, out any) error {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("response contains trailing JSON/content")
	}
	return nil
}

func repairPrompt(original, raw, parseErr string) string {
	return fmt.Sprintf(`Your previous response did not match the required JSON structure or validation rules.
Repair only what is needed to satisfy the original request and validation error. Return ONLY one JSON object, no markdown.

IMPORTANT REPAIR RULES:
- For enum/constrained fields, use ONLY the exact allowed values stated in the ORIGINAL REQUEST or VALIDATION ERROR.
- Do not invent synonyms, executable names, command names, aliases, or alternative enum labels.
- Preserve valid content from the previous response unless it must change to satisfy validation.
- If the VALIDATION ERROR gives a concrete structural correction (for example action="tool" with a specific tool.kind), follow that correction literally while preserving the original semantic intent and tool input.
- If a field is optional and no valid value is appropriate, omit it or use the explicitly allowed empty value rather than inventing one.

ORIGINAL REQUEST:
%s

PREVIOUS RESPONSE:
%s

VALIDATION ERROR:
%s`, original, raw, parseErr)
}

func (c *OpenAICompatible) chat(ctx context.Context, model, system, user string, jsonMode bool) (string, error) {
	cfg := c.Config()
	if model == "" {
		return "", errors.New("no model selected for this inference")
	}
	body := map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": user}},
		"stream":      false,
		"temperature": 0.2,
		"max_tokens":  cfg.OutputReserve,
	}
	if jsonMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	if !strings.Contains(cfg.OllamaURL, "api.openai.com") && cfg.ContextBudget > 0 {
		body["options"] = map[string]any{
			"num_ctx": cfg.ContextBudget,
		}
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	c.inf.Add(1)
	if err := c.do(ctx, http.MethodPost, endpoint(cfg.OllamaURL, "/v1/chat/completions"), body, &resp, time.Duration(cfg.ModelTimeoutSeconds)*time.Second); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("model returned no choices")
	}
	if resp.Choices[0].FinishReason == "length" {
		return "", errors.New("model output was truncated (hit max_tokens or context limit)")
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *OpenAICompatible) Embed(ctx context.Context, input string) ([]float32, error) {
	cfg := c.Config()
	if cfg.EmbeddingModel == "" {
		return nil, errors.New("no embedding model selected")
	}
	body := map[string]any{"model": cfg.EmbeddingModel, "input": input}
	var resp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	c.emb.Add(1)
	if err := c.do(ctx, http.MethodPost, endpoint(cfg.OllamaURL, "/v1/embeddings"), body, &resp, time.Duration(cfg.ModelTimeoutSeconds)*time.Second); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		return nil, errors.New("embedding endpoint returned no vector")
	}
	return resp.Data[0].Embedding, nil
}

func (c *OpenAICompatible) do(ctx context.Context, method, url string, body any, out any, timeout time.Duration) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	cfg := c.Config()
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	client := *c.HTTP
	if timeout > 0 {
		client.Timeout = timeout
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ollama returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("decode ollama response: %w", err)
		}
	}
	return nil
}

func endpoint(base, path string) string { return strings.TrimRight(base, "/") + path }
