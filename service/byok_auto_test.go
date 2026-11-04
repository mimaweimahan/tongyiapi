package service

import (
	"testing"
)

func TestByokClassifyChatIntent(t *testing.T) {
	agent := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "你好"},
		},
		"tools": []any{map[string]any{"type": "function"}},
	}
	if got := ByokClassifyChatIntent(agent); got != "agent" {
		t.Fatalf("agent intent: got %s", got)
	}

	chat := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "你好"},
		},
	}
	if got := ByokClassifyChatIntent(chat); got != "chat" {
		t.Fatalf("chat intent: got %s", got)
	}

	code := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "帮我改代码 ```go\nfunc main(){}\n```"},
		},
	}
	if got := ByokClassifyChatIntent(code); got != "code" {
		t.Fatalf("code intent: got %s", got)
	}
}

func TestByokAutoChatCandidatesVoucherFirst(t *testing.T) {
	names := []string{
		"deepseek-ai/DeepSeek-V4.1-Flash",
		"deepseek-ai/DeepSeek-V3.2",
		"inclusionAI/Ling-flash-2.0",
		"BAAI/bge-m3",
		"auto",
	}
	payload := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "你好"},
		},
		"tools": []any{map[string]any{"type": "function"}},
	}
	cands, intent := ByokAutoChatCandidates(names, payload)
	if intent != "agent" {
		t.Fatalf("intent=%s", intent)
	}
	if len(cands) == 0 {
		t.Fatal("empty candidates")
	}
	if cands[0] != "deepseek-ai/DeepSeek-V3.2" {
		t.Fatalf("first candidate want V3.2, got %s (all=%v)", cands[0], cands)
	}
	for _, c := range cands {
		if c == "BAAI/bge-m3" || c == "auto" {
			t.Fatalf("non-chat slipped in: %v", cands)
		}
	}
}

func TestByokShouldFailover(t *testing.T) {
	if !ByokShouldFailover(429, nil) {
		t.Fatal("429")
	}
	if !ByokShouldFailover(404, nil) {
		t.Fatal("404")
	}
	if ByokShouldFailover(401, []byte("unauthorized")) {
		t.Fatal("401 should not failover")
	}
	if !ByokShouldFailover(400, []byte(`{"error":{"message":"model_not_found"}}`)) {
		t.Fatal("400 model_not_found")
	}
	if !ByokShouldFailover(400, []byte(`{"error":{"message":"Model x has not been priced","code":"model_price_error"}}`)) {
		t.Fatal("400 model_price_error")
	}
}
