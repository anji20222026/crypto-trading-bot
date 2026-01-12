package executors

import (
	"fmt"

	"github.com/oak/crypto-trading-bot/internal/config"
	"github.com/sirupsen/logrus"
)

// PositionManager manages position allocation across multiple symbols
// PositionManager 管理多币种的仓位分配
type PositionManager struct {
	config *config.Config
	logger *logrus.Logger
}

// NewPositionManager creates a new position manager
// NewPositionManager 创建新的仓位管理器
func NewPositionManager(cfg *config.Config, logger *logrus.Logger) *PositionManager {
	return &PositionManager{
		config: cfg,
		logger: logger,
	}
}

// TradeDecision represents a trading decision for a symbol
// TradeDecision 表示某个币种的交易决策
type TradeDecision struct {
	Symbol       string
	Action       string  // BUY, SELL, HOLD, CLOSE_LONG, CLOSE_SHORT
	Confidence   float64 // 0.0 - 1.0
	CurrentPrice float64
	ATR          float64 // For volatility adjustment
	Reasoning    string
}

// AllocatePositions allocates positions across multiple symbols based on configuration
// AllocatePositions 根据配置在多个币种之间分配仓位
func (pm *PositionManager) AllocatePositions(decisions map[string]*TradeDecision) map[string]float64 {
	pm.logger.Info("========================================")
	pm.logger.Infof("📊 仓位分配模式: %s", pm.config.PositionAllocationMode)
	pm.logger.Info("========================================")

	switch pm.config.PositionAllocationMode {
	case "equal":
		return pm.allocateEqual(decisions)
	case "weighted":
		return pm.allocateWeighted(decisions)
	case "fixed":
		return pm.allocateFixed(decisions)
	default:
		pm.logger.Warnf("未知的仓位分配模式: %s，使用 weighted 模式", pm.config.PositionAllocationMode)
		return pm.allocateWeighted(decisions)
	}
}

// allocateEqual allocates positions equally among all valid signals
// allocateEqual 在所有有效信号之间均分仓位
func (pm *PositionManager) allocateEqual(decisions map[string]*TradeDecision) map[string]float64 {
	positions := make(map[string]float64)

	// Filter valid signals (non-HOLD)
	// 过滤有效信号（非 HOLD）
	validSymbols := []string{}
	for symbol, decision := range decisions {
		if pm.isValidSignal(decision) {
			validSymbols = append(validSymbols, symbol)
		}
	}

	if len(validSymbols) == 0 {
		pm.logger.Info("⚠️  没有有效信号，全部 HOLD")
		return positions
	}

	// Equal allocation
	// 均分仓位
	totalLimit := pm.config.TotalPositionLimit
	perSymbol := totalLimit / float64(len(validSymbols))

	// Apply max symbol position limit
	// 应用单币种上限
	if perSymbol > pm.config.MaxSymbolPosition {
		perSymbol = pm.config.MaxSymbolPosition
	}

	for _, symbol := range validSymbols {
		if perSymbol >= pm.config.MinSymbolPosition {
			positions[symbol] = perSymbol
			pm.logger.Infof("  %s: %.1f%% (均分)", symbol, perSymbol)
		} else {
			pm.logger.Infof("  %s: 0%% (低于最小仓位 %.1f%%)", symbol, pm.config.MinSymbolPosition)
		}
	}

	pm.logTotalPosition(positions)
	return positions
}

// allocateWeighted allocates positions weighted by confidence
// allocateWeighted 按置信度加权分配仓位
func (pm *PositionManager) allocateWeighted(decisions map[string]*TradeDecision) map[string]float64 {
	positions := make(map[string]float64)

	// Filter valid signals and calculate total confidence
	// 过滤有效信号并计算总置信度
	validDecisions := make(map[string]*TradeDecision)
	totalConfidence := 0.0

	for symbol, decision := range decisions {
		if pm.isValidSignal(decision) {
			validDecisions[symbol] = decision
			totalConfidence += decision.Confidence
		}
	}

	if len(validDecisions) == 0 {
		pm.logger.Info("⚠️  没有有效信号，全部 HOLD")
		return positions
	}

	pm.logger.Infof("📈 有效信号数: %d, 总置信度: %.2f", len(validDecisions), totalConfidence)

	// Allocate by confidence weight
	// 按置信度权重分配
	totalLimit := pm.config.TotalPositionLimit

	for symbol, decision := range validDecisions {
		// Weight = confidence / total confidence
		// 权重 = 该币种置信度 / 总置信度
		weight := decision.Confidence / totalConfidence

		// Position = total limit × weight
		// 仓位 = 总上限 × 权重
		position := totalLimit * weight

		// Apply symbol limits
		// 应用单币种限制
		if position > pm.config.MaxSymbolPosition {
			pm.logger.Infof("  %s: %.1f%% → %.1f%% (受单币种上限限制)", symbol, position, pm.config.MaxSymbolPosition)
			position = pm.config.MaxSymbolPosition
		}

		if position < pm.config.MinSymbolPosition {
			pm.logger.Infof("  %s: %.1f%% → 0%% (低于最小仓位)", symbol, position)
			position = 0.0
		}

		if position > 0 {
			positions[symbol] = position
			pm.logger.Infof("  %s: %.1f%% (置信度: %.2f, 权重: %.1f%%)", symbol, position, decision.Confidence, weight*100)
		}
	}

	// Check total position and scale down if needed
	// 检查总仓位，如果超限则按比例缩减
	positions = pm.applyTotalLimit(positions)

	pm.logTotalPosition(positions)
	return positions
}

// allocateFixed allocates fixed position size to each signal
// allocateFixed 为每个信号分配固定仓位
func (pm *PositionManager) allocateFixed(decisions map[string]*TradeDecision) map[string]float64 {
	positions := make(map[string]float64)

	fixedSize := pm.config.FixedPositionSize

	for symbol, decision := range decisions {
		if pm.isValidSignal(decision) {
			position := fixedSize

			// Apply symbol limits
			// 应用单币种限制
			if position > pm.config.MaxSymbolPosition {
				position = pm.config.MaxSymbolPosition
			}

			if position >= pm.config.MinSymbolPosition {
				positions[symbol] = position
				pm.logger.Infof("  %s: %.1f%% (固定)", symbol, position)
			} else {
				pm.logger.Infof("  %s: 0%% (低于最小仓位)", symbol)
			}
		}
	}

	// Apply total limit
	// 应用总仓位限制
	positions = pm.applyTotalLimit(positions)

	pm.logTotalPosition(positions)
	return positions
}

// isValidSignal checks if a decision is a valid trading signal
// isValidSignal 检查决策是否为有效交易信号
func (pm *PositionManager) isValidSignal(decision *TradeDecision) bool {
	if decision == nil {
		return false
	}

	// HOLD or CLOSE_* actions don't need new positions
	// HOLD 或 CLOSE_* 动作不需要新仓位
	if decision.Action == "HOLD" || decision.Action == "CLOSE_LONG" || decision.Action == "CLOSE_SHORT" {
		return false
	}

	// Only BUY and SELL need positions
	// 只有 BUY 和 SELL 需要仓位
	if decision.Action != "BUY" && decision.Action != "SELL" {
		return false
	}

	// Confidence should be reasonable
	// 置信度应该合理
	if decision.Confidence <= 0 || decision.Confidence > 1.0 {
		pm.logger.Warnf("⚠️  %s 置信度异常: %.2f", decision.Symbol, decision.Confidence)
		return false
	}

	return true
}

// applyTotalLimit scales down positions if total exceeds limit
// applyTotalLimit 如果总仓位超限则按比例缩减
func (pm *PositionManager) applyTotalLimit(positions map[string]float64) map[string]float64 {
	total := 0.0
	for _, pos := range positions {
		total += pos
	}

	limit := pm.config.TotalPositionLimit
	if total <= limit {
		return positions // No scaling needed
	}

	// Scale down proportionally
	// 按比例缩减
	scale := limit / total
	pm.logger.Warnf("⚠️  总仓位 %.1f%% 超过限制 %.1f%%，按比例缩减 (×%.2f)", total, limit, scale)

	for symbol := range positions {
		oldPos := positions[symbol]
		positions[symbol] = oldPos * scale
		pm.logger.Infof("  %s: %.1f%% → %.1f%%", symbol, oldPos, positions[symbol])
	}

	return positions
}

// logTotalPosition logs the total allocated position
// logTotalPosition 记录总分配仓位
func (pm *PositionManager) logTotalPosition(positions map[string]float64) {
	total := 0.0
	for _, pos := range positions {
		total += pos
	}

	pm.logger.Info("========================================")
	pm.logger.Infof("📊 总仓位: %.1f%% / %.1f%%", total, pm.config.TotalPositionLimit)
	pm.logger.Info("========================================")
}

// ValidatePositionSize validates and clamps a position size
// ValidatePositionSize 验证并限制仓位大小
func (pm *PositionManager) ValidatePositionSize(symbol string, size float64) float64 {
	if size < pm.config.MinSymbolPosition {
		pm.logger.Infof("⚠️  %s 仓位 %.1f%% 低于最小值 %.1f%%，设为 0", symbol, size, pm.config.MinSymbolPosition)
		return 0.0
	}

	if size > pm.config.MaxSymbolPosition {
		pm.logger.Warnf("⚠️  %s 仓位 %.1f%% 超过最大值 %.1f%%，强制限制", symbol, size, pm.config.MaxSymbolPosition)
		return pm.config.MaxSymbolPosition
	}

	return size
}

// GetPositionSummary returns a summary of position allocation
// GetPositionSummary 返回仓位分配摘要
func (pm *PositionManager) GetPositionSummary(positions map[string]float64) string {
	if len(positions) == 0 {
		return "无持仓"
	}

	total := 0.0
	summary := ""
	for symbol, pos := range positions {
		total += pos
		summary += fmt.Sprintf("%s: %.1f%%, ", symbol, pos)
	}

	summary += fmt.Sprintf("总计: %.1f%%", total)
	return summary
}

