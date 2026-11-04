package model

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ByokGuide is a public markdown guide page (DEV_FEATURES §3.6).
type ByokGuide struct {
	Slug        string `json:"slug" gorm:"primaryKey;size:64"`
	Title       string `json:"title" gorm:"size:120;not null"`
	Description string `json:"description" gorm:"size:255;not null"`
	Body        string `json:"body" gorm:"type:text;not null"`
	SortOrder   int    `json:"sort_order" gorm:"not null;default:0"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;not null;default:0"`
}

func (ByokGuide) TableName() string { return "byok_guides" }

// ByokVendorPrice stores official vendor list prices as decimal strings.
// Never accumulate money with float64 (DEV_FEATURES §3.5).
type ByokVendorPrice struct {
	Provider  string `json:"provider" gorm:"primaryKey;size:32"`
	Model     string `json:"model" gorm:"primaryKey;size:191"`
	Input     string `json:"input_cny" gorm:"column:input_cny;size:64;not null"`
	Output    string `json:"output_cny" gorm:"column:output_cny;size:64;not null"`
	Unit      string `json:"unit" gorm:"size:32;not null"`
	Currency  string `json:"currency" gorm:"size:8;not null;default:CNY"`
	Source    string `json:"source" gorm:"size:64;not null"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint;not null;default:0"`
}

func (ByokVendorPrice) TableName() string { return "byok_vendor_prices" }

var byokGuideSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type byokGuideSeed struct {
	Slug        string
	Title       string
	Description string
	Body        string
	SortOrder   int
}

var byokDefaultGuides = []byokGuideSeed{
	{
		Slug:        "cursor",
		SortOrder:   1,
		Title:       "Cursor / Claude Code 怎么接统一 API",
		Description: "把本站 Base URL 和本站令牌填进 Cursor 或 Claude Code，模型名必须和你登记过的一致。",
		Body: "本站把多家官方密钥收成一个 OpenAI 兼容入口。工具里只填本站地址和本站令牌，不要把厂商官方密钥直接贴进 Cursor。\n\n" +
			"## Cursor 怎么填\n\n" +
			"- **Base URL**：本站公开接口，以 /v1 结尾\n" +
			"- **API Key**：本站令牌，sk- 开头\n" +
			"- **模型名**：必须和渠道里登记的名字完全一致，或填 auto\n\n" +
			"## Claude Code\n\n" +
			"同样只填 Base URL、本站令牌、模型名。Anthropic 官方密钥先在「我的密钥」里保存。\n\n" +
			"## 常见失败\n\n" +
			"- 401：令牌不对，或厂商密钥无效\n" +
			"- 分组下无可用渠道：令牌没绑到你的私人分组\n" +
			"- 模型不存在：把嵌入模型名当成对话模型用了",
	},
	{
		Slug:        "openclaw",
		SortOrder:   2,
		Title:       "小龙虾、Hermes 怎么接统一 API",
		Description: "OpenClaw、小龙虾和 Hermes Agent 只要自定义 OpenAI 端点。Base URL 填本站 /v1，密钥填本站令牌。",
		Body: "小龙虾就是 OpenClaw。Hermes Agent 的 Custom Endpoint 也是 OpenAI 兼容的。\n\n" +
			"## 统一填法\n\n" +
			"- **Base URL**：本站公开接口，带 /v1\n" +
			"- **API Key**：本站令牌\n" +
			"- **模型**：你在厂商密钥页登记过的名字，或 auto\n\n" +
			"不要把 OpenAI、硅基流动的官方密钥直接贴进这些工具。官方密钥只放在本站「我的密钥」。\n\n" +
			"## 会不会自动切换厂商\n\n" +
			"填具体模型名时不会。填 auto 时，只在你已保存的对话模型里选，忙了或不可用再换下一家。不会按「要出图」自动改走 Flux。",
	},
	{
		Slug:        "siliconflow",
		SortOrder:   3,
		Title:       "硅基流动多模型：选模型名就等于选厂商",
		Description: "本站令牌没有厂商下拉。请求里的模型名决定走哪家渠道。硅基流动的 ID 和官方 DeepSeek 不是同一个名字。",
		Body: "硅基流动一家下面就有很多模型。路由只看请求里的 model 字符串。\n\n" +
			"## 怎么认到硅基流动\n\n" +
			"在「我的密钥」保存硅基流动官方密钥并写入模型 ID。工具里选例如 deepseek-ai/DeepSeek-V3，请求就会进硅基流动渠道。\n\n" +
			"硅基流动名字通常带前缀（deepseek-ai/、Qwen/）。官方 DeepSeek 常用 deepseek-chat，不是同一个字符串。\n\n" +
			"## 费用\n\n" +
			"真钱按你上传那把钥匙的厂商结算。打硅基流动的 DeepSeek 模型，账单在硅基流动。本站不加价。",
	},
	{
		Slug:        "byok",
		SortOrder:   4,
		Title:       "和代充站的区别：费用走官方账单",
		Description: "本站不卖额度、不加价。真钱按你上传那家厂商的官方价扣。",
		Body: "代充站卖的是别人的额度。本站不卖模型，只帮你把自己的官方密钥接到一个地址。\n\n" +
			"## 钱从哪扣\n\n" +
			"调用走你上传的那把官方钥匙。OpenAI 的请求出现在 OpenAI 账单，硅基流动的请求出现在硅基流动账单。本站令牌不另收模型费。\n\n" +
			"控制台里的额度、倍率是网关自己的记账闸，不是厂商真钱。\n\n" +
			"## 适合谁\n\n" +
			"- 自己有官方账号，只要聚合，不要买别人的额度\n" +
			"- 用 Cursor、Claude Code、小龙虾、Hermes，不想在每个软件里贴十几把钥匙\n\n" +
			"## 不适合谁\n\n" +
			"想充值买现成模型、不想自己申请厂商账号的人。这里没有货盘，也没有代充折扣。",
	},
	{
		Slug:        "auto",
		SortOrder:   5,
		Title:       "模型填 auto：自动选对话模型",
		Description: "统一令牌仍要带模型名。填 auto 后，本站只在你已接入的对话模型里选，忙了或不可用就换下一家。",
		Body: "## 怎么用\n\n" +
			"- **Base URL**：本站 /v1\n" +
			"- **API Key**：本站令牌\n" +
			"- **模型名**：auto\n\n" +
			"## 本站怎么选\n\n" +
			"1. 只看你账号下已经保存、并且能对话的模型。硅基先走代金券模型。\n" +
			"2. 跳过向量、语音识别、语音合成、生图、视频。\n" +
			"3. 按意图在券池里重排，失败再 failover。\n" +
			"4. 不会用别人的钥匙。同一轮回复中途不换模型。\n\n" +
			"## 做不到的\n\n" +
			"- 不会因为你说「出一张图」就自动改走 Flux\n" +
			"- 没接入的厂商不会被选中",
	},
}

// EnsureByokGuides seeds default guides if missing; refreshes selected slugs.
func EnsureByokGuides() error {
	if DB == nil {
		return errors.New("db not ready")
	}
	now := time.Now().Unix()
	refresh := map[string]bool{"cursor": true, "openclaw": true, "auto": true}
	for _, item := range byokDefaultGuides {
		var existing ByokGuide
		err := DB.Where("slug = ?", item.Slug).First(&existing).Error
		if err == nil {
			if refresh[item.Slug] {
				existing.Title = item.Title
				existing.Description = item.Description
				existing.Body = item.Body
				existing.SortOrder = item.SortOrder
				existing.UpdatedAt = now
				if saveErr := DB.Save(&existing).Error; saveErr != nil {
					return saveErr
				}
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		guide := ByokGuide{
			Slug:        item.Slug,
			Title:       item.Title,
			Description: item.Description,
			Body:        item.Body,
			SortOrder:   item.SortOrder,
			UpdatedAt:   now,
		}
		if createErr := DB.Create(&guide).Error; createErr != nil {
			return createErr
		}
	}
	return nil
}

func ListByokGuides() ([]ByokGuide, error) {
	var rows []ByokGuide
	err := DB.Order("sort_order asc, slug asc").Find(&rows).Error
	return rows, err
}

func GetByokGuide(slug string) (*ByokGuide, error) {
	if !byokGuideSlugRe.MatchString(slug) {
		return nil, errors.New("invalid slug")
	}
	var row ByokGuide
	if err := DB.Where("slug = ?", slug).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func UpdateByokGuide(slug, title, description, body string) error {
	guide, err := GetByokGuide(slug)
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	body = strings.TrimSpace(body)
	if title == "" || description == "" || body == "" {
		return errors.New("title, description and body are required")
	}
	if len([]rune(title)) > 80 || len([]rune(description)) > 180 {
		return errors.New("title or description too long")
	}
	if len(body) > 20000 {
		return errors.New("body too long")
	}
	guide.Title = title
	guide.Description = description
	guide.Body = body
	guide.UpdatedAt = time.Now().Unix()
	return DB.Save(guide).Error
}

func ListByokVendorPrices(provider string) ([]ByokVendorPrice, error) {
	var rows []ByokVendorPrice
	q := DB.Order("provider asc, model asc")
	if provider != "" {
		q = q.Where("provider = ?", provider)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func CountByokVendorPrices(provider string) (int64, error) {
	var n int64
	q := DB.Model(&ByokVendorPrice{})
	if provider != "" {
		q = q.Where("provider = ?", provider)
	}
	err := q.Count(&n).Error
	return n, err
}

// ReplaceByokVendorPrices replaces one provider's price rows atomically.
func ReplaceByokVendorPrices(provider, source string, rows []ByokVendorPrice) (int, error) {
	if provider == "" {
		return 0, errors.New("provider required")
	}
	now := time.Now().Unix()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("provider = ?", provider).Delete(&ByokVendorPrice{}).Error; err != nil {
			return err
		}
		for i := range rows {
			row := rows[i]
			row.Provider = provider
			row.Source = source
			row.UpdatedAt = now
			if row.Currency == "" {
				row.Currency = "CNY"
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return len(rows), err
}
