package executors

import (
	"github.com/oak/crypto-trading-bot/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPositionManager_AllocateWeighted(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	cfg := &config.Config{
		TotalPositionLimit:     60.0,
		MaxSymbolPosition:      30.0,
		MinSymbolPosition:      10.0,
		PositionAllocationMode: "weighted",
	}

	pm := NewPositionManager(cfg, logger)

	t.Run("两个币种不同置信度", func(t *testing.T) {
		decisions := map[string]*TradeDecision{
			"BTC/USDT": {
				Symbol:     "BTC/USDT",
				Action:     "BUY",
				Confidence: 0.8,
			},
			"ETH/USDT": {
				Symbol:     "ETH/USDT",
				Action:     "BUY",
				Confidence: 0.6,
			},
		}

		positions := pm.AllocatePositions(decisions)

		// BTC 置信度更高，应该获得更大仓位
		assert.Greater(t, positions["BTC/USDT"], positions["ETH/USDT"])

		// 总仓位不应超过限制
		total := positions["BTC/USDT"] + positions["ETH/USDT"]
		assert.LessOrEqual(t, total, cfg.TotalPositionLimit)

		// 单币种不应超过限制
		assert.LessOrEqual(t, positions["BTC/USDT"], cfg.MaxSymbolPosition)
		assert.LessOrEqual(t, positions["ETH/USDT"], cfg.MaxSymbolPosition)
	})

	t.Run("一个 HOLD 一个 BUY", func(t *testing.T) {
		decisions := map[string]*TradeDecision{
			"BTC/USDT": {
				Symbol:     "BTC/USDT",
				Action:     "BUY",
				Confidence: 0.7,
			},
			"ETH/USDT": {
				Symbol:     "ETH/USDT",
				Action:     "HOLD",
				Confidence: 0.3,
			},
		}

		positions := pm.AllocatePositions(decisions)

		// 只有 BTC 应该有仓位
		assert.Greater(t, positions["BTC/USDT"], 0.0)
		assert.Equal(t, 0.0, positions["ETH/USDT"])

		// BTC 应该受单币种上限限制
		assert.LessOrEqual(t, positions["BTC/USDT"], cfg.MaxSymbolPosition)
	})

	t.Run("三个币种均分场景", func(t *testing.T) {
		decisions := map[string]*TradeDecision{
			"BTC/USDT": {
				Symbol:     "BTC/USDT",
				Action:     "BUY",
				Confidence: 0.5,
			},
			"ETH/USDT": {
				Symbol:     "ETH/USDT",
				Action:     "BUY",
				Confidence: 0.5,
			},
			"SOL/USDT": {
				Symbol:     "SOL/USDT",
				Action:     "SELL",
				Confidence: 0.5,
			},
		}

		positions := pm.AllocatePositions(decisions)

		// 置信度相同，应该接近均分
		assert.InDelta(t, positions["BTC/USDT"], positions["ETH/USDT"], 0.1)
		assert.InDelta(t, positions["ETH/USDT"], positions["SOL/USDT"], 0.1)

		// 总仓位应该接近上限
		total := positions["BTC/USDT"] + positions["ETH/USDT"] + positions["SOL/USDT"]
		assert.InDelta(t, total, cfg.TotalPositionLimit, 1.0)
	})
}

func TestPositionManager_AllocateEqual(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := &config.Config{
		TotalPositionLimit:     60.0,
		MaxSymbolPosition:      30.0,
		MinSymbolPosition:      10.0,
		PositionAllocationMode: "equal",
	}

	pm := NewPositionManager(cfg, logger)

	t.Run("三个信号均分", func(t *testing.T) {
		decisions := map[string]*TradeDecision{
			"BTC/USDT": {Symbol: "BTC/USDT", Action: "BUY", Confidence: 0.9},
			"ETH/USDT": {Symbol: "ETH/USDT", Action: "BUY", Confidence: 0.5},
			"SOL/USDT": {Symbol: "SOL/USDT", Action: "SELL", Confidence: 0.6},
		}

		positions := pm.AllocatePositions(decisions)

		// 应该完全均分（忽略置信度）
		assert.Equal(t, positions["BTC/USDT"], positions["ETH/USDT"])
		assert.Equal(t, positions["ETH/USDT"], positions["SOL/USDT"])

		// 每个应该是 60% / 3 = 20%
		assert.InDelta(t, 20.0, positions["BTC/USDT"], 0.1)
	})
}

func TestPositionManager_AllocateFixed(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := &config.Config{
		TotalPositionLimit:     60.0,
		MaxSymbolPosition:      30.0,
		MinSymbolPosition:      10.0,
		PositionAllocationMode: "fixed",
		FixedPositionSize:      25.0,
	}

	pm := NewPositionManager(cfg, logger)

	t.Run("固定仓位", func(t *testing.T) {
		decisions := map[string]*TradeDecision{
			"BTC/USDT": {Symbol: "BTC/USDT", Action: "BUY", Confidence: 0.9},
			"ETH/USDT": {Symbol: "ETH/USDT", Action: "BUY", Confidence: 0.5},
		}

		positions := pm.AllocatePositions(decisions)

		// 每个都应该是固定的 25%
		assert.Equal(t, 25.0, positions["BTC/USDT"])
		assert.Equal(t, 25.0, positions["ETH/USDT"])
	})
}

