package service

import (
	"regexp"
	"strings"
)

// BYOK auto routing (DEV_FEATURES §3.4) — ported from portal app.py.

var ByokChatSkipRE = regexp.MustCompile(`(?i)(embed|bge|rerank|asr|tts|whisper|image|flux|kolors|video|audio|sensevoice|cosyvoice|ocr|speech|live)`)

var byokAutoAliases = map[string]struct{}{
	"auto":      {},
	"auto-chat": {},
}

var byokPreferredChat = []string{
	"deepseek-ai/DeepSeek-V3.2",
	"Pro/deepseek-ai/DeepSeek-V3.2",
	"deepseek-ai/DeepSeek-V3",
	"Pro/deepseek-ai/DeepSeek-V3",
	"inclusionAI/Ling-flash-2.0",
	"Qwen/Qwen3-32B",
	"zai-org/GLM-4.5-Air",
	"deepseek-ai/DeepSeek-R1",
	"deepseek-ai/DeepSeek-V3.1-Terminus",
	"gemini-3-flash-preview",
	"deepseek-ai/DeepSeek-V4.1-Flash",
	"Pro/moonshotai/Kimi-K2.6",
	"deepseek-ai/DeepSeek-V4-Flash",
	"gemini-flash-latest",
	"gemini-3.8-flash",
}

// Silicon voucher / coupon chat models — try before balance-only models.
var byokSiliconVoucherChat = map[string]struct{}{
	"deepseek-ai/DeepSeek-V3.2":                   {},
	"Pro/deepseek-ai/DeepSeek-V3.2":               {},
	"deepseek-ai/DeepSeek-V3.1-Terminus":          {},
	"Pro/deepseek-ai/DeepSeek-V3.1-Terminus":      {},
	"deepseek-ai/DeepSeek-V3":                     {},
	"Pro/deepseek-ai/DeepSeek-V3":                 {},
	"deepseek-ai/DeepSeek-V2.5":                   {},
	"deepseek-ai/DeepSeek-R1":                     {},
	"Pro/deepseek-ai/DeepSeek-R1":                 {},
	"deepseek-ai/DeepSeek-R1-0528-Qwen3-8B":       {},
	"deepseek-ai/DeepSeek-R1-Distill-Qwen-32B":    {},
	"deepseek-ai/DeepSeek-R1-Distill-Qwen-14B":    {},
	"deepseek-ai/DeepSeek-R1-Distill-Qwen-7B":     {},
	"Pro/deepseek-ai/DeepSeek-R1-Distill-Qwen-7B": {},
	"inclusionAI/Ling-flash-2.0":                  {},
	"inclusionAI/Ling-mini-2.0":                   {},
	"inclusionAI/Ring-flash-2.0":                  {},
	"Qwen/Qwen3-32B":                              {},
	"Qwen/Qwen3-14B":                              {},
	"Qwen/Qwen3-8B":                               {},
	"Qwen/Qwen3-235B-A22B":                        {},
	"Qwen/Qwen3-235B-A22B-Instruct-2507":          {},
	"Qwen/Qwen3-235B-A22B-Thinking-2507":          {},
	"Qwen/Qwen3-30B-A3B":                          {},
	"Qwen/Qwen3-30B-A3B-Instruct-2507":            {},
	"Qwen/Qwen3-30B-A3B-Thinking-2507":            {},
	"Qwen/Qwen3-Next-80B-A3B-Instruct":            {},
	"Qwen/Qwen3-Next-80B-A3B-Thinking":            {},
	"Qwen/Qwen3-Coder-30B-A3B-Instruct":           {},
	"Qwen/Qwen3-Coder-480B-A35B-Instruct":         {},
	"Qwen/Qwen3-VL-235B-A22B-Instruct":            {},
	"Qwen/Qwen3-VL-235B-A22B-Thinking":            {},
	"Qwen/Qwen3-VL-32B-Instruct":                  {},
	"Qwen/Qwen3-VL-32B-Thinking":                  {},
	"Qwen/Qwen3-VL-30B-A3B-Instruct":              {},
	"Qwen/Qwen3-VL-30B-A3B-Thinking":              {},
	"Qwen/Qwen3-VL-8B-Instruct":                   {},
	"Qwen/Qwen3-VL-8B-Thinking":                   {},
	"Qwen/Qwen3-Omni-30B-A3B-Instruct":            {},
	"Qwen/Qwen3-Omni-30B-A3B-Thinking":            {},
	"Qwen/Qwen2.5-72B-Instruct":                   {},
	"Qwen/Qwen2.5-72B-Instruct-128K":              {},
	"Qwen/Qwen2.5-32B-Instruct":                   {},
	"Qwen/Qwen2.5-14B-Instruct":                   {},
	"Qwen/Qwen2.5-7B-Instruct":                    {},
	"Pro/Qwen/Qwen2.5-7B-Instruct":                {},
	"Qwen/Qwen2.5-Coder-32B-Instruct":             {},
	"Qwen/Qwen2.5-Coder-7B-Instruct":              {},
	"Pro/Qwen/Qwen2.5-Coder-7B-Instruct":          {},
	"Qwen/Qwen2-7B-Instruct":                      {},
	"Pro/Qwen/Qwen2-7B-Instruct":                  {},
	"Qwen/QwQ-32B":                                {},
	"Qwen/QVQ-72B-Preview":                        {},
	"zai-org/GLM-4.6":                             {},
	"zai-org/GLM-4.6V":                            {},
	"zai-org/GLM-4.5":                             {},
	"zai-org/GLM-4.5-Air":                         {},
	"zai-org/GLM-4.5V":                            {},
	"THUDM/GLM-4-32B-0414":                        {},
	"THUDM/GLM-4-9B-0414":                         {},
	"THUDM/GLM-Z1-32B-0414":                       {},
	"THUDM/GLM-Z1-9B-0414":                        {},
	"THUDM/GLM-Z1-Rumination-32B-0414":            {},
	"THUDM/GLM-4.1V-9B-Thinking":                  {},
	"Pro/THUDM/GLM-4.1V-9B-Thinking":              {},
	"THUDM/glm-4-9b-chat":                         {},
	"Pro/THUDM/glm-4-9b-chat":                     {},
	"ByteDance-Seed/Seed-OSS-36B-Instruct":        {},
	"SeedLLM/Seed-Rice-7B":                        {},
	"moonshotai/Kimi-K2-Instruct-0905":            {},
	"moonshotai/Kimi-K2-Thinking":                 {},
	"moonshotai/Kimi-Dev-72B":                     {},
	"MiniMaxAI/MiniMax-M2":                        {},
	"MiniMaxAI/MiniMax-M1-80k":                    {},
	"stepfun-ai/step3":                            {},
	"stepfun-ai/Step-3.5-Flash":                   {},
	"tencent/Hunyuan-A13B-Instruct":               {},
	"baidu/ERNIE-4.5-300B-A47B":                   {},
	"Kwaipilot/KAT-Dev":                           {},
	"Tongyi-Zhiwen/QwenLong-L1-32B":               {},
	"internlm/internlm2_5-7b-chat":                {},
	"ascend-tribe/pangu-pro-moe":                  {},
}

const (
	ByokAutoVoucherTries = 3
	ByokAutoBalanceTries = 2
	ByokLongChatChars    = 12000
)

var byokIntentVoucher = map[string][]string{
	"vision": {
		"Qwen/Qwen3-VL-32B-Instruct",
		"Qwen/Qwen3-VL-30B-A3B-Instruct",
		"zai-org/GLM-4.5V",
		"Qwen/Qwen3-VL-8B-Instruct",
		"Qwen/Qwen3-Omni-30B-A3B-Instruct",
	},
	"reason": {
		"deepseek-ai/DeepSeek-R1",
		"deepseek-ai/DeepSeek-V3.2",
		"Qwen/Qwen3-32B",
		"Qwen/Qwen3-30B-A3B-Thinking-2507",
	},
	"code": {
		"deepseek-ai/DeepSeek-V3.2",
		"Qwen/Qwen3-Coder-30B-A3B-Instruct",
		"deepseek-ai/DeepSeek-V3",
		"Qwen/Qwen3-32B",
	},
	"agent": {
		"deepseek-ai/DeepSeek-V3.2",
		"deepseek-ai/DeepSeek-V3",
		"inclusionAI/Ling-flash-2.0",
		"Qwen/Qwen3-32B",
	},
	"long": {
		"deepseek-ai/DeepSeek-V3.2",
		"deepseek-ai/DeepSeek-V3.1-Terminus",
		"deepseek-ai/DeepSeek-V3",
	},
	"chat": {
		"inclusionAI/Ling-flash-2.0",
		"zai-org/GLM-4.5-Air",
		"Qwen/Qwen3-14B",
		"deepseek-ai/DeepSeek-V3.2",
	},
}

var byokCodeHintRE = regexp.MustCompile("(?i)(```|def |class |function |import |from |console\\.|traceback|\\.py\\b|\\.ts\\b|\\.tsx\\b|\\.js\\b|\\.go\\b|\\.java\\b|" +
	"git |npm |pip |SELECT |docker |kubectl |帮我(改|写|修)代码|写一个函数|报错|stack trace)")

var byokReasonHintRE = regexp.MustCompile("(?i)(证明|推理|一步步|思考过程|算一下|方程|积分|求导|概率|olympiad|prove |step by step|think carefully|数学题)")

var byokFailoverNeedles = []string{
	"does not exist",
	"no longer available",
	"无可用渠道",
	"too busy",
	"high demand",
	"model_not_found",
	"model_price_error",
	"has not been priced",
	"not been priced",
	"voucher",
	"coupon",
	"代金券",
	"不适用",
	"不支持使用",
	"charge balance",
	"onlybychargebalance",
}

func ByokIsAutoAlias(name string) bool {
	_, ok := byokAutoAliases[strings.TrimSpace(strings.ToLower(name))]
	return ok
}

func ByokIsChatModel(name string) bool {
	item := strings.TrimSpace(name)
	if item == "" || ByokIsAutoAlias(item) {
		return false
	}
	return !ByokChatSkipRE.MatchString(item)
}

func byokPreferChatOrder(names []string) []string {
	lookup := map[string]struct{}{}
	for _, n := range names {
		lookup[n] = struct{}{}
	}
	ordered := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, prefer := range byokPreferredChat {
		if _, ok := lookup[prefer]; ok {
			if _, dup := seen[prefer]; !dup {
				ordered = append(ordered, prefer)
				seen[prefer] = struct{}{}
			}
		}
	}
	for _, item := range names {
		if _, dup := seen[item]; dup {
			continue
		}
		ordered = append(ordered, item)
		seen[item] = struct{}{}
	}
	return ordered
}

// ByokRankChatModels filters non-chat models and orders voucher pool before balance pool.
func ByokRankChatModels(names []string) []string {
	seen := make([]string, 0, len(names))
	dup := map[string]struct{}{}
	for _, name := range names {
		item := strings.TrimSpace(name)
		if !ByokIsChatModel(item) {
			continue
		}
		if _, ok := dup[item]; ok {
			continue
		}
		dup[item] = struct{}{}
		seen = append(seen, item)
	}
	voucher := make([]string, 0)
	others := make([]string, 0)
	for _, item := range seen {
		if _, ok := byokSiliconVoucherChat[item]; ok {
			voucher = append(voucher, item)
		} else {
			others = append(others, item)
		}
	}
	return append(byokPreferChatOrder(voucher), byokPreferChatOrder(others)...)
}

func ByokModelFamily(name string) string {
	item := strings.TrimSpace(name)
	if strings.HasPrefix(item, "Pro/") {
		return item[4:]
	}
	return item
}

func ByokPickUniqueModels(names []string, limit int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, limit)
	for _, name := range names {
		family := ByokModelFamily(name)
		if family == "" {
			continue
		}
		if _, ok := seen[family]; ok {
			continue
		}
		seen[family] = struct{}{}
		out = append(out, name)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func byokFlattenMessageContent(content any) (string, bool) {
	switch v := content.(type) {
	case string:
		return v, false
	case []any:
		texts := make([]string, 0, len(v))
		hasImage := false
		for _, part := range v {
			m, ok := part.(map[string]any)
			if !ok {
				continue
			}
			kind, _ := m["type"].(string)
			if kind == "image_url" || kind == "image" || kind == "input_image" || m["image_url"] != nil || m["image"] != nil {
				hasImage = true
			}
			if t, ok := m["text"].(string); ok {
				texts = append(texts, t)
			}
		}
		return strings.Join(texts, "\n"), hasImage
	default:
		return "", false
	}
}

func byokLastUserText(messages []any) string {
	for i := len(messages) - 1; i >= 0; i-- {
		item, ok := messages[i].(map[string]any)
		if !ok {
			continue
		}
		role, _ := item["role"].(string)
		if role != "user" {
			continue
		}
		text, _ := byokFlattenMessageContent(item["content"])
		return text
	}
	return ""
}

func byokMessagesHaveImage(messages []any) bool {
	for _, raw := range messages {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		_, hasImage := byokFlattenMessageContent(item["content"])
		if hasImage {
			return true
		}
	}
	return false
}

// ByokClassifyChatIntent mirrors portal classify_chat_intent.
func ByokClassifyChatIntent(payload map[string]any) string {
	messages, _ := payload["messages"].([]any)
	if byokMessagesHaveImage(messages) {
		return "vision"
	}
	probe := byokLastUserText(messages)
	if byokReasonHintRE.MatchString(probe) {
		return "reason"
	}
	if byokCodeHintRE.MatchString(probe) {
		return "code"
	}
	if payload["tools"] != nil || payload["functions"] != nil {
		return "agent"
	}
	total := 0
	for _, raw := range messages {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		text, _ := byokFlattenMessageContent(item["content"])
		total += len(text)
	}
	if total >= ByokLongChatChars {
		return "long"
	}
	return "chat"
}

func byokIntentVoucherOrder(voucher []string, intent string) []string {
	preferred := byokIntentVoucher[intent]
	if preferred == nil {
		preferred = byokIntentVoucher["chat"]
	}
	inVoucher := map[string]struct{}{}
	for _, n := range voucher {
		inVoucher[n] = struct{}{}
	}
	ordered := make([]string, 0, len(voucher))
	seen := map[string]struct{}{}
	for _, name := range preferred {
		if _, ok := inVoucher[name]; ok {
			if _, dup := seen[name]; !dup {
				ordered = append(ordered, name)
				seen[name] = struct{}{}
			}
		}
	}
	for _, name := range voucher {
		if _, dup := seen[name]; dup {
			continue
		}
		ordered = append(ordered, name)
		seen[name] = struct{}{}
	}
	return ordered
}

// ByokAutoChatCandidates returns try list (voucher then balance) and intent.
func ByokAutoChatCandidates(names []string, payload map[string]any) (candidates []string, intent string) {
	ranked := ByokRankChatModels(names)
	voucher := make([]string, 0)
	balance := make([]string, 0)
	for _, item := range ranked {
		if _, ok := byokSiliconVoucherChat[item]; ok {
			voucher = append(voucher, item)
		} else {
			balance = append(balance, item)
		}
	}
	intent = ByokClassifyChatIntent(payload)
	voucher = byokIntentVoucherOrder(voucher, intent)
	candidates = append(
		ByokPickUniqueModels(voucher, ByokAutoVoucherTries),
		ByokPickUniqueModels(balance, ByokAutoBalanceTries)...,
	)
	return candidates, intent
}

// ByokShouldFailover mirrors portal should_failover.
func ByokShouldFailover(status int, body []byte) bool {
	if status == 429 || status == 503 || status == 404 {
		return true
	}
	if status != 400 && status != 402 && status != 403 {
		return false
	}
	lowered := strings.ToLower(string(body))
	for _, token := range byokFailoverNeedles {
		if strings.Contains(lowered, strings.ToLower(token)) {
			return true
		}
	}
	return false
}
