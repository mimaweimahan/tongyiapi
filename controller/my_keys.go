package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// byokProvider describes one official-vendor channel template for the
// single-facade "My Keys" page (DEV_FEATURES §3.2).
type byokProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Models   string `json:"models"`
	KeyHint  string `json:"key_hint"`
	NeedBase bool   `json:"need_base"`
	BaseHint string `json:"base_hint,omitempty"`
	Other    string `json:"other,omitempty"`
	// KeyURL is the official console page for creating/copying an API key
	// (not a pricing page). Empty means the UI must not render a link.
	KeyURL string `json:"key_url,omitempty"`
	// LogoURL overrides /logos/{id}.svg when set.
	LogoURL string `json:"logo_url,omitempty"`
	// PricingTier is free|paid from byok_providers.
	PricingTier string `json:"pricing_tier"`
	// CanConnect is true for free vendors, or paid vendors with an active entitlement.
	CanConnect bool `json:"can_connect"`
}

// byokProviderCodeFallback is the first-boot seed and emergency fallback
// when byok_providers is empty. Runtime catalog is DB-backed.
var byokProviderCodeFallback = []byokProvider{
	{ID: "openai", Name: "OpenAI", Type: constant.ChannelTypeOpenAI, Models: "gpt-4o-mini,gpt-4o", KeyHint: "sk-...", KeyURL: "https://platform.openai.com/api-keys"},
	{ID: "azure", Name: "Azure OpenAI", Type: constant.ChannelTypeAzure, Models: "", KeyHint: "密钥", NeedBase: true, BaseHint: "https://资源名.openai.azure.com", Other: "2024-10-21", KeyURL: "https://portal.azure.com/"},
	{ID: "anthropic", Name: "Anthropic", Type: constant.ChannelTypeAnthropic, Models: "claude-sonnet-4-5,claude-3-5-haiku-latest", KeyHint: "sk-ant-...", KeyURL: "https://console.anthropic.com/settings/keys"},
	{ID: "gemini", Name: "Gemini", Type: constant.ChannelTypeGemini, Models: "gemini-2.5-flash,gemini-2.5-pro", KeyHint: "AIza...", KeyURL: "https://aistudio.google.com/apikey"},
	{ID: "deepseek", Name: "DeepSeek", Type: constant.ChannelTypeDeepSeek, Models: "deepseek-chat,deepseek-reasoner", KeyHint: "sk-...", KeyURL: "https://platform.deepseek.com/api_keys"},
	{ID: "ali", Name: "阿里百炼", Type: constant.ChannelTypeAli, Models: "qwen-plus,qwen-turbo", KeyHint: "sk-...", KeyURL: "https://bailian.console.aliyun.com/?tab=model#/api-key"},
	{ID: "moonshot", Name: "月之暗面", Type: constant.ChannelTypeMoonshot, Models: "moonshot-v1-8k,moonshot-v1-32k,moonshot-v1-128k", KeyHint: "sk-...", KeyURL: "https://platform.moonshot.cn/console/api-keys"},
	{ID: "zhipu", Name: "智谱", Type: constant.ChannelTypeZhipu_v4, Models: "glm-4-flash,glm-4-plus", KeyHint: "密钥", KeyURL: "https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys"},
	{ID: "baidu", Name: "百度千帆", Type: constant.ChannelTypeBaiduV2, Models: "ernie-4.0-8k", KeyHint: "密钥", KeyURL: "https://console.bce.baidu.com/qianfan/ais/console/apiKey"},
	{ID: "tencent", Name: "腾讯混元", Type: constant.ChannelTypeTencent, Models: "hunyuan-lite,hunyuan-standard", KeyHint: "SecretId|SecretKey", KeyURL: "https://console.cloud.tencent.com/cam/capi"},
	{ID: "volc", Name: "火山方舟", Type: constant.ChannelTypeVolcEngine, Models: "", KeyHint: "密钥", NeedBase: true, BaseHint: "https://ark.cn-beijing.volces.com", KeyURL: "https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey"},
	{ID: "silicon", Name: "硅基流动", Type: constant.ChannelTypeSiliconFlow, Models: "deepseek-ai/DeepSeek-V3", KeyHint: "sk-...", KeyURL: "https://cloud.siliconflow.cn/account/ak"},
	{ID: "minimax", Name: "MiniMax", Type: constant.ChannelTypeMiniMax, Models: "MiniMax-Text-01", KeyHint: "密钥", KeyURL: "https://platform.minimaxi.com/user-center/basic-information/interface-key"},
}

func byokFromModel(row model.ByokProvider) byokProvider {
	tier := model.NormalizeByokPricingTier(row.PricingTier)
	return byokProvider{
		ID:          row.ID,
		Name:        row.Name,
		Type:        row.ChannelType,
		Models:      row.Models,
		KeyHint:     row.KeyHint,
		NeedBase:    row.NeedBase,
		BaseHint:    row.BaseHint,
		Other:       row.Other,
		KeyURL:      row.KeyURL,
		LogoURL:     row.LogoURL,
		PricingTier: tier,
		CanConnect:  true, // filled per-user in byokAttachConnectFlags
	}
}

func byokAttachConnectFlags(userID int, list []byokProvider) []byokProvider {
	out := make([]byokProvider, len(list))
	for i, p := range list {
		out[i] = p
		out[i].PricingTier = model.NormalizeByokPricingTier(p.PricingTier)
		out[i].CanConnect = model.CanConnectByokProvider(userID, out[i].PricingTier, p.ID)
	}
	return out
}

func byokRequireCanConnect(c *gin.Context, userID int, provider byokProvider) bool {
	tier := model.NormalizeByokPricingTier(provider.PricingTier)
	if model.CanConnectByokProvider(userID, tier, provider.ID) {
		return true
	}
	common.ApiErrorMsg(c, "该厂商需开通后才能接入；开通后仍使用你自己的官方 API Key，本站不代持转卖")
	return false
}

func byokLoadEnabled() ([]byokProvider, map[string]byokProvider) {
	rows, err := model.ListEnabledByokProviders()
	list := make([]byokProvider, 0)
	if err == nil && len(rows) > 0 {
		for _, row := range rows {
			list = append(list, byokFromModel(row))
		}
	} else {
		list = append(list, byokProviderCodeFallback...)
	}
	byID := make(map[string]byokProvider, len(list))
	for _, p := range list {
		byID[p.ID] = p
	}
	return list, byID
}

func byokResolveEnabled(id string) (byokProvider, bool) {
	_, byID := byokLoadEnabled()
	p, ok := byID[strings.TrimSpace(id)]
	return p, ok
}

func byokResolveKnown(id string) (byokProvider, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return byokProvider{}, false
	}
	row, err := model.GetByokProvider(id)
	if err == nil {
		return byokFromModel(*row), true
	}
	for _, p := range byokProviderCodeFallback {
		if p.ID == id {
			return p, true
		}
	}
	return byokProvider{}, false
}

func byokChannelName(userID int, providerID string) string {
	return fmt.Sprintf("%s-%s", model.PersonalGroupName(userID), providerID)
}

func byokNormalizeModels(raw string, fallback string) string {
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
	if len(out) == 0 {
		return strings.TrimSpace(fallback)
	}
	return strings.Join(out, ",")
}

func byokKeyHint(secret string) string {
	s := strings.TrimSpace(secret)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= 8 {
		return "••••"
	}
	return string(runes[:4]) + "…" + string(runes[len(runes)-4:])
}

func byokFindChannelByName(name string) (*model.Channel, error) {
	var channel model.Channel
	err := model.DB.Where("name = ?", name).First(&channel).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

type myKeyLinkDTO struct {
	Provider   string   `json:"provider"`
	ChannelID  int      `json:"channel_id"`
	Name       string   `json:"name"`
	Group      string   `json:"group"`
	Models     string   `json:"models"`
	ChatModels []string `json:"chat_models"`
	BaseURL    string   `json:"base_url,omitempty"`
	KeyHint    string   `json:"key_hint"`
	Status     int      `json:"status"`
}

type saveMyKeyRequest struct {
	Provider string `json:"provider"`
	Secret   string `json:"secret"`
	Models   string `json:"models"`
	BaseURL  string `json:"base_url"`
}

type fetchMyKeyModelsRequest struct {
	Provider string `json:"provider"`
	Secret   string `json:"secret"`
	BaseURL  string `json:"base_url"`
}

// GetMyKeys returns the official provider catalog plus this user's linked channels.
func GetMyKeys(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "unauthorized")
		return
	}
	group := model.PersonalGroupName(userID)
	providers, _ := byokLoadEnabled()
	providers = byokAttachConnectFlags(userID, providers)
	links := make(map[string]myKeyLinkDTO)
	for _, provider := range providers {
		name := byokChannelName(userID, provider.ID)
		channel, err := byokFindChannelByName(name)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if channel == nil {
			continue
		}
		// Only expose channels that still belong to this personal group.
		if strings.TrimSpace(channel.Group) != group {
			continue
		}
		baseURL := ""
		if channel.BaseURL != nil {
			baseURL = *channel.BaseURL
		}
		links[provider.ID] = myKeyLinkDTO{
			Provider:   provider.ID,
			ChannelID:  channel.Id,
			Name:       channel.Name,
			Group:      channel.Group,
			Models:     channel.Models,
			ChatModels: byokRankChatModels(byokSplitModels(channel.Models)),
			BaseURL:    baseURL,
			KeyHint:    byokKeyHint(channel.Key),
			Status:     channel.Status,
		}
	}
	common.ApiSuccess(c, gin.H{
		"providers": providers,
		"links":     links,
		"group":     group,
	})
}

// SaveMyKey creates or updates the user's official-vendor channel.
// Channel name is forced to u{id}-{provider}; group is forced to u{id}.
func SaveMyKey(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "unauthorized")
		return
	}
	var req saveMyKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	provider, ok := byokResolveEnabled(strings.TrimSpace(req.Provider))
	if !ok {
		common.ApiErrorMsg(c, "unsupported provider")
		return
	}
	if !byokRequireCanConnect(c, userID, provider) {
		return
	}
	secret := strings.TrimSpace(req.Secret)
	models := byokNormalizeModels(req.Models, provider.Models)
	baseURL := strings.TrimSpace(req.BaseURL)
	group := model.PersonalGroupName(userID)
	name := byokChannelName(userID, provider.ID)

	existing, err := byokFindChannelByName(name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if existing == nil && secret == "" {
		common.ApiErrorMsg(c, "secret is required for the first save")
		return
	}
	if models == "" {
		common.ApiErrorMsg(c, "at least one model is required")
		return
	}
	if provider.NeedBase && provider.ID == "azure" && baseURL == "" && existing == nil {
		common.ApiErrorMsg(c, "azure base_url is required")
		return
	}

	model.EnsurePersonalGroupRatio(group)

	if existing == nil {
		channel := &model.Channel{
			Type:        provider.Type,
			Key:         secret,
			Name:        name,
			Models:      models,
			Group:       group,
			Status:      common.ChannelStatusEnabled,
			CreatedTime: common.GetTimestamp(),
			Other:       provider.Other,
		}
		if baseURL != "" {
			channel.BaseURL = &baseURL
		}
		weight := uint(0)
		priority := int64(0)
		autoBan := 1
		channel.Weight = &weight
		channel.Priority = &priority
		channel.AutoBan = &autoBan
		if err := channel.Insert(); err != nil {
			common.ApiError(c, err)
			return
		}
		model.InitChannelCache()
		common.ApiSuccess(c, gin.H{
			"provider":   provider.ID,
			"channel_id": channel.Id,
			"name":       channel.Name,
			"group":      channel.Group,
			"key_hint":   byokKeyHint(secret),
		})
		return
	}

	if strings.TrimSpace(existing.Group) != group {
		common.ApiErrorMsg(c, "channel ownership mismatch")
		return
	}
	existing.Type = provider.Type
	existing.Name = name
	existing.Group = group
	existing.Models = models
	if provider.Other != "" {
		existing.Other = provider.Other
	}
	if secret != "" {
		existing.Key = secret
	}
	if baseURL != "" {
		existing.BaseURL = &baseURL
	}
	if err := existing.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	hint := byokKeyHint(secret)
	if hint == "" {
		hint = byokKeyHint(existing.Key)
	}
	common.ApiSuccess(c, gin.H{
		"provider":   provider.ID,
		"channel_id": existing.Id,
		"name":       existing.Name,
		"group":      existing.Group,
		"key_hint":   hint,
	})
}

// DeleteMyKey removes the user's channel for a provider id.
func DeleteMyKey(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "unauthorized")
		return
	}
	providerID := strings.TrimSpace(c.Param("provider"))
	if _, ok := byokResolveKnown(providerID); !ok {
		common.ApiErrorMsg(c, "unsupported provider")
		return
	}
	group := model.PersonalGroupName(userID)
	name := byokChannelName(userID, providerID)
	existing, err := byokFindChannelByName(name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if existing == nil {
		common.ApiSuccess(c, gin.H{"deleted": false})
		return
	}
	if strings.TrimSpace(existing.Group) != group {
		common.ApiErrorMsg(c, "channel ownership mismatch")
		return
	}
	if err := existing.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	common.ApiSuccess(c, gin.H{"deleted": true, "channel_id": existing.Id})
}

// FetchMyKeyModels pulls upstream model ids using a plaintext vendor key
// supplied by the client (same rule as the portal: must re-enter the secret).
func FetchMyKeyModels(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "unauthorized")
		return
	}
	var req fetchMyKeyModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	provider, ok := byokResolveEnabled(strings.TrimSpace(req.Provider))
	if !ok {
		common.ApiErrorMsg(c, "unsupported provider")
		return
	}
	if !byokRequireCanConnect(c, userID, provider) {
		return
	}
	secret := strings.TrimSpace(req.Secret)
	if secret == "" {
		common.ApiErrorMsg(c, "secret is required to fetch models")
		return
	}
	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL == "" {
		baseURL = constant.GetChannelBaseURL(provider.Type)
	}
	channel := &model.Channel{
		Type:    provider.Type,
		Key:     strings.Split(secret, "\n")[0],
		BaseURL: &baseURL,
		Other:   provider.Other,
	}
	models, err := fetchChannelUpstreamModelIDs(channel)
	if err != nil {
		common.ApiErrorMsg(c, fmt.Sprintf("failed to fetch models: %s", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
}

type testMyKeyRequest struct {
	Provider string `json:"provider"`
	Secret   string `json:"secret"`
	BaseURL  string `json:"base_url"`
	Models   string `json:"models"`
}

type byokTestCacheEntry struct {
	At      time.Time
	OK      bool
	Data    gin.H
	Message string
}

var byokTestCooldown sync.Map // fingerprint -> byokTestCacheEntry

func byokTestFingerprint(userID int, providerID, secret, baseURL string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s", userID, providerID, secret, baseURL)))
	return hex.EncodeToString(sum[:12])
}

func byokFormatProbeError(err error) string {
	if err == nil {
		return "probe failed"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "status code: 401"):
		return "厂商拒绝：密钥无效或未授权 (401)"
	case strings.Contains(msg, "status code: 403"):
		return "厂商拒绝：权限不足 (403)"
	case strings.Contains(msg, "status code: 404"):
		return "厂商接口不存在或 Base URL 不正确 (404)"
	case strings.Contains(msg, "status code: 429"):
		return "厂商限流，请稍后再试 (429)"
	}
	if idx := strings.Index(msg, "status code: "); idx >= 0 {
		code := strings.TrimSpace(msg[idx+len("status code: "):])
		if space := strings.IndexAny(code, " \n\t"); space > 0 {
			code = code[:space]
		}
		return fmt.Sprintf("厂商返回错误 (HTTP %s)", code)
	}
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return "连接厂商超时"
	case strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "network is unreachable") ||
		strings.Contains(lower, "dial tcp"):
		return "无法连接厂商网络"
	default:
		if len(msg) > 180 {
			msg = msg[:180] + "…"
		}
		return "探测失败：" + msg
	}
}

func byokProbeSample(models []string, n int) []string {
	if n <= 0 || len(models) == 0 {
		return []string{}
	}
	if len(models) <= n {
		out := make([]string, len(models))
		copy(out, models)
		return out
	}
	out := make([]string, n)
	copy(out, models[:n])
	return out
}

// TestMyKey probes a plaintext vendor key without persisting a channel.
// Reuses fetchChannelUpstreamModelIDs (same path as fetch_models). Does NOT
// expose admin TestChannel to end users.
func TestMyKey(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorMsg(c, "unauthorized")
		return
	}
	var req testMyKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	provider, ok := byokResolveEnabled(strings.TrimSpace(req.Provider))
	if !ok {
		common.ApiErrorMsg(c, "unsupported provider")
		return
	}
	if !byokRequireCanConnect(c, userID, provider) {
		return
	}
	secret := strings.TrimSpace(req.Secret)
	if secret == "" {
		common.ApiErrorMsg(c, "secret is required")
		return
	}
	baseURL := strings.TrimSpace(req.BaseURL)
	if provider.NeedBase && baseURL == "" {
		common.ApiErrorMsg(c, "base_url is required for this provider")
		return
	}

	fp := byokTestFingerprint(userID, provider.ID, secret, baseURL)
	if raw, hit := byokTestCooldown.Load(fp); hit {
		if entry, ok := raw.(byokTestCacheEntry); ok && time.Since(entry.At) < 2*time.Second {
			if entry.OK {
				common.ApiSuccess(c, entry.Data)
			} else {
				common.ApiErrorMsg(c, entry.Message)
			}
			return
		}
	}

	if baseURL == "" {
		baseURL = constant.GetChannelBaseURL(provider.Type)
	}
	channel := &model.Channel{
		Type:    provider.Type,
		Key:     strings.Split(secret, "\n")[0],
		BaseURL: &baseURL,
		Other:   provider.Other,
	}
	// Optional models field is accepted for forward-compat but unused for probe.
	_ = req.Models

	models, err := fetchChannelUpstreamModelIDs(channel)
	if err != nil {
		msg := byokFormatProbeError(err)
		byokTestCooldown.Store(fp, byokTestCacheEntry{
			At:      time.Now(),
			OK:      false,
			Message: msg,
		})
		common.ApiErrorMsg(c, msg)
		return
	}
	data := gin.H{
		"ok":           true,
		"models_count": len(models),
		"sample":       byokProbeSample(models, 8),
		"models":       models,
	}
	byokTestCooldown.Store(fp, byokTestCacheEntry{
		At:   time.Now(),
		OK:   true,
		Data: data,
	})
	common.ApiSuccess(c, data)
}
