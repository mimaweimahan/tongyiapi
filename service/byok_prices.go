package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
)

var (
	siliconPriceURLs = []string{
		"https://cdn.jsdelivr.net/gh/Oaklight/onehub_prices@prices/siliconflow_prices.json",
		"https://cdn.jsdmirror.com/gh/Oaklight/onehub_prices@prices/siliconflow_prices.json",
		"https://raw.githubusercontent.com/Oaklight/onehub_prices/prices/siliconflow_prices.json",
	}
	geminiPriceURLs = []string{
		"https://ai.google.dev/gemini-api/docs/pricing",
		"https://ai.google.dev/gemini-api/docs/pricing?hl=en",
	}
	litellmPriceURLs = []string{
		"https://cdn.jsdelivr.net/gh/BerriAI/litellm@main/model_prices_and_context_window.json",
		"https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
	}
	onehubToCNYM = decimal.RequireFromString("14")
)

func formatMoney(value decimal.Decimal) string {
	text := value.Round(4).StringFixed(4)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func httpGetBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "byok-new-api/1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func httpGetJSON(url string, dest any) error {
	raw, err := httpGetBytes(url)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

// RefreshSiliconPrices pulls siliconflow list prices via onehub mirror (CNY / 1M tokens).
func RefreshSiliconPrices() (int, error) {
	var lastErr error
	for _, url := range siliconPriceURLs {
		var payload any
		if err := httpGetJSON(url, &payload); err != nil {
			lastErr = err
			continue
		}
		items := payload
		if m, ok := payload.(map[string]any); ok {
			items = m["data"]
		}
		list, ok := items.([]any)
		if !ok {
			lastErr = errors.New("unexpected silicon price payload")
			continue
		}
		rows := make([]model.ByokVendorPrice, 0, len(list))
		for _, raw := range list {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			name := strings.TrimSpace(fmt.Sprint(item["model"]))
			if name == "" || name == "<nil>" {
				continue
			}
			kind := fmt.Sprint(item["type"])
			if kind == "" || kind == "<nil>" {
				kind = "tokens"
			}
			inRaw, e1 := decimal.NewFromString(fmt.Sprint(item["input"]))
			outRaw, e2 := decimal.NewFromString(fmt.Sprint(item["output"]))
			if e1 != nil || e2 != nil {
				continue
			}
			unit := "each"
			in := inRaw
			out := outRaw
			if kind == "tokens" {
				in = inRaw.Mul(onehubToCNYM)
				out = outRaw.Mul(onehubToCNYM)
				unit = "token_m"
			}
			rows = append(rows, model.ByokVendorPrice{
				Model:    name,
				Input:    formatMoney(in),
				Output:   formatMoney(out),
				Unit:     unit,
				Currency: "CNY",
			})
		}
		if len(rows) == 0 {
			lastErr = errors.New("empty silicon price rows")
			continue
		}
		return model.ReplaceByokVendorPrices("silicon", "siliconflow", rows)
	}
	if lastErr == nil {
		lastErr = errors.New("silicon price refresh failed")
	}
	return 0, lastErr
}

func parseGeminiOfficialPage(text string) map[string][2]decimal.Decimal {
	tokenRe := regexp.MustCompile("`((?:gemini|gemma)-[a-z0-9._-]+)`")
	dollarRe := regexp.MustCompile(`\$([0-9]+(?:\.[0-9]+)?)`)
	matches := tokenRe.FindAllStringSubmatchIndex(text, -1)
	pending := make([]string, 0)
	found := map[string][2]decimal.Decimal{}
	for index, loc := range matches {
		name := text[loc[2]:loc[3]]
		pending = append(pending, name)
		end := len(text)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		} else if loc[1]+4000 < end {
			end = loc[1] + 4000
		}
		chunk := text[loc[1]:end]
		var inp, out *decimal.Decimal
		for _, line := range strings.Split(chunk, "\n") {
			low := strings.ToLower(line)
			if inp == nil && strings.Contains(low, "input price") {
				nums := dollarRe.FindStringSubmatch(line)
				if len(nums) > 1 {
					v := decimal.RequireFromString(nums[1])
					inp = &v
				}
			} else if out == nil && strings.Contains(low, "output price") {
				nums := dollarRe.FindStringSubmatch(line)
				if len(nums) > 1 {
					v := decimal.RequireFromString(nums[1])
					out = &v
				}
			}
			if inp != nil && out != nil {
				break
			}
		}
		if inp == nil || out == nil {
			continue
		}
		for _, n := range pending {
			found[n] = [2]decimal.Decimal{*inp, *out}
		}
		pending = pending[:0]
	}
	return found
}

func parseGeminiLiteLLM(payload map[string]any) map[string][2]decimal.Decimal {
	found := map[string][2]decimal.Decimal{}
	million := decimal.NewFromInt(1000000)
	for key, raw := range payload {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		provider := fmt.Sprint(item["litellm_provider"])
		modelName := ""
		if provider == "gemini" {
			parts := strings.Split(key, "/")
			modelName = parts[len(parts)-1]
		} else if strings.HasPrefix(key, "gemini/") {
			modelName = strings.TrimPrefix(key, "gemini/")
		}
		if modelName == "" || !(strings.HasPrefix(modelName, "gemini-") || strings.HasPrefix(modelName, "gemma-")) {
			continue
		}
		inTok, err1 := decimal.NewFromString(fmt.Sprint(item["input_cost_per_token"]))
		outTok, err2 := decimal.NewFromString(fmt.Sprint(item["output_cost_per_token"]))
		if err1 != nil || err2 != nil {
			continue
		}
		in := inTok.Mul(million)
		out := outTok.Mul(million)
		if in.IsZero() && out.IsZero() {
			continue
		}
		if _, exists := found[modelName]; !exists {
			found[modelName] = [2]decimal.Decimal{in, out}
		}
	}
	return found
}

func rowsFromGeminiMap(found map[string][2]decimal.Decimal) []model.ByokVendorPrice {
	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]model.ByokVendorPrice, 0, len(names))
	for _, name := range names {
		prices := found[name]
		rows = append(rows, model.ByokVendorPrice{
			Model:    name,
			Input:    formatMoney(prices[0]),
			Output:   formatMoney(prices[1]),
			Unit:     "token_m",
			Currency: "USD",
		})
	}
	return rows
}

// RefreshGeminiPrices pulls Gemini official page, falling back to litellm.
func RefreshGeminiPrices() (int, error) {
	var lastErr error
	for _, url := range geminiPriceURLs {
		textBytes, err := httpGetBytes(url)
		if err != nil {
			lastErr = err
			continue
		}
		found := parseGeminiOfficialPage(string(textBytes))
		rows := rowsFromGeminiMap(found)
		if len(rows) >= 3 {
			return model.ReplaceByokVendorPrices("gemini", "official", rows)
		}
		lastErr = errors.New("gemini official parse too few rows")
	}
	for _, url := range litellmPriceURLs {
		var payload map[string]any
		if err := httpGetJSON(url, &payload); err != nil {
			lastErr = err
			continue
		}
		rows := rowsFromGeminiMap(parseGeminiLiteLLM(payload))
		if len(rows) == 0 {
			lastErr = errors.New("empty litellm gemini rows")
			continue
		}
		return model.ReplaceByokVendorPrices("gemini", "litellm", rows)
	}
	if lastErr == nil {
		lastErr = errors.New("gemini price refresh failed")
	}
	return 0, lastErr
}

// EnsureOfficialPrices refreshes empty vendor tables best-effort.
func EnsureOfficialPrices() {
	if n, err := model.CountByokVendorPrices("silicon"); err == nil && n == 0 {
		_, _ = RefreshSiliconPrices()
	}
	if n, err := model.CountByokVendorPrices("gemini"); err == nil && n == 0 {
		_, _ = RefreshGeminiPrices()
	}
}

// EstimateUsageCostItem is one log line priced with decimal math.
type EstimateUsageCostItem struct {
	CreatedAt        int64  `json:"created_at"`
	ModelName        string `json:"model_name"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Currency         string `json:"currency,omitempty"`
	Cost             string `json:"cost,omitempty"`
	Priced           bool   `json:"priced"`
}

// EstimateUsageCost totals official vendor cost for consume logs without float accumulation.
func EstimateUsageCost(logs []*model.Log) (items []EstimateUsageCostItem, totals map[string]string, known int, err error) {
	prices, err := model.ListByokVendorPrices("")
	if err != nil {
		return nil, nil, 0, err
	}
	byModel := make(map[string]model.ByokVendorPrice, len(prices))
	for _, p := range prices {
		byModel[p.Model] = p
	}
	totalDec := map[string]decimal.Decimal{}
	items = make([]EstimateUsageCostItem, 0, len(logs))
	million := decimal.NewFromInt(1000000)
	for _, row := range logs {
		if row == nil || row.Type != model.LogTypeConsume || row.ModelName == "" {
			continue
		}
		item := EstimateUsageCostItem{
			CreatedAt:        row.CreatedAt,
			ModelName:        row.ModelName,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
		}
		price, ok := byModel[row.ModelName]
		if !ok {
			items = append(items, item)
			continue
		}
		currency := price.Currency
		if currency == "" {
			currency = "CNY"
		}
		item.Currency = currency
		var cost decimal.Decimal
		if price.Unit == "token_m" {
			inPrice, e1 := decimal.NewFromString(price.Input)
			outPrice, e2 := decimal.NewFromString(price.Output)
			if e1 != nil || e2 != nil {
				items = append(items, item)
				continue
			}
			cost = inPrice.Mul(decimal.NewFromInt(int64(row.PromptTokens))).Div(million).
				Add(outPrice.Mul(decimal.NewFromInt(int64(row.CompletionTokens))).Div(million))
		} else {
			outPrice, e1 := decimal.NewFromString(price.Output)
			if e1 != nil {
				outPrice, e1 = decimal.NewFromString(price.Input)
			}
			if e1 != nil {
				items = append(items, item)
				continue
			}
			cost = outPrice
		}
		item.Priced = true
		item.Cost = formatMoney(cost)
		totalDec[currency] = totalDec[currency].Add(cost)
		known++
		items = append(items, item)
	}
	totals = map[string]string{}
	for currency, amount := range totalDec {
		totals[currency] = formatMoney(amount)
	}
	return items, totals, known, nil
}
