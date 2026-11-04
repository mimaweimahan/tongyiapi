package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const byokInternalHeader = "X-Byok-Internal"

var byokHopHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailers":            {},
	"transfer-encoding":   {},
	"upgrade":             {},
	"content-length":      {},
}

func byokListenPort() string {
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		return p
	}
	if common.Port != nil && *common.Port > 0 {
		return strconv.Itoa(*common.Port)
	}
	return "3002"
}

func byokPeekChatModel(c *gin.Context) (string, map[string]any, error) {
	var payload map[string]any
	if err := common.UnmarshalBodyReusable(c, &payload); err != nil {
		return "", nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	modelName, _ := payload["model"].(string)
	return strings.TrimSpace(modelName), payload, nil
}

// byokCollectTokenChatModels builds the auto candidate pool.
// Prefer token model_limits (vendor+auto lock). Otherwise use enabled models for
// the personal group. Do NOT apply HasModelBillingConfig here — BYOK routes on
// the user's own vendor keys; missing local price config must not empty the pool.
func byokCollectTokenChatModels(c *gin.Context) []string {
	modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	if modelLimitEnable {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		if ok {
			if tokenModelLimit, ok := s.(map[string]bool); ok && len(tokenModelLimit) > 0 {
				names := make([]string, 0, len(tokenModelLimit))
				for name, allowed := range tokenModelLimit {
					if allowed {
						names = append(names, name)
					}
				}
				return service.ByokRankChatModels(names)
			}
		}
	}

	groups, err := getModelListGroups(c)
	if err != nil {
		return nil
	}
	return service.ByokRankChatModels(service.GetGroupsEnabledModels(groups.ownerGroups))
}

func byokWriteOpenAIError(c *gin.Context, status int, message, code string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "invalid_request_error",
			"code":    code,
		},
	})
}

// byokInjectAutoModels prepends auto/auto-chat and keeps chat models ordered (portal proxy_models).
func byokInjectAutoModels(models []dto.OpenAIModels) []dto.OpenAIModels {
	created := int(time.Now().Unix())
	byID := make(map[string]dto.OpenAIModels, len(models))
	ids := make([]string, 0, len(models))
	for _, item := range models {
		if item.Id == "" {
			continue
		}
		byID[item.Id] = item
		ids = append(ids, item.Id)
	}
	out := make([]dto.OpenAIModels, 0, len(models)+2)
	for _, name := range []string{"auto", "auto-chat"} {
		if item, ok := byID[name]; ok {
			out = append(out, item)
			continue
		}
		out = append(out, dto.OpenAIModels{
			Id:      name,
			Object:  "model",
			Created: created,
			OwnedBy: "byok",
		})
	}
	for _, name := range service.ByokRankChatModels(ids) {
		if service.ByokIsAutoAlias(name) {
			continue
		}
		if item, ok := byID[name]; ok {
			out = append(out, item)
		}
	}
	return out
}

func byokSyntheticAutoModel(modelId string) (dto.OpenAIModels, bool) {
	if !service.ByokIsAutoAlias(modelId) {
		return dto.OpenAIModels{}, false
	}
	return dto.OpenAIModels{
		Id:      strings.TrimSpace(strings.ToLower(modelId)),
		Object:  "model",
		Created: int(time.Now().Unix()),
		OwnedBy: "byok",
	}, true
}

func byokCopyResponseHeaders(dst http.Header, src http.Header, extra map[string]string) {
	for key, values := range src {
		lk := strings.ToLower(key)
		if _, skip := byokHopHeaders[lk]; skip {
			continue
		}
		if strings.HasPrefix(lk, "x-byok-") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	for key, value := range extra {
		dst.Set(key, value)
	}
}

func byokRelayChatOnce(c *gin.Context) {
	middleware.Distribute()(c)
	if c.IsAborted() {
		return
	}
	Relay(c, types.RelayFormatOpenAI)
}

// ByokChatCompletions handles /v1/chat/completions with portal-compatible auto routing.
// Non-auto requests go through Distribute + Relay. Auto uses loopback with rewritten model
// so each attempt reuses the normal channel selection path (same-round no mid-stream swap).
func ByokChatCompletions(c *gin.Context) {
	modelName, payload, err := byokPeekChatModel(c)
	if err != nil {
		byokWriteOpenAIError(c, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}

	if c.GetHeader(byokInternalHeader) == "1" && service.ByokIsAutoAlias(modelName) {
		byokWriteOpenAIError(c, http.StatusBadRequest, "internal auto loop detected", "invalid_request_error")
		return
	}

	if !service.ByokIsAutoAlias(modelName) {
		byokRelayChatOnce(c)
		return
	}

	candidates, intent := service.ByokAutoChatCandidates(byokCollectTokenChatModels(c), payload)
	if len(candidates) == 0 {
		byokWriteOpenAIError(c, http.StatusBadRequest,
			"没有可用的对话模型。先在厂商密钥里接入至少一家，再把模型填成 auto。",
			"no_chat_model")
		return
	}

	auth := c.GetHeader("Authorization")
	if auth == "" {
		auth = c.GetHeader("authorization")
	}
	client := &http.Client{Timeout: 120 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%s/v1/chat/completions", byokListenPort())

	tried := make([]string, 0, len(candidates))
	var lastStatus int
	var lastReason string
	var lastPayload []byte
	var lastHeader http.Header

	for _, candidate := range candidates {
		tried = append(tried, candidate)
		nextPayload := make(map[string]any, len(payload)+1)
		for k, v := range payload {
			nextPayload[k] = v
		}
		nextPayload["model"] = candidate
		body, marshalErr := common.Marshal(nextPayload)
		if marshalErr != nil {
			byokWriteOpenAIError(c, http.StatusInternalServerError, marshalErr.Error(), "server_error")
			return
		}

		req, reqErr := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(body))
		if reqErr != nil {
			byokWriteOpenAIError(c, http.StatusInternalServerError, reqErr.Error(), "server_error")
			return
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(byokInternalHeader, "1")
		if rid := c.GetHeader("X-Request-Id"); rid != "" {
			req.Header.Set("X-Request-Id", rid)
		}
		if ua := c.GetHeader("User-Agent"); ua != "" {
			req.Header.Set("User-Agent", ua)
		}

		resp, doErr := client.Do(req)
		if doErr != nil {
			lastStatus = http.StatusBadGateway
			lastReason = "Bad Gateway"
			lastPayload = []byte(fmt.Sprintf(`{"error":{"message":%q,"type":"api_error","code":"upstream_error"}}`, doErr.Error()))
			lastHeader = http.Header{}
			continue
		}

		extra := map[string]string{
			"X-Byok-Model":  candidate,
			"X-Byok-Tried":  strings.Join(tried, ","),
			"X-Byok-Intent": intent,
		}

		if resp.StatusCode < 400 {
			byokCopyResponseHeaders(c.Writer.Header(), resp.Header, extra)
			c.Status(resp.StatusCode)
			_, _ = io.Copy(c.Writer, resp.Body)
			_ = resp.Body.Close()
			return
		}

		errBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if !service.ByokShouldFailover(resp.StatusCode, errBody) {
			byokCopyResponseHeaders(c.Writer.Header(), resp.Header, extra)
			c.Status(resp.StatusCode)
			if len(errBody) > 0 {
				_, _ = c.Writer.Write(errBody)
			}
			return
		}

		lastStatus = resp.StatusCode
		lastReason = resp.Status
		lastPayload = errBody
		lastHeader = resp.Header.Clone()
	}

	extra := map[string]string{
		"X-Byok-Tried":  strings.Join(tried, ","),
		"X-Byok-Intent": intent,
	}
	if lastStatus == 0 {
		lastStatus = http.StatusBadGateway
		lastPayload = []byte(`{"error":{"message":"auto routing failed","type":"api_error","code":"auto_failed"}}`)
	}
	_ = lastReason
	byokCopyResponseHeaders(c.Writer.Header(), lastHeader, extra)
	c.Status(lastStatus)
	if len(lastPayload) > 0 {
		_, _ = c.Writer.Write(lastPayload)
	}
}
