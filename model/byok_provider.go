package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

// ByokProvider is the admin-configurable catalog of official vendors
// that users may attach personal keys to on /my-keys.
// This is NOT the shared channel pool under /channels.
type ByokProvider struct {
	ID                 string `json:"id" gorm:"primaryKey;size:64"`
	Name               string `json:"name" gorm:"size:120;not null"`
	ChannelType        int    `json:"channel_type" gorm:"column:channel_type;not null"`
	Models             string `json:"models" gorm:"type:text"`
	KeyHint            string `json:"key_hint" gorm:"size:64"`
	NeedBase           bool   `json:"need_base" gorm:"not null;default:false"`
	BaseHint           string `json:"base_hint" gorm:"size:255"`
	Other              string `json:"other" gorm:"size:128"`
	KeyURL             string `json:"key_url" gorm:"size:512"`
	LogoURL            string `json:"logo_url" gorm:"size:512"`
	Enabled            bool   `json:"enabled" gorm:"not null;default:true;index"`
	Sort               int    `json:"sort" gorm:"not null;default:0;index"`
	IsOpenAICompatible bool   `json:"is_openai_compatible" gorm:"not null;default:false"`
	// PricingTier: free = any user may attach their own key;
	// paid = requires an active byok_provider_entitlements row first (scheme B).
	PricingTier string `json:"pricing_tier" gorm:"size:16;not null;default:free;index"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;not null;default:0"`
}

func (ByokProvider) TableName() string { return "byok_providers" }

var byokProviderIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

type byokProviderSeed struct {
	ID                 string
	Name               string
	ChannelType        int
	Models             string
	KeyHint            string
	NeedBase           bool
	BaseHint           string
	Other              string
	KeyURL             string
	IsOpenAICompatible bool
	Sort               int
}

// Default seed catalog (13 vendors). Used on first boot and as code fallback.
var byokProviderSeeds = []byokProviderSeed{
	{ID: "openai", Name: "OpenAI", ChannelType: constant.ChannelTypeOpenAI, Models: "gpt-4o-mini,gpt-4o", KeyHint: "sk-...", KeyURL: "https://platform.openai.com/api-keys", Sort: 10},
	{ID: "azure", Name: "Azure OpenAI", ChannelType: constant.ChannelTypeAzure, Models: "", KeyHint: "密钥", NeedBase: true, BaseHint: "https://资源名.openai.azure.com", Other: "2024-10-21", KeyURL: "https://portal.azure.com/", Sort: 20},
	{ID: "anthropic", Name: "Anthropic", ChannelType: constant.ChannelTypeAnthropic, Models: "claude-sonnet-4-5,claude-3-5-haiku-latest", KeyHint: "sk-ant-...", KeyURL: "https://console.anthropic.com/settings/keys", Sort: 30},
	{ID: "gemini", Name: "Gemini", ChannelType: constant.ChannelTypeGemini, Models: "gemini-2.5-flash,gemini-2.5-pro", KeyHint: "AIza...", KeyURL: "https://aistudio.google.com/apikey", Sort: 40},
	{ID: "deepseek", Name: "DeepSeek", ChannelType: constant.ChannelTypeDeepSeek, Models: "deepseek-chat,deepseek-reasoner", KeyHint: "sk-...", KeyURL: "https://platform.deepseek.com/api_keys", Sort: 50},
	{ID: "ali", Name: "阿里百炼", ChannelType: constant.ChannelTypeAli, Models: "qwen-plus,qwen-turbo", KeyHint: "sk-...", KeyURL: "https://bailian.console.aliyun.com/?tab=model#/api-key", Sort: 60},
	{ID: "moonshot", Name: "月之暗面", ChannelType: constant.ChannelTypeMoonshot, Models: "moonshot-v1-8k,moonshot-v1-32k,moonshot-v1-128k", KeyHint: "sk-...", KeyURL: "https://platform.moonshot.cn/console/api-keys", Sort: 70},
	{ID: "zhipu", Name: "智谱", ChannelType: constant.ChannelTypeZhipu_v4, Models: "glm-4-flash,glm-4-plus", KeyHint: "密钥", KeyURL: "https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys", Sort: 80},
	{ID: "baidu", Name: "百度千帆", ChannelType: constant.ChannelTypeBaiduV2, Models: "ernie-4.0-8k", KeyHint: "密钥", KeyURL: "https://console.bce.baidu.com/qianfan/ais/console/apiKey", Sort: 90},
	{ID: "tencent", Name: "腾讯混元", ChannelType: constant.ChannelTypeTencent, Models: "hunyuan-lite,hunyuan-standard", KeyHint: "SecretId|SecretKey", KeyURL: "https://console.cloud.tencent.com/cam/capi", Sort: 100},
	{ID: "volc", Name: "火山方舟", ChannelType: constant.ChannelTypeVolcEngine, Models: "", KeyHint: "密钥", NeedBase: true, BaseHint: "https://ark.cn-beijing.volces.com", KeyURL: "https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey", Sort: 110},
	{ID: "silicon", Name: "硅基流动", ChannelType: constant.ChannelTypeSiliconFlow, Models: "deepseek-ai/DeepSeek-V3", KeyHint: "sk-...", KeyURL: "https://cloud.siliconflow.cn/account/ak", Sort: 120},
	{ID: "minimax", Name: "MiniMax", ChannelType: constant.ChannelTypeMiniMax, Models: "MiniMax-Text-01", KeyHint: "密钥", KeyURL: "https://platform.minimaxi.com/user-center/basic-information/interface-key", Sort: 130},
}

// EnsureByokProviders inserts missing seed rows. Existing rows are left as-is
// so admin edits remain the source of truth after first boot.
func EnsureByokProviders() error {
	if DB == nil {
		return errors.New("db not ready")
	}
	now := time.Now().Unix()
	for _, item := range byokProviderSeeds {
		var existing ByokProvider
		err := DB.Where("id = ?", item.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := ByokProvider{
			ID:                 item.ID,
			Name:               item.Name,
			ChannelType:        item.ChannelType,
			Models:             item.Models,
			KeyHint:            item.KeyHint,
			NeedBase:           item.NeedBase,
			BaseHint:           item.BaseHint,
			Other:              item.Other,
			KeyURL:             item.KeyURL,
			Enabled:            true,
			Sort:               item.Sort,
			IsOpenAICompatible: item.IsOpenAICompatible,
			PricingTier:        ByokPricingFree,
			UpdatedAt:          now,
		}
		if createErr := DB.Create(&row).Error; createErr != nil {
			return createErr
		}
	}
	// Backfill empty/NULL tiers after AutoMigrate on existing DBs.
	if err := DB.Model(&ByokProvider{}).
		Where("pricing_tier = ? OR pricing_tier IS NULL", "").
		Update("pricing_tier", ByokPricingFree).Error; err != nil {
		return err
	}
	return nil
}

// SeedByokProvidersFallback returns in-code defaults when DB is empty/unavailable.
func SeedByokProvidersFallback() []ByokProvider {
	now := time.Now().Unix()
	out := make([]ByokProvider, 0, len(byokProviderSeeds))
	for _, item := range byokProviderSeeds {
		out = append(out, ByokProvider{
			ID:                 item.ID,
			Name:               item.Name,
			ChannelType:        item.ChannelType,
			Models:             item.Models,
			KeyHint:            item.KeyHint,
			NeedBase:           item.NeedBase,
			BaseHint:           item.BaseHint,
			Other:              item.Other,
			KeyURL:             item.KeyURL,
			Enabled:            true,
			Sort:               item.Sort,
			IsOpenAICompatible: item.IsOpenAICompatible,
			PricingTier:        ByokPricingFree,
			UpdatedAt:          now,
		})
	}
	return out
}

func ListAllByokProviders() ([]ByokProvider, error) {
	var rows []ByokProvider
	err := DB.Order("sort asc, id asc").Find(&rows).Error
	return rows, err
}

func ListEnabledByokProviders() ([]ByokProvider, error) {
	var rows []ByokProvider
	err := DB.Where("enabled = ?", true).Order("sort asc, id asc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return SeedByokProvidersFallback(), nil
	}
	return rows, nil
}

func GetByokProvider(id string) (*ByokProvider, error) {
	id = strings.TrimSpace(id)
	if !byokProviderIDRe.MatchString(id) {
		return nil, errors.New("invalid provider id")
	}
	var row ByokProvider
	if err := DB.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func GetEnabledByokProvider(id string) (*ByokProvider, error) {
	row, err := GetByokProvider(id)
	if err != nil {
		return nil, err
	}
	if !row.Enabled {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

type ByokProviderInput struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ChannelType        int    `json:"channel_type"`
	Models             string `json:"models"`
	KeyHint            string `json:"key_hint"`
	NeedBase           bool   `json:"need_base"`
	BaseHint           string `json:"base_hint"`
	Other              string `json:"other"`
	KeyURL             string `json:"key_url"`
	LogoURL            string `json:"logo_url"`
	Enabled            *bool  `json:"enabled"`
	Sort               *int   `json:"sort"`
	IsOpenAICompatible bool   `json:"is_openai_compatible"`
	PricingTier        string `json:"pricing_tier"`
}

func normalizeByokProviderInput(in *ByokProviderInput, creating bool) error {
	if in == nil {
		return errors.New("empty body")
	}
	in.ID = strings.TrimSpace(strings.ToLower(in.ID))
	in.Name = strings.TrimSpace(in.Name)
	in.Models = strings.TrimSpace(in.Models)
	in.KeyHint = strings.TrimSpace(in.KeyHint)
	in.BaseHint = strings.TrimSpace(in.BaseHint)
	in.Other = strings.TrimSpace(in.Other)
	in.KeyURL = strings.TrimSpace(in.KeyURL)
	in.LogoURL = strings.TrimSpace(in.LogoURL)
	if creating {
		if !byokProviderIDRe.MatchString(in.ID) {
			return errors.New("invalid provider id (use lowercase letters, digits, _ or -)")
		}
	}
	if in.Name == "" {
		return errors.New("name is required")
	}
	if len([]rune(in.Name)) > 80 {
		return errors.New("name too long")
	}
	if in.ChannelType <= 0 {
		if in.IsOpenAICompatible {
			in.ChannelType = constant.ChannelTypeOpenAI
		} else {
			return errors.New("channel_type is required")
		}
	}
	if in.ChannelType >= constant.ChannelTypeDummy {
		return fmt.Errorf("channel_type out of range")
	}
	if in.IsOpenAICompatible && in.ChannelType == 0 {
		in.ChannelType = constant.ChannelTypeOpenAI
	}
	if len(in.KeyURL) > 512 || len(in.LogoURL) > 512 {
		return errors.New("key_url or logo_url too long")
	}
	in.PricingTier = NormalizeByokPricingTier(in.PricingTier)
	return nil
}

func CreateByokProvider(in ByokProviderInput) (*ByokProvider, error) {
	if err := normalizeByokProviderInput(&in, true); err != nil {
		return nil, err
	}
	var existing ByokProvider
	err := DB.Where("id = ?", in.ID).First(&existing).Error
	if err == nil {
		return nil, errors.New("provider id already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	sortVal := 1000
	if in.Sort != nil {
		sortVal = *in.Sort
	}
	row := ByokProvider{
		ID:                 in.ID,
		Name:               in.Name,
		ChannelType:        in.ChannelType,
		Models:             in.Models,
		KeyHint:            in.KeyHint,
		NeedBase:           in.NeedBase,
		BaseHint:           in.BaseHint,
		Other:              in.Other,
		KeyURL:             in.KeyURL,
		LogoURL:            in.LogoURL,
		Enabled:            enabled,
		Sort:               sortVal,
		IsOpenAICompatible: in.IsOpenAICompatible,
		PricingTier:        in.PricingTier,
		UpdatedAt:          time.Now().Unix(),
	}
	if err := DB.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func UpdateByokProvider(id string, in ByokProviderInput) (*ByokProvider, error) {
	row, err := GetByokProvider(id)
	if err != nil {
		return nil, err
	}
	in.ID = row.ID
	if err := normalizeByokProviderInput(&in, false); err != nil {
		return nil, err
	}
	row.Name = in.Name
	row.ChannelType = in.ChannelType
	row.Models = in.Models
	row.KeyHint = in.KeyHint
	row.NeedBase = in.NeedBase
	row.BaseHint = in.BaseHint
	row.Other = in.Other
	row.KeyURL = in.KeyURL
	row.LogoURL = in.LogoURL
	row.IsOpenAICompatible = in.IsOpenAICompatible
	row.PricingTier = in.PricingTier
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	if in.Sort != nil {
		row.Sort = *in.Sort
	}
	row.UpdatedAt = time.Now().Unix()
	if err := DB.Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func SetByokProviderEnabled(id string, enabled bool) (*ByokProvider, error) {
	row, err := GetByokProvider(id)
	if err != nil {
		return nil, err
	}
	row.Enabled = enabled
	row.UpdatedAt = time.Now().Unix()
	if err := DB.Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func ReorderByokProviders(ids []string) error {
	if len(ids) == 0 {
		return errors.New("ids required")
	}
	seen := map[string]struct{}{}
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		for i, raw := range ids {
			id := strings.TrimSpace(strings.ToLower(raw))
			if !byokProviderIDRe.MatchString(id) {
				return fmt.Errorf("invalid id: %s", raw)
			}
			if _, ok := seen[id]; ok {
				return fmt.Errorf("duplicate id: %s", id)
			}
			seen[id] = struct{}{}
			res := tx.Model(&ByokProvider{}).Where("id = ?", id).Updates(map[string]any{
				"sort":       (i + 1) * 10,
				"updated_at": now,
			})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("provider not found: %s", id)
			}
		}
		return nil
	})
}
