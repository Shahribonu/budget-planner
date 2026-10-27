// Package llm holds the model fallbacks of the capture pipeline. Models only propose:
// the classifier returns a category key that the caller validates against the database.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const model = "claude-haiku-5-5"

// Category is one choice offered to the model.
type Category struct {
	Key  string
	Kind string // expense or income
}

type Result struct {
	Key          string // "" when the model is not sure
	InputTokens  int64
	OutputTokens int64
}

type Classifier struct {
	client anthropic.Client
}

// New returns nil when apiKey is empty, so callers can treat the fallback as optional.
func New(apiKey string) *Classifier {
	if apiKey == "" {
		return nil
	}
	return &Classifier{client: anthropic.NewClient(option.WithAPIKey(apiKey), option.WithMaxRetries(1))}
}

const system = `You categorize short personal-finance notes from users in Uzbekistan.
Notes are in Uzbek (Latin or Cyrillic), Russian or English, often informal, and may name a shop,
brand or service (for example Korzinka is a supermarket, Evos is fast food, Paynet tops up phones).
Pick the single best category key from the list for the kind given. If the note is too vague to
tell, answer "unknown" rather than guessing.`

// card numbers, phone numbers and other long digit runs never leave the server
var longDigits = regexp.MustCompile(`\d[\d\s-]{3,}\d`)

func Mask(s string) string { return longDigits.ReplaceAllString(s, "#") }

// Classify asks the model for a category key of the given kind for description.
func (c *Classifier) Classify(ctx context.Context, description, kind string, cats []Category) (Result, error) {
	keys := []any{"unknown"}
	var list strings.Builder
	for _, cat := range cats {
		if cat.Kind == kind {
			keys = append(keys, cat.Key)
			list.WriteString("- " + cat.Key + "\n")
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"category": map[string]any{"type": "string", "enum": keys}},
		"required":             []string{"category"},
		"additionalProperties": false,
	}
	prompt := fmt.Sprintf("Kind: %s\nCategories:\n%sNote: %q", kind, list.String(), Mask(description))

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 256,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Thinking:  anthropic.ThinkingConfigParamUnion{OfDisabled: &anthropic.ThinkingConfigDisabledParam{}},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return Result{}, err
	}
	res := Result{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens}
	if resp.StopReason != anthropic.StopReasonEndTurn {
		return res, nil
	}
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			var out struct {
				Category string `json:"category"`
			}
			if err := json.Unmarshal([]byte(b.Text), &out); err != nil {
				return res, fmt.Errorf("parse classifier output: %w", err)
			}
			if out.Category != "unknown" {
				res.Key = out.Category
			}
		}
	}
	return res, nil
}
