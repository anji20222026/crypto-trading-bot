package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	openaiComponent "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// OllamaModel represents a model in Ollama
// OllamaModel 表示 Ollama 中的一个模型
type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
}

// OllamaListResponse represents the response from Ollama's /api/tags endpoint
// OllamaListResponse 表示 Ollama /api/tags 端点的响应
type OllamaListResponse struct {
	Models []OllamaModel `json:"models"`
}

func main() {

	ollamaBaseURL := "http://192.168.0.99:11434"
	ollamaAPIURL := ollamaBaseURL + "/v1"
	targetModel := "qwen2.5:7b-instruct-q4_K_M"

	// 步骤 1: 检查 Ollama 服务
	//t.Log("📡 步骤 1/4: 检查 Ollama 服务...")
	resp, _ := http.Get(ollamaBaseURL)

	defer resp.Body.Close()

	inputData := map[string]interface{}{
		"account": map[string]interface{}{
			"total_funds":     10000,
			"available_funds": 4500,
		},
		"trading_pairs": map[string]interface{}{
			"BTC/USDT": map[string]interface{}{
				"market": map[string]interface{}{
					"current_price": 91685.4,
					"trend":         "UP",
					"phase":         "IMPULSE",
					"EMA12":         90715,
					"EMA26":         90614,
					"MACD":          101.3,
					"RSI7":          79.1,
					"ATR7":          350,
				},
				"position": map[string]interface{}{
					"side":            "LONG",
					"size":            0.3,
					"entry_price":     91200,
					"unrealized_pnl":  485,
					"stop_loss":       90800,
					"take_profit":     93000,
					"allocated_funds": 3000,
				},
			},
			"ETH/USDT": map[string]interface{}{
				"market": map[string]interface{}{
					"current_price": 3100.5,
					"trend":         "DOWN",
					"phase":         "CORRECTION",
					"EMA12":         3150,
					"EMA26":         3120,
					"MACD":          -25,
					"RSI7":          34,
					"ATR7":          45,
				},
				"position": map[string]interface{}{
					"side":            "SHORT",
					"size":            0.2,
					"entry_price":     3150,
					"unrealized_pnl":  98,
					"stop_loss":       3180,
					"take_profit":     3000,
					"allocated_funds": 2500,
				},
			},
		},
	}

	inputJSON, _ := json.MarshalIndent(inputData, "", "  ")

	systemPrompt := `你是加密货币交易决策助手。

## 你的任务
对每个交易对，输出一个动作。

## 动作选项
- BUY: 开多仓
- SELL: 开空仓
- CLOSE_LONG: 平多仓
- CLOSE_SHORT: 平空仓
- HOLD: 无操作

## 决策规则
1. 如果有持仓（position.side != "NONE"）：
   - 趋势反转 → 平仓
   - 止损/止盈触发 → 平仓
   - 趋势延续 → 持有

2. 如果无持仓（position.side == "NONE"）：
   - 趋势明确 + 风险可控 → 开仓
   - 趋势不明确 → 观望

## 输出格式（必须严格遵守）
{
  "BTC/USDT": {
    "action": "HOLD",
    "reasoning": "原因（20字内）"
  },
  "ETH/USDT": {
    "action": "CLOSE_SHORT",
    "reasoning": "原因（20字内）"
  }
}

## 重要提示
- 只输出 JSON，不要其他文字
- reasoning 必须简短（20字内）
- 不确定时选择 HOLD`

	userPrompt := fmt.Sprintf("输入数据：\n%s\n\n请输出交易决策（JSON 格式）：", string(inputJSON))

	ctx := context.Background()

	// 创建 ChatModel
	chatComponent, _ := openaiComponent.NewChatModel(ctx, &openaiComponent.ChatModelConfig{
		APIKey:  "ollama",
		BaseURL: ollamaAPIURL,
		Model:   targetModel,
	})

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	startTime := time.Now()
	result, _ := chatComponent.Generate(ctx, messages)
	duration := time.Since(startTime)

	// 解析响应
	response := result.Content

	fmt.Println(result, duration)

	// 清理响应（移除可能的 markdown 代码块）
	cleanedResponse := response
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```json")
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	var decisions map[string]struct {
		Action    string `json:"action"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &decisions); err != nil {

	}

	for symbol, decision := range decisions {

		// 验证 action 是否合法
		validActions := map[string]bool{
			"BUY":         true,
			"SELL":        true,
			"CLOSE_LONG":  true,
			"CLOSE_SHORT": true,
			"HOLD":        true,
		}

		if !validActions[decision.Action] {
			fmt.Println("❌ %s 的 action 无效: %s", symbol, decision.Action)
		}

		// 验证 reasoning 不为空
		if decision.Reasoning == "" {
			fmt.Println("❌ %s 的 reasoning 为空", symbol)
		}
	}

}
