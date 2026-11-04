package controller

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// byokChatSkipRE mirrors portal CHAT_SKIP_RE: skip non-chat models when
// locking a vendor+auto site token (DEV_FEATURES §3.3).
var byokChatSkipRE = regexp.MustCompile(`(?i)(embed|bge|rerank|asr|tts|whisper|image|flux|kolors|video|audio|sensevoice|cosyvoice|ocr|speech|live)`)

var byokAutoAliases = map[string]struct{}{
	"auto":      {},
	"auto-chat": {},
}

func byokIsAutoAlias(name string) bool {
	_, ok := byokAutoAliases[strings.TrimSpace(strings.ToLower(name))]
	return ok
}

func byokIsAllProviders(providerID string) bool {
	id := strings.TrimSpace(strings.ToLower(providerID))
	return id == "" || id == "all" || id == "auto"
}

func byokIsChatModel(name string) bool {
	item := strings.TrimSpace(name)
	if item == "" || byokIsAutoAlias(item) {
		return false
	}
	return !byokChatSkipRE.MatchString(item)
}

func byokSplitModels(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// byokRankChatModels keeps chat-capable models only (portal rank_chat_models
// without voucher reordering — P4 owns routing priority).
func byokRankChatModels(names []string) []string {
	out := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		item := strings.TrimSpace(name)
		if !byokIsChatModel(item) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func byokPublicAPIBase(c *gin.Context) string {
	addr := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if addr == "" || strings.Contains(addr, "localhost") || strings.Contains(addr, "127.0.0.1") {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		if proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); proto != "" {
			scheme = strings.Split(proto, ",")[0]
			scheme = strings.TrimSpace(scheme)
		}
		host := strings.TrimSpace(c.Request.Host)
		if host == "" {
			host = "127.0.0.1:3002"
		}
		addr = scheme + "://" + host
	}
	return addr + "/v1"
}

func byokFormatSiteKey(raw string) string {
	key := strings.TrimSpace(raw)
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "sk-") {
		return key
	}
	return "sk-" + key
}

type createSiteTokenRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Name     string `json:"name"`
}

// CreateSiteToken creates a personal-group site token with BYOK model-lock rules.
// §3.3: vendor+auto locks that channel's chat models; all+auto leaves limits empty.
func CreateSiteToken(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "unauthorized")
		return
	}
	var req createSiteTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	providerID := strings.TrimSpace(req.Provider)
	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = "auto"
	}
	tokenName := strings.TrimSpace(req.Name)
	if tokenName == "" {
		tokenName = "site"
	}
	if len(tokenName) > 50 {
		common.ApiErrorMsg(c, "token name is too long")
		return
	}

	group := model.PersonalGroupName(userID)
	providers, _ := byokLoadEnabled()
	links := make(map[string]*model.Channel)
	for _, provider := range providers {
		channel, err := byokFindChannelByName(byokChannelName(userID, provider.ID))
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if channel == nil {
			continue
		}
		if strings.TrimSpace(channel.Group) != group {
			continue
		}
		links[provider.ID] = channel
	}
	if len(links) == 0 {
		common.ApiErrorMsg(c, "connect at least one vendor key first")
		return
	}

	var restrict string
	scope := "all"
	resolvedProvider := "all"
	displayModel := modelName

	if byokIsAllProviders(providerID) {
		if !byokIsAutoAlias(modelName) {
			common.ApiErrorMsg(c, "when using all connected vendors, model must be auto")
			return
		}
		restrict = ""
		scope = "all"
		resolvedProvider = "all"
		displayModel = "auto"
	} else {
		if _, ok := byokResolveEnabled(providerID); !ok {
			common.ApiErrorMsg(c, "unsupported provider")
			return
		}
		channel, ok := links[providerID]
		if !ok {
			common.ApiErrorMsg(c, "this vendor key is not connected yet")
			return
		}
		allowed := byokRankChatModels(byokSplitModels(channel.Models))
		if byokIsAutoAlias(modelName) {
			if len(allowed) == 0 {
				common.ApiErrorMsg(c, "this vendor has no chat models")
				return
			}
			restrict = strings.Join(allowed, ",")
			displayModel = "auto"
		} else {
			found := false
			for _, item := range allowed {
				if item == modelName {
					found = true
					break
				}
			}
			if !found {
				common.ApiErrorMsg(c, "model is not in this vendor's connected chat list")
				return
			}
			restrict = modelName
			displayModel = modelName
		}
		scope = "vendor"
		resolvedProvider = providerID
	}

	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		common.ApiErrorMsg(c, fmt.Sprintf("maximum number of tokens reached (%d)", maxTokens))
		return
	}

	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorMsg(c, "failed to generate token")
		common.SysLog(common.LogText("failed to generate site token key: %s", err.Error()))
		return
	}

	model.EnsurePersonalGroupRatio(group)

	cleanToken := model.Token{
		UserId:              userID,
		Name:                tokenName,
		Key:                 key,
		Status:              common.TokenStatusEnabled,
		CreatedTime:         common.GetTimestamp(),
		AccessedTime:        common.GetTimestamp(),
		ExpiredTime:         -1,
		RemainQuota:         0,
		UnlimitedQuota:      true,
		ModelLimitsEnabled:  restrict != "",
		ModelLimits:         restrict,
		Group:               group,
		AllowUserQuotaQuery: true,
	}
	if err := cleanToken.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":       cleanToken.Id,
		"name":     cleanToken.Name,
		"key":      byokFormatSiteKey(cleanToken.Key),
		"model":    displayModel,
		"scope":    scope,
		"provider": resolvedProvider,
		"group":    group,
		"base_url": byokPublicAPIBase(c),
		"models":   restrict,
	})
}
