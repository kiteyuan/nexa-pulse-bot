package process

import (
	"testing"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func TestRunRejectsShortAndDuplicate(t *testing.T) {
	st := kernel.Settings{MinLength: 8, BlockKeywords: []string{"广告"}}
	if out := Run(Input{Content: "短"}, st); out.LLMStatus != "rejected" {
		t.Fatalf("short: %+v", out)
	}
	if out := Run(Input{Content: "这是一条正常内容", Duplicate: true}, st); out.LLMStatus != "rejected" {
		t.Fatalf("dup: %+v", out)
	}
	if out := Run(Input{Content: "免费广告来了"}, st); out.LLMStatus != "rejected" {
		t.Fatalf("kw: %+v", out)
	}
}

func TestRunSkipsWhenLLMOff(t *testing.T) {
	st := kernel.Settings{MinLength: 4}
	out := Run(Input{Content: "这是一条正常内容"}, st)
	if out.LLMStatus != "skipped" {
		t.Fatalf("got %+v", out)
	}
}

func TestRunCallsLLMWhenEnabled(t *testing.T) {
	st := kernel.Settings{MinLength: 4, LLMEnabled: true}
	out := Run(Input{Content: "这是一条正常内容"}, st)
	if out.LLMStatus != "call" {
		t.Fatalf("got %+v", out)
	}
}
