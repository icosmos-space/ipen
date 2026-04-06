package agents

import (
	"reflect"
	"testing"

	"github.com/icosmos-space/ipen/core/llm"
)

func TestNormalizeChatOptionsInjectsStreamCallbacksAndLabel(t *testing.T) {
	chunkCallback := func(chunk llm.StreamChunk) {}
	progressCallback := func(progress llm.StreamProgress) {}

	agent := NewBaseAgent(AgentContext{
		OnStreamChunk:    chunkCallback,
		OnStreamProgress: progressCallback,
	})

	input := &llm.ChatOptions{
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	normalized := agent.normalizeChatOptions(input)

	if normalized == input {
		t.Fatal("expected normalized options to be cloned")
	}
	if normalized.StreamLabel != "base" {
		t.Fatalf("expected default stream label 'base', got %q", normalized.StreamLabel)
	}
	if normalized.OnStreamChunk == nil {
		t.Fatal("expected chunk callback to be injected")
	}
	if normalized.OnStreamProgress == nil {
		t.Fatal("expected progress callback to be injected")
	}
	if normalized.Temperature != input.Temperature || normalized.MaxTokens != input.MaxTokens {
		t.Fatal("expected existing option values to be preserved")
	}

	if input.StreamLabel != "" {
		t.Fatalf("expected original options to stay unchanged, got label %q", input.StreamLabel)
	}
	if input.OnStreamChunk != nil || input.OnStreamProgress != nil {
		t.Fatal("expected original callbacks to remain nil")
	}
}

func TestNormalizeChatOptionsPreservesExplicitStreamSettings(t *testing.T) {
	contextChunkCallback := func(chunk llm.StreamChunk) {}
	contextProgressCallback := func(progress llm.StreamProgress) {}
	explicitChunkCallback := func(chunk llm.StreamChunk) {}
	explicitProgressCallback := func(progress llm.StreamProgress) {}

	agent := NewBaseAgent(AgentContext{
		OnStreamChunk:    contextChunkCallback,
		OnStreamProgress: contextProgressCallback,
	})

	normalized := agent.normalizeChatOptions(&llm.ChatOptions{
		StreamLabel:      "writer",
		OnStreamChunk:    explicitChunkCallback,
		OnStreamProgress: explicitProgressCallback,
	})

	if normalized.StreamLabel != "writer" {
		t.Fatalf("expected explicit stream label to be preserved, got %q", normalized.StreamLabel)
	}
	if normalized.OnStreamChunk == nil || normalized.OnStreamProgress == nil {
		t.Fatal("expected explicit callbacks to be preserved")
	}
	if reflect.ValueOf(normalized.OnStreamChunk).Pointer() == reflect.ValueOf(contextChunkCallback).Pointer() {
		t.Fatal("expected explicit chunk callback to win over context callback")
	}
	if reflect.ValueOf(normalized.OnStreamProgress).Pointer() == reflect.ValueOf(contextProgressCallback).Pointer() {
		t.Fatal("expected explicit progress callback to win over context callback")
	}
}
