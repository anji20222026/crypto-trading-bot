package executors

import (
	"context"
	"fmt"
	"sort"

	"github.com/oak/crypto-trading-bot/internal/config"
	"github.com/oak/crypto-trading-bot/internal/logger"
)

// FundManager manages fund allocation across multiple trading pairs
// FundManager 管理多个交易对的资金分配
type FundManager struct {
	config   *config.Config
	executor *BinanceExecutor
	logger   *logger.ColorLogger
}

// NewFundManager creates a new fund manager
// NewFundManager 创建新的资金管理器
func NewFundManager(cfg *config.Config, executor *BinanceExecutor, log *logger.ColorLogger) *FundManager {
	return &FundManager{
		config:   cfg,
		executor: executor,
		logger:   log,
	}
}

// TradingDecisionWithConfidence represents a trading decision with confidence
// TradingDecisionWithConfidence 表示带置信度的交易决策
type TradingDecisionWithConfidence struct {
	Symbol     string
	Action     TradeAction
	Confidence float64
	// Note: PositionSize is NOT used - fund manager calculates allocation independently
	// 注意：不使用 PositionSize - 资金管理器独立计算分配
}

// FundAllocationResult represents the result of fund allocation
// FundAllocationResult 表示资金分配结果
type FundAllocationResult struct {
	Symbol               string
	AllocatedPercent     float64 // 分配的资金百分比 / Allocated fund percentage
	CanTrade             bool    // 是否可以交易 / Whether can trade
	Reason               string  // 原因 / Reason
	AdjustedByConfidence bool    // 是否根据置信度调整 / Whether adjusted by confidence
}

// AllocateFunds allocates funds to multiple trading pairs based on usage limit
// AllocateFunds 根据使用率限制为多个交易对分配资金
func (fm *FundManager) AllocateFunds(ctx context.Context, decisions []*TradingDecisionWithConfidence) (map[string]*FundAllocationResult, error) {
	fm.logger.Header("资金分配管理器", '=', 80)

	// Step 1: Get account balance and current positions
	// 步骤 1: 获取账户余额和当前持仓
	balance, err := fm.executor.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取账户余额失败: %w", err)
	}
	fm.logger.Info(fmt.Sprintf("💰 账户总余额: %.2f USDT", balance))

	// Step 2: Calculate current fund usage
	// 步骤 2: 计算当前资金使用率
	totalMarginUsed := 0.0
	for _, decision := range decisions {
		position, err := fm.executor.GetCurrentPosition(ctx, decision.Symbol)
		if err == nil && position != nil {
			// Calculate margin used: position_value / leverage
			// 计算已用保证金: 持仓价值 / 杠杆
			positionValue := position.Size * position.EntryPrice
			marginUsed := positionValue / float64(position.Leverage)
			totalMarginUsed += marginUsed
			fm.logger.Info(fmt.Sprintf("  %s: 持仓价值 $%.2f, 杠杆 %dx, 保证金 $%.2f",
				decision.Symbol, positionValue, position.Leverage, marginUsed))
		}
	}

	currentUsagePercent := (totalMarginUsed / balance) * 100
	fm.logger.Info(fmt.Sprintf("📊 当前资金使用率: %.2f%% (%.2f / %.2f USDT)", currentUsagePercent, totalMarginUsed, balance))

	// Step 3: Check if usage exceeds 70%
	// 步骤 3: 检查使用率是否超过 70%
	maxUsagePercent := 70.0
	if currentUsagePercent >= maxUsagePercent {
		fm.logger.Warning(fmt.Sprintf("⚠️  当前资金使用率 %.2f%% ≥ %.2f%%，拒绝所有新交易", currentUsagePercent, maxUsagePercent))
		results := make(map[string]*FundAllocationResult)
		for _, decision := range decisions {
			results[decision.Symbol] = &FundAllocationResult{
				Symbol:           decision.Symbol,
				AllocatedPercent: 0,
				CanTrade:         false,
				Reason:           fmt.Sprintf("资金使用率 %.2f%% 已达上限 %.2f%%", currentUsagePercent, maxUsagePercent),
			}
		}
		return results, nil
	}

	// Step 4: Calculate available funds
	// 步骤 4: 计算可用资金
	availableUsagePercent := maxUsagePercent - currentUsagePercent
	availableFunds := balance * (availableUsagePercent / 100)
	fm.logger.Success(fmt.Sprintf("✅ 可用资金使用率: %.2f%% (%.2f USDT)", availableUsagePercent, availableFunds))

	// Step 5: Filter valid trading decisions (only BUY/SELL, not HOLD)
	// 步骤 5: 过滤有效的交易决策（只有 BUY/SELL，不包括 HOLD）
	validDecisions := []*TradingDecisionWithConfidence{}
	for _, decision := range decisions {
		if decision.Action == ActionBuy || decision.Action == ActionSell {
			validDecisions = append(validDecisions, decision)
		}
	}

	if len(validDecisions) == 0 {
		fm.logger.Info("ℹ️  没有需要开仓的交易对")
		return make(map[string]*FundAllocationResult), nil
	}

	fm.logger.Info(fmt.Sprintf("📋 需要开仓的交易对数量: %d", len(validDecisions)))

	// Step 6: Allocate 30% of available funds, evenly distributed
	// 步骤 6: 分配可用资金的 30%，平均分配
	allocationRatio := 0.30
	totalAllocationFunds := availableFunds * allocationRatio
	perSymbolFunds := totalAllocationFunds / float64(len(validDecisions))
	perSymbolPercent := (perSymbolFunds / balance) * 100

	fm.logger.Info(fmt.Sprintf("💡 分配策略: 可用资金的 %.0f%% = %.2f USDT", allocationRatio*100, totalAllocationFunds))
	fm.logger.Info(fmt.Sprintf("📐 平均每个交易对: %.2f USDT (%.2f%%)", perSymbolFunds, perSymbolPercent))

	// Step 7: Check if allocation exceeds limit
	// 步骤 7: 检查分配后是否超过限制
	results := make(map[string]*FundAllocationResult)

	projectedUsagePercent := currentUsagePercent + (totalAllocationFunds/balance)*100
	if projectedUsagePercent <= maxUsagePercent {
		// All symbols can be allocated evenly
		// 所有交易对可以平均分配
		fm.logger.Success(fmt.Sprintf("✅ 预计使用率: %.2f%% ≤ %.2f%%，所有交易对平均分配", projectedUsagePercent, maxUsagePercent))
		for _, decision := range validDecisions {
			results[decision.Symbol] = &FundAllocationResult{
				Symbol:           decision.Symbol,
				AllocatedPercent: perSymbolPercent,
				CanTrade:         true,
				Reason:           fmt.Sprintf("平均分配 %.2f%%", perSymbolPercent),
			}
		}
	} else {
		// Exceeds limit, prioritize by confidence
		// 超过限制，按置信度优先分配
		fm.logger.Warning(fmt.Sprintf("⚠️  预计使用率 %.2f%% > %.2f%%，按置信度优先分配", projectedUsagePercent, maxUsagePercent))

		// Sort by confidence (descending)
		// 按置信度排序（降序）
		sort.Slice(validDecisions, func(i, j int) bool {
			return validDecisions[i].Confidence > validDecisions[j].Confidence
		})

		remainingFunds := availableFunds
		for _, decision := range validDecisions {
			if remainingFunds <= 0 {
				results[decision.Symbol] = &FundAllocationResult{
					Symbol:               decision.Symbol,
					AllocatedPercent:     0,
					CanTrade:             false,
					Reason:               "可用资金已耗尽",
					AdjustedByConfidence: true,
				}
				continue
			}

			// Allocate funds to this symbol
			// 为该交易对分配资金
			allocatedFunds := perSymbolFunds
			if allocatedFunds > remainingFunds {
				allocatedFunds = remainingFunds
			}
			allocatedPercent := (allocatedFunds / balance) * 100

			results[decision.Symbol] = &FundAllocationResult{
				Symbol:               decision.Symbol,
				AllocatedPercent:     allocatedPercent,
				CanTrade:             true,
				Reason:               fmt.Sprintf("置信度 %.2f，优先分配 %.2f%%", decision.Confidence, allocatedPercent),
				AdjustedByConfidence: true,
			}

			remainingFunds -= allocatedFunds
			fm.logger.Info(fmt.Sprintf("  ✅ %s: 置信度 %.2f, 分配 %.2f%% (剩余 %.2f USDT)",
				decision.Symbol, decision.Confidence, allocatedPercent, remainingFunds))
		}
	}

	return results, nil
}

