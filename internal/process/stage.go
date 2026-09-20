package process

import (
	"context"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

// Input is a raw item already stored by any source.
type Input struct {
	Content   string
	Media     int
	Duplicate bool
}

// Output is the shared decision. LLMStatus "call" means the caller should claim the row and run ApplyLLM.
// Delivery is not decided here.
type Output struct {
	LLMStatus  string
	Reason     string
	Result     any
	Importance float64
}

// Run sits behind every source. Telegram and RSS only insert rows; they do not call this.
func Run(in Input, st kernel.Settings) Output {
	ok, reason := Passes(in.Content, st.MinLength, st.BlockKeywords, in.Media > 0)
	if !ok {
		return Output{LLMStatus: "rejected", Reason: reason}
	}
	if in.Duplicate {
		return Output{LLMStatus: "rejected", Reason: "文本 hash 重复"}
	}
	if !st.LLMEnabled {
		return Output{
			LLMStatus: "skipped", Reason: "",
			Result:     map[string]any{"send": true, "title": FallbackTitle(in.Content), "body": strings.TrimSpace(in.Content), "reason": "LLM关闭直通"},
			Importance: 5,
		}
	}
	return Output{LLMStatus: "call"}
}

// ApplyLLM is the LLM module. Call it only after Run returns LLMStatus "call".
func ApplyLLM(ctx context.Context, st kernel.Settings, content string, media int) (Output, error) {
	review, err := ReviewMessage(ctx, st, content, media)
	if err != nil {
		return Output{}, err
	}
	if !review.Send {
		return Output{LLMStatus: "rejected", Reason: review.Reason, Result: review, Importance: review.Importance}, nil
	}
	if review.Title == "" {
		review.Title = FallbackTitle(content)
	}
	if review.Body == "" {
		review.Body = strings.TrimSpace(content)
	}
	return Output{LLMStatus: "approved", Result: review, Importance: review.Importance}, nil
}
