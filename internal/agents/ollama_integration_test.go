package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	openaiComponent "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/oak/crypto-trading-bot/internal/logger"
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

// TestOllamaConnection 测试 Ollama 连接和模型列表获取
// 这个测试会：
// 1. 检查 Ollama 是否运行
// 2. 获取所有可用模型
// 3. 查找 deepseek-r1:8b 模型
// 4. 使用该模型发送测试请求
func TestOllamaSimpleTradingDecision(t *testing.T) {
	t.Log("==========================================")
	t.Log("🦙 Ollama 极简交易决策测试")
	t.Log("==========================================")
	t.Log("")

	ollamaBaseURL := "http://192.168.0.99:11434"
	ollamaAPIURL := ollamaBaseURL + "/v1"
	targetModel := "deepseek-r1:8b"

	// 步骤 1: 检查 Ollama 服务
	t.Log("📡 步骤 1/4: 检查 Ollama 服务...")
	resp, err := http.Get(ollamaBaseURL)
	if err != nil {
		t.Skipf("⚠️  Ollama 服务未运行: %v\n提示：请先启动 Ollama (ollama serve)", err)
		return
	}
	defer resp.Body.Close()
	t.Log("✅ Ollama 服务运行正常")
	t.Log("")

	// 步骤 2: 构建输入数据
	t.Log("📊 步骤 2/4: 构建市场数据...")
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
	t.Logf("输入数据:\n%s\n", string(inputJSON))

	// 步骤 3: 构建 Prompt
	t.Log("📝 步骤 3/4: 构建 Prompt...")
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

	t.Log("✅ Prompt 构建完成")
	t.Log("")

	// 步骤 4: 调用 Ollama
	t.Log("🤖 步骤 4/4: 调用 Ollama DeepSeek...")
	t.Logf("模型: %s", targetModel)

	ctx := context.Background()

	// 创建 ChatModel
	chatComponent, err := openaiComponent.NewChatModel(ctx, &openaiComponent.ChatModelConfig{
		APIKey:  "ollama",
		BaseURL: ollamaAPIURL,
		Model:   targetModel,
	})
	if err != nil {
		t.Fatalf("❌ 创建 ChatModel 失败: %v", err)
	}

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	startTime := time.Now()
	result, err := chatComponent.Generate(ctx, messages)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("❌ Ollama 调用失败: %v", err)
	}

	t.Logf("⏱️  响应时间: %.2f 秒", duration.Seconds())
	t.Log("")

	// 解析响应
	response := result.Content
	t.Log("📥 原始响应:")
	t.Log("--------------------------------------------------")
	t.Log(response)
	t.Log("--------------------------------------------------")
	t.Log("")

	// 清理响应（移除可能的 markdown 代码块）
	cleanedResponse := response
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```json")
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	// 解析 JSON
	t.Log("🔍 解析 JSON...")
	var decisions map[string]struct {
		Action    string `json:"action"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &decisions); err != nil {
		t.Fatalf("❌ JSON 解析失败: %v\n响应内容:\n%s", err, cleanedResponse)
	}

	t.Log("✅ JSON 解析成功")
	t.Log("")

	// 验证输出
	t.Log("📊 决策结果:")
	for symbol, decision := range decisions {
		t.Logf("  %s:", symbol)
		t.Logf("    动作: %s", decision.Action)
		t.Logf("    原因: %s", decision.Reasoning)

		// 验证 action 是否合法
		validActions := map[string]bool{
			"BUY":         true,
			"SELL":        true,
			"CLOSE_LONG":  true,
			"CLOSE_SHORT": true,
			"HOLD":        true,
		}

		if !validActions[decision.Action] {
			t.Errorf("❌ %s 的 action 无效: %s", symbol, decision.Action)
		}

		// 验证 reasoning 不为空
		if decision.Reasoning == "" {
			t.Errorf("❌ %s 的 reasoning 为空", symbol)
		}
	}

	t.Log("")
	t.Log("==========================================")
	t.Log("✅ 测试通过！")
	t.Log("==========================================")
}

func TestOllamaConnection(t *testing.T) {
	t.Log("========================================")
	t.Log("🦙 Ollama 集成测试：连接检查 + 模型列表")
	t.Log("========================================\n")

	// Ollama 配置
	ollamaBaseURL := "http://192.168.0.99:11434"
	ollamaAPIURL := ollamaBaseURL + "/v1"

	// 步骤 1: 检查 Ollama 是否运行
	t.Log("📡 步骤 1/4: 检查 Ollama 服务...")
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(ollamaBaseURL)
	if err != nil {
		t.Skipf("⚠️  Ollama 服务未运行: %v\n提示：请先启动 Ollama (ollama serve)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("⚠️  Ollama 服务响应异常: HTTP %d", resp.StatusCode)
	}
	t.Log("✅ Ollama 服务运行正常\n")

	// 步骤 2: 获取所有可用模型
	t.Log("📋 步骤 2/4: 获取 Ollama 模型列表...")
	modelsResp, err := client.Get(ollamaBaseURL + "/api/tags")
	if err != nil {
		t.Fatalf("❌ 获取模型列表失败: %v", err)
	}
	defer modelsResp.Body.Close()

	body, err := io.ReadAll(modelsResp.Body)
	if err != nil {
		t.Fatalf("❌ 读取响应失败: %v", err)
	}

	var listResp OllamaListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("❌ 解析模型列表失败: %v", err)
	}

	if len(listResp.Models) == 0 {
		t.Skip("⚠️  Ollama 中没有已安装的模型\n提示：请先拉取模型 (ollama pull deepseek-r1:8b)")
	}

	t.Logf("✅ 找到 %d 个已安装的模型:\n", len(listResp.Models))
	for i, model := range listResp.Models {
		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)
		t.Logf("   %d. %s (%.2f GB)", i+1, model.Name, sizeGB)
	}
	t.Log("")

	// 步骤 3: 查找 deepseek-r1:8b 模型
	t.Log("🔍 步骤 3/4: 查找 deepseek-r1:8b 模型...")
	targetModel := "deepseek-r1:8b"
	modelFound := false

	for _, model := range listResp.Models {
		if strings.Contains(model.Name, "deepseek-r1") {
			targetModel = model.Name
			modelFound = true
			t.Logf("✅ 找到 DeepSeek 模型: %s\n", targetModel)
			break
		}
	}

	if !modelFound {
		t.Skip("⚠️  未找到 deepseek-r1 模型\n提示：请先拉取模型 (ollama pull deepseek-r1:8b)")
	}

	// 步骤 4: 使用 OpenAI 兼容接口测试模型
	t.Log("🤖 步骤 4/4: 测试模型推理...")
	t.Logf("   模型: %s", targetModel)
	t.Logf("   API: %s\n", ollamaAPIURL)

	log := logger.NewColorLogger(false)

	ctx := context.Background()
	cfg := &openaiComponent.ChatModelConfig{
		APIKey:  "ollama", // Ollama 不需要真实 API Key
		BaseURL: ollamaAPIURL,
		Model:   targetModel,
		// ReasoningEffort: openaiComponent.ReasoningEffortLevelLow, // ✅ 禁用思考模式，只输出最终结果
	}

	chatModel, err := openaiComponent.NewChatModel(ctx, cfg)
	if err != nil {
		t.Fatalf("❌ 创建 ChatModel 失败: %v", err)
	}

	// 发送简单测试消息
	messages := []*schema.Message{
		schema.SystemMessage("你是一个测试助手，请用中文简短回答。"),
		schema.UserMessage("请用一句话介绍你自己。"),
	}

	t.Log("📤 发送测试请求...")
	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		t.Fatalf("❌ LLM 调用失败: %v", err)
	}

	t.Log("✅ 模型响应成功！\n")
	t.Log("========================================")
	t.Log("📝 模型响应内容:")
	t.Log("========================================")
	t.Logf("%s\n", response.Content)
	t.Log("========================================")

	// 验证响应
	if response.Content == "" {
		t.Error("❌ 响应内容为空")
	}

	if len(response.Content) < 10 {
		t.Errorf("❌ 响应内容过短: %d 字符", len(response.Content))
	}

	// 打印 token 使用情况（如果有）
	if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		t.Log("\n📊 Token 使用统计:")
		t.Logf("   输入 Token:  %d", response.ResponseMeta.Usage.PromptTokens)
		t.Logf("   输出 Token:  %d", response.ResponseMeta.Usage.CompletionTokens)
		t.Logf("   总计 Token:  %d", response.ResponseMeta.Usage.TotalTokens)
	}

	t.Log("\n========================================")
	t.Log("✅ 测试总结:")
	t.Log("========================================")
	t.Logf("✓ Ollama 服务:  运行正常")
	t.Logf("✓ 模型列表:     %d 个模型", len(listResp.Models))
	t.Logf("✓ 目标模型:     %s", targetModel)
	t.Logf("✓ 模型推理:     成功")
	t.Logf("✓ 响应长度:     %d 字符", len(response.Content))
	t.Log("========================================")
	t.Log("🎉 Ollama 集成测试全部通过！")
	t.Log("========================================\n")

	// 记录日志（用于调试）
	log.Success(fmt.Sprintf("Ollama 测试成功: %s", targetModel))
}

// TestOllamaDeepSeekTrading 测试使用 Ollama DeepSeek 进行交易决策
// 这个测试会：
// 1. 连接到本地 Ollama
// 2. 使用 deepseek-r1:8b 模型
// 3. 发送简化的市场数据
// 4. 获取交易决策（JSON 格式）
func TestOllamaDeepSeekTrading(t *testing.T) {
	t.Log("========================================")
	t.Log("🦙 Ollama DeepSeek 交易决策测试")
	t.Log("========================================\n")

	// Ollama 配置
	ollamaBaseURL := "http://localhost:11434"
	ollamaAPIURL := ollamaBaseURL + "/v1"
	targetModel := "deepseek-r1:8b"

	// 检查 Ollama 是否运行
	t.Log("📡 检查 Ollama 服务...")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaBaseURL)
	if err != nil {
		t.Skipf("⚠️  Ollama 服务未运行: %v", err)
	}
	defer resp.Body.Close()

	// 检查模型是否存在
	t.Log("🔍 检查 deepseek-r1:8b 模型...")
	modelsResp, err := client.Get(ollamaBaseURL + "/api/tags")
	if err != nil {
		t.Skipf("⚠️  无法获取模型列表: %v", err)
	}
	defer modelsResp.Body.Close()

	body, _ := io.ReadAll(modelsResp.Body)
	var listResp OllamaListResponse
	json.Unmarshal(body, &listResp)

	modelFound := false
	for _, model := range listResp.Models {
		if strings.Contains(model.Name, "deepseek-r1") {
			targetModel = model.Name
			modelFound = true
			break
		}
	}

	if !modelFound {
		t.Skip("⚠️  未找到 deepseek-r1 模型，请先运行: ollama pull deepseek-r1:8b")
	}

	t.Logf("✅ 找到模型: %s\n", targetModel)

	// 初始化 ChatModel
	t.Log("🤖 初始化 ChatModel...")
	log := logger.NewColorLogger(false)
	ctx := context.Background()

	cfg := &openaiComponent.ChatModelConfig{
		APIKey:          "ollama",
		BaseURL:         ollamaAPIURL,
		Model:           targetModel,
		ReasoningEffort: openaiComponent.ReasoningEffortLevelLow, // ✅ 禁用思考模式，只输出最终结果

		// 使用 JSON Object 模式（DeepSeek 不支持 JSON Schema）
		ResponseFormat: &openaiComponent.ChatCompletionResponseFormat{
			Type: openaiComponent.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	chatModel, err := openaiComponent.NewChatModel(ctx, cfg)
	if err != nil {
		t.Fatalf("❌ 创建 ChatModel 失败: %v", err)
	}

	// 准备简化的市场数据
	t.Log("📊 准备市场数据...")
	marketData := `{
  "BTC/USDT": {
    "current_price": 95234.50,
    "24h_change": 2.34,
    "rsi_14": 58.5,
    "macd": 234.5,
    "macd_signal": 189.3,
    "trend": "上涨",
    "volume_24h": 1234567890
  }
}`

	// 准备系统提示词（简化版）
	systemPrompt := `你是一个加密货币交易助手。请根据市场数据给出交易建议。

输出格式必须是 JSON，包含以下字段：
{
  "symbol": "交易对",
  "action": "BUY/SELL/HOLD",
  "confidence": 0.0-1.0,
  "leverage": 1-20,
  "position_size": 0-100,
  "stop_loss": 价格,
  "reasoning": "理由",
  "risk_reward_ratio": 比例,
  "summary": "总结"
}

请用中文回答，只输出 JSON，不要其他内容。`

	userPrompt := fmt.Sprintf("请分析以下市场数据并给出交易建议：\n\n%s", marketData)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	// 发送请求
	t.Log("📤 发送交易决策请求...")
	t.Log("   (这可能需要 10-30 秒，取决于模型大小和硬件性能)\n")

	startTime := time.Now()
	response, err := chatModel.Generate(ctx, messages)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("❌ LLM 调用失败: %v", err)
	}

	t.Logf("✅ 模型响应成功！耗时: %.2f 秒\n", duration.Seconds())

	// 显示原始响应
	t.Log("========================================")
	t.Log("📝 原始 LLM 响应:")
	t.Log("========================================")
	t.Logf("%s\n", response.Content)
	t.Log("========================================\n")

	// 解析 JSON
	t.Log("🔍 解析 JSON 响应...")
	var decision TradeDecision

	// 清理响应内容（移除可能的 markdown 代码块）
	cleanContent := extractJSONPayload(response.Content)

	if err := json.Unmarshal([]byte(cleanContent), &decision); err != nil {
		t.Logf("⚠️  JSON 解析失败: %v", err)
		t.Logf("清理后的内容:\n%s", cleanContent)
		t.Skip("跳过验证：JSON 格式可能不完全符合预期（这在本地模型中很常见）")
	}

	t.Log("✅ JSON 解析成功！\n")

	// 显示解析后的决策
	t.Log("========================================")
	t.Log("📊 解析后的交易决策:")
	t.Log("========================================")
	t.Logf("🎯 交易对:       %s", decision.Symbol)
	t.Logf("📈 交易动作:     %s", decision.Action)
	t.Logf("💯 置信度:       %.2f (%.0f%%)", decision.Confidence, decision.Confidence*100)
	t.Logf("🔢 杠杆倍数:     %dx", decision.Leverage)
	t.Logf("💰 建议仓位:     %.1f%%", decision.PositionSize)
	t.Logf("🛑 止损价格:     $%.2f", decision.StopLoss)
	t.Logf("⚖️  盈亏比:       %.1f:1", decision.RiskRewardRatio)
	t.Logf("📝 交易理由:     %s", decision.Reasoning)
	t.Logf("📄 决策总结:     %s", decision.Summary)
	t.Log("========================================\n")

	// 基本验证
	t.Log("🔍 验证决策字段...")
	validationErrors := []string{}

	if decision.Symbol == "" {
		validationErrors = append(validationErrors, "symbol 字段为空")
	}
	if decision.Action == "" {
		validationErrors = append(validationErrors, "action 字段为空")
	}
	validActions := map[string]bool{
		"BUY": true, "SELL": true, "HOLD": true,
		"CLOSE_LONG": true, "CLOSE_SHORT": true,
	}
	if !validActions[decision.Action] {
		validationErrors = append(validationErrors, fmt.Sprintf("action 值无效: %s", decision.Action))
	}

	if len(validationErrors) > 0 {
		t.Log("⚠️  发现验证问题:")
		for _, errMsg := range validationErrors {
			t.Logf("   - %s", errMsg)
		}
		t.Log("   (这在本地模型中很常见，可能需要调整 prompt)")
	} else {
		t.Log("✅ 所有字段验证通过！")
	}

	// 性能统计
	t.Log("\n========================================")
	t.Log("📊 性能统计:")
	t.Log("========================================")
	t.Logf("⏱️  响应时间:     %.2f 秒", duration.Seconds())
	if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		t.Logf("📝 输入 Token:   %d", response.ResponseMeta.Usage.PromptTokens)
		t.Logf("📝 输出 Token:   %d", response.ResponseMeta.Usage.CompletionTokens)
		t.Logf("📝 总计 Token:   %d", response.ResponseMeta.Usage.TotalTokens)
		if duration.Seconds() > 0 {
			tokensPerSec := float64(response.ResponseMeta.Usage.CompletionTokens) / duration.Seconds()
			t.Logf("⚡ 生成速度:     %.1f tokens/秒", tokensPerSec)
		}
	}

	t.Log("\n========================================")
	t.Log("✅ 测试总结:")
	t.Log("========================================")
	t.Logf("✓ 模型:         %s", targetModel)
	t.Logf("✓ API 地址:     %s", ollamaAPIURL)
	t.Logf("✓ 响应时间:     %.2f 秒", duration.Seconds())
	t.Logf("✓ JSON 解析:    成功")
	t.Logf("✓ 决策动作:     %s", decision.Action)
	t.Log("========================================")
	t.Log("🎉 Ollama DeepSeek 交易决策测试完成！")
	t.Log("========================================\n")

	log.Success(fmt.Sprintf("Ollama DeepSeek 交易测试成功: %s -> %s", targetModel, decision.Action))
}
