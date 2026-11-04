package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var byokVendorNames = map[string]string{
	"silicon": "硅基流动",
	"gemini":  "Gemini",
}

// GetByokPrices returns official vendor price rows (public).
func GetByokPrices(c *gin.Context) {
	service.EnsureOfficialPrices()
	provider := strings.TrimSpace(c.Query("provider"))
	rows, err := model.ListByokVendorPrices(provider)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items":   rows,
		"vendors": byokVendorNames,
		"count":   len(rows),
	})
}

// SyncByokPrices refreshes silicon and/or gemini official prices (root).
func SyncByokPrices(c *gin.Context) {
	provider := strings.TrimSpace(c.Query("provider"))
	if provider == "" {
		var body struct {
			Provider string `json:"provider"`
		}
		_ = c.ShouldBindJSON(&body)
		provider = strings.TrimSpace(body.Provider)
	}
	result := gin.H{}
	var err error
	var n int
	switch provider {
	case "", "all":
		n, err = service.RefreshSiliconPrices()
		result["silicon"] = n
		if err != nil {
			result["silicon_error"] = err.Error()
		}
		n2, err2 := service.RefreshGeminiPrices()
		result["gemini"] = n2
		if err2 != nil {
			result["gemini_error"] = err2.Error()
			if err == nil {
				err = err2
			}
		}
	case "silicon":
		n, err = service.RefreshSiliconPrices()
		result["silicon"] = n
	case "gemini":
		n, err = service.RefreshGeminiPrices()
		result["gemini"] = n
	default:
		common.ApiErrorMsg(c, "unknown provider")
		return
	}
	if err != nil && result["silicon"] == nil && result["gemini"] == nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

// GetByokGuides lists public guides.
func GetByokGuides(c *gin.Context) {
	_ = model.EnsureByokGuides()
	rows, err := model.ListByokGuides()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// omit heavy body on list
	type guideCard struct {
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
		UpdatedAt   int64  `json:"updated_at"`
	}
	cards := make([]guideCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, guideCard{
			Slug:        row.Slug,
			Title:       row.Title,
			Description: row.Description,
			SortOrder:   row.SortOrder,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	common.ApiSuccess(c, gin.H{"items": cards})
}

// GetByokGuide returns one guide including body.
func GetByokGuide(c *gin.Context) {
	_ = model.EnsureByokGuides()
	slug := c.Param("slug")
	row, err := model.GetByokGuide(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "invalid slug") {
			common.ApiErrorMsg(c, "guide not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, row)
}

// UpdateByokGuide updates guide content (root).
func UpdateByokGuide(c *gin.Context) {
	slug := c.Param("slug")
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Body        string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateByokGuide(slug, body.Title, body.Description, body.Body); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	row, _ := model.GetByokGuide(slug)
	common.ApiSuccess(c, row)
}

// GetByokUsageCost estimates official cost for the current user's consume logs.
func GetByokUsageCost(c *gin.Context) {
	service.EnsureOfficialPrices()
	userId := c.GetInt("id")
	logs, _, err := model.GetUserLogs(userId, model.LogFilter{Type: model.LogTypeConsume}, 0, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items, totals, known, err := service.EstimateUsageCost(logs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items":  items,
		"totals": totals,
		"known":  known,
		"count":  len(items),
	})
}

// RobotsTxt serves crawler rules; keys/admin paths are disallowed.
func RobotsTxt(c *gin.Context) {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	body := "" +
		"User-agent: *\n" +
		"Allow: /\n" +
		"Allow: /guides\n" +
		"Allow: /prices\n" +
		"Disallow: /dashboard\n" +
		"Disallow: /my-keys\n" +
		"Disallow: /keys\n" +
		"Disallow: /usage-logs\n" +
		"Disallow: /usage-cost\n" +
		"Disallow: /users\n" +
		"Disallow: /channels\n" +
		"Disallow: /system-settings\n" +
		"Disallow: /system-info\n" +
		"Disallow: /sign-in\n" +
		"Disallow: /sign-up\n" +
		"Disallow: /guides-admin\n" +
		"Disallow: /wallet\n" +
		"Disallow: /playground\n" +
		"Sitemap: " + base + "/sitemap.xml\n"
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

// SitemapXml lists public marketing pages.
func SitemapXml(c *gin.Context) {
	_ = model.EnsureByokGuides()
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	now := time.Now().UTC().Format("2006-01-02")
	urls := []struct {
		Path     string
		Freq     string
		Priority string
	}{
		{"/", "weekly", "1.0"},
		{"/guides", "weekly", "0.8"},
		{"/prices", "weekly", "0.6"},
	}
	if guides, err := model.ListByokGuides(); err == nil {
		for _, g := range guides {
			urls = append(urls, struct {
				Path     string
				Freq     string
				Priority string
			}{"/guides/" + g.Slug, "monthly", "0.7"})
		}
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, u := range urls {
		fmt.Fprintf(&b, "<url><loc>%s%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>\n",
			xmlEscape(base), xmlEscape(u.Path), now, u.Freq, u.Priority)
	}
	b.WriteString("</urlset>\n")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(b.String()))
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
