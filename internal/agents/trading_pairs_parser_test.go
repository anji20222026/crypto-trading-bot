package agents

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/oak/crypto-trading-bot/internal/logger"
)

// TestParseTradingPairsResponse tests parsing of the new trading_pairs format
// TestParseTradingPairsResponse 测试解析新的 trading_pairs 格式
func TestParseTradingPairsResponse(t *testing.T) {
	// Sample JSON response in trading_pairs format
	// trading_pairs 格式的示例 JSON 响应
	jsonResponse := `{
  "trading_pairs": {
    "BTC/USDT": {
      "trend_analyzer": {
        "trend": { "value": "UP" },
        "phase": { "value": "IMPULSE" },
        "risk": { "value": "LOW" }
      },
      "confidence_to_leverage": {
        "confidence": { "value": 0.85 },
        "leverage": { "value": 10, "min": 5, "max": 15 }
      },
      "risk_metrics": {
        "ATR": { "value": 650.5 },
        "stop_loss": { "value": 89000.0 },
        "take_profit": { "value": 95000.0 },
        "estimated_risk_reward": { "value": 3.8 },
        "support_levels": [89000, 88500],
        "resistance_levels": [92000, 93000]
      },
      "decision_gate": {
        "value": "TRADE",
        "reason_code": { "value": [] }
      },
      "trading_signal": {
        "action": { "value": "BUY" },
        "stop_loss": 89000.0,
        "take_profit": 95000.0,
        "position_size": 10.0,
        "reasoning": "强势上涨趋势，突破关键阻力位"
      }
    }
  }
}`

	// Parse JSON
	// 解析 JSON
	var tradingPairsResp TradingPairsResponse
	err := sonic.Unmarshal([]byte(jsonResponse), &tradingPairsResp)
	if err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}

	// Verify parsing
	// 验证解析
	if len(tradingPairsResp.TradingPairs) != 1 {
		t.Errorf("期望 1 个交易对，实际得到 %d 个", len(tradingPairsResp.TradingPairs))
	}

	btcDecision, ok := tradingPairsResp.TradingPairs["BTC/USDT"]
	if !ok {
		t.Fatal("未找到 BTC/USDT 交易对")
	}

	// Verify trend analyzer
	// 验证趋势分析器
	if btcDecision.TrendAnalyzer == nil {
		t.Fatal("trend_analyzer 为空")
	}
	if btcDecision.TrendAnalyzer.Trend.Value != "UP" {
		t.Errorf("期望趋势为 UP，实际为 %s", btcDecision.TrendAnalyzer.Trend.Value)
	}

	// Verify confidence and leverage
	// 验证置信度和杠杆
	if btcDecision.ConfidenceToLeverage == nil {
		t.Fatal("confidence_to_leverage 为空")
	}
	if btcDecision.ConfidenceToLeverage.Confidence.Value != 0.85 {
		t.Errorf("期望置信度为 0.85，实际为 %.2f", btcDecision.ConfidenceToLeverage.Confidence.Value)
	}
	if btcDecision.ConfidenceToLeverage.Leverage.Value != 10 {
		t.Errorf("期望杠杆为 10，实际为 %d", btcDecision.ConfidenceToLeverage.Leverage.Value)
	}

	// Verify trading signal
	// 验证交易信号
	if btcDecision.TradingSignal == nil {
		t.Fatal("trading_signal 为空")
	}
	if btcDecision.TradingSignal.Action.Value != "BUY" {
		t.Errorf("期望动作为 BUY，实际为 %s", btcDecision.TradingSignal.Action.Value)
	}

	t.Log("✅ trading_pairs 格式解析测试通过")
}

// TestConvertTradingPairsToLegacyFormat tests conversion from trading_pairs to legacy format
// TestConvertTradingPairsToLegacyFormat 测试从 trading_pairs 转换为旧格式
func TestConvertTradingPairsToLegacyFormat(t *testing.T) {
	// Create sample trading pairs
	// 创建示例交易对
	tradingPairs := map[string]*SymbolDecision{
		"BTC/USDT": {
			TrendAnalyzer: &TrendAnalyzer{
				Trend: &EnumValue{Value: "UP"},
				Phase: &EnumValue{Value: "IMPULSE"},
				Risk:  &EnumValue{Value: "LOW"},
			},
			ConfidenceToLeverage: &ConfidenceToLeverage{
				Confidence: &ValueOnly{Value: 0.85},
				Leverage:   &LeverageValue{Value: 10, Min: 5, Max: 15},
			},
			RiskMetrics: &RiskMetrics{
				ATR:                 &ATRValue{Value: 650.5},
				StopLoss:            &ValueOnly{Value: 89000.0},
				TakeProfit:          &ValueOnly{Value: 95000.0},
				EstimatedRiskReward: &ValueOnly{Value: 3.8},
				SupportLevels:       []float64{89000, 88500},
				ResistanceLevels:    []float64{92000, 93000},
			},
			DecisionGate: &DecisionGate{
				Value:      "TRADE",
				ReasonCode: &ReasonCode{Value: []string{}},
			},
			TradingSignal: &TradingSignal{
				Action:       &EnumValue{Value: "BUY"},
				StopLoss:     89000.0,
				TakeProfit:   95000.0,
				PositionSize: 10.0,
				Reasoning:    "强势上涨趋势，突破关键阻力位",
			},
		},
	}

	// Convert to legacy format
	// 转换为旧格式
	log := logger.NewColorLogger(true)
	legacyFormat, err := convertTradingPairsToLegacyFormat(tradingPairs, log)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	// Verify conversion
	// 验证转换
	btcDecision, ok := legacyFormat["BTC/USDT"]
	if !ok {
		t.Fatal("未找到 BTC/USDT 交易对")
	}

	if btcDecision.Symbol != "BTC/USDT" {
		t.Errorf("期望 symbol 为 BTC/USDT，实际为 %s", btcDecision.Symbol)
	}
	if btcDecision.Action != "BUY" {
		t.Errorf("期望 action 为 BUY，实际为 %s", btcDecision.Action)
	}
	if btcDecision.Confidence != 0.85 {
		t.Errorf("期望 confidence 为 0.85，实际为 %.2f", btcDecision.Confidence)
	}
	if btcDecision.Leverage != 10 {
		t.Errorf("期望 leverage 为 10，实际为 %d", btcDecision.Leverage)
	}
	if btcDecision.StopLoss != 89000.0 {
		t.Errorf("期望 stop_loss 为 89000.0，实际为 %.2f", btcDecision.StopLoss)
	}
	if btcDecision.PositionSize != 10.0 {
		t.Errorf("期望 position_size 为 10.0，实际为 %.2f", btcDecision.PositionSize)
	}
	if btcDecision.RiskRewardRatio != 3.8 {
		t.Errorf("期望 risk_reward_ratio 为 3.8，实际为 %.2f", btcDecision.RiskRewardRatio)
	}

	t.Log("✅ trading_pairs 转换为旧格式测试通过")
}

