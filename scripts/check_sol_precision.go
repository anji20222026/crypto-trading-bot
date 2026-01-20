package main

import (
	"context"
	"fmt"
	"os"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/oak/crypto-trading-bot/internal/config"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create Binance futures client
	futures.UseTestnet = cfg.BinanceTestMode
	client := futures.NewClient(cfg.BinanceAPIKey, cfg.BinanceAPISecret)

	// Get exchange info
	exchangeInfo, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get exchange info: %v\n", err)
		os.Exit(1)
	}

	// Find SOL/USDT symbol
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Symbol == "SOLUSDT" {
			fmt.Printf("Symbol: %s\n", symbol.Symbol)
			fmt.Printf("Status: %s\n", symbol.Status)
			fmt.Printf("Base Asset: %s\n", symbol.BaseAsset)
			fmt.Printf("Quote Asset: %s\n", symbol.QuoteAsset)
			fmt.Printf("Price Precision: %d\n", symbol.PricePrecision)
			fmt.Printf("Quantity Precision: %d\n", symbol.QuantityPrecision)
			fmt.Printf("Base Asset Precision: %d\n", symbol.BaseAssetPrecision)
			fmt.Printf("Quote Precision: %d\n", symbol.QuotePrecision)
			
			fmt.Println("\nFilters:")
			for _, filter := range symbol.Filters {
				fmt.Printf("  %s: %+v\n", filter["filterType"], filter)
			}
			
			return
		}
	}

	fmt.Println("SOL/USDT not found")
}

