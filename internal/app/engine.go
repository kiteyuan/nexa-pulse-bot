package app

import (
	"context"
	"time"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
	"github.com/kiteyuan/nexa-pulse-bot/internal/process"
)

type Engine struct {
	Pipe    ports.Pipeline
	Sources []ports.Intake
}

func (e *Engine) Run(ctx context.Context) {
	go e.collectLoop(ctx)
	go e.processLoop(ctx)
}

func (e *Engine) collectLoop(ctx context.Context) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			st, err := e.Pipe.Settings(ctx)
			wait := 30 * time.Minute
			if err == nil {
				wait = time.Duration(st.CollectInterval * float64(time.Second))
			}
			if wait < time.Second {
				wait = time.Second
			}
			_ = e.pollSources(ctx)
			timer.Reset(wait)
		}
	}
}

func (e *Engine) processLoop(ctx context.Context) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			st, err := e.Pipe.Settings(ctx)
			wait := 2 * time.Second
			if err == nil {
				wait = time.Duration(st.PollInterval * float64(time.Second))
			}
			if wait < time.Second {
				wait = time.Second
			}
			if err == nil {
				e.process(ctx, st)
			}
			timer.Reset(wait)
		}
	}
}

func (e *Engine) process(ctx context.Context, st kernel.Settings) {
	items, err := e.Pipe.PendingMessages(ctx, 20)
	if err != nil {
		return
	}
	for _, msg := range items {
		e.processOne(ctx, st, msg)
	}
}

func (e *Engine) processOne(ctx context.Context, st kernel.Settings, msg kernel.Message) {
	dup, err := e.Pipe.HashExists(ctx, msg.ContentHash, msg.ID)
	if err != nil {
		return
	}
	out := process.Run(process.Input{Content: msg.Content, Media: len(msg.MediaPaths), Duplicate: dup}, st)
	if out.LLMStatus == "call" {
		claimed, err := e.Pipe.ClaimProcessing(ctx, msg.ID)
		if err != nil || !claimed {
			return
		}
		out, err = process.ApplyLLM(ctx, st, msg.Content, len(msg.MediaPaths))
		if err != nil {
			_ = e.Pipe.FinishMessage(ctx, msg.ID, "error", "llm 失败", nil, 0)
			e.Pipe.AddLog(ctx, "ERROR", "llm", "调用失败")
			return
		}
	}
	if out.LLMStatus == "rejected" {
		e.Pipe.AddLog(ctx, "INFO", "process", "规则拒绝: "+out.Reason)
	}
	_ = e.Pipe.FinishMessage(ctx, msg.ID, out.LLMStatus, out.Reason, out.Result, out.Importance)
}

func (e *Engine) pollSources(ctx context.Context) error {
	for _, src := range e.Sources {
		if src == nil {
			continue
		}
		if err := src.Poll(ctx); err != nil {
			e.Pipe.AddLog(ctx, "ERROR", src.Name(), "采集失败")
		}
	}
	return nil
}
