package models

import "time"

const (
	WhaleChainEthereum = "ethereum"
	WhaleChainBSC      = "bsc"
	WhaleChainPolygon  = "polygon"
	WhaleChainArbitrum = "arbitrum"
	WhaleChainOptimism = "optimism"
)

const (
	WhaleDirectionExchangeToWallet = "exchange_to_wallet" // Possible accumulation
	WhaleDirectionWalletToExchange = "wallet_to_exchange" // Possible sell signal
	WhaleDirectionWalletToWallet   = "wallet_to_wallet"   // Whale transfer
	WhaleDirectionUnknown          = "unknown"
)

const (
	WhaleStatusNew       = "new"       // Just detected
	WhaleStatusNotified  = "notified"  // Notifications sent
	WhaleStatusProcessed = "processed" // Analyzed
)

// WhaleTransaction represents a large cryptocurrency transaction
type WhaleTransaction struct {
	BaseModel

	Chain          string  `gorm:"index;not null" json:"chain"`                // ethereum, bsc, polygon, etc.
	TxHash         string  `gorm:"uniqueIndex;not null" json:"tx_hash"`        // Transaction hash
	Token          string  `gorm:"index;not null" json:"token"`                // Token symbol (ETH, BTC, USDT, etc.)
	TokenAddress   string  `json:"token_address,omitempty"`                    // Token contract address
	Amount         float64 `gorm:"not null" json:"amount"`                     // Amount in token units
	AmountUSD      float64 `gorm:"index;not null" json:"amount_usd"`           // Amount in USD
	FromAddress    string  `gorm:"index;not null" json:"from_address"`         // Sender address
	ToAddress      string  `gorm:"index;not null" json:"to_address"`           // Receiver address
	FromLabel      string  `json:"from_label,omitempty"`                       // Known address label (Binance, Coinbase, etc.)
	ToLabel        string  `json:"to_label,omitempty"`                         // Known address label
	Direction      string  `gorm:"index" json:"direction"`                     // exchange_to_wallet, wallet_to_exchange, etc.
	BlockNumber    uint64  `gorm:"index" json:"block_number"`                  // Block number
	BlockTimestamp int64   `gorm:"index" json:"block_timestamp"`               // Unix timestamp
	GasUsed        uint64  `json:"gas_used,omitempty"`                         // Gas used
	GasPrice       uint64  `json:"gas_price,omitempty"`                        // Gas price in wei
	Status         string  `gorm:"index;default:'new'" json:"status"`          // new, notified, processed
	IsNotified     bool    `gorm:"default:false" json:"is_notified"`           // Whether users were notified
	ExplorerURL    string  `json:"explorer_url"`                               // Blockchain explorer URL

	// Historical analysis
	HistoricalOutcome string  `json:"historical_outcome,omitempty"` // What happened after similar transactions
	PriceChange24h    float64 `json:"price_change_24h,omitempty"`   // Price change 24h after transaction

	Metadata JSONMap `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
}

func (*WhaleTransaction) TableName() string {
	return "whale_transactions"
}

// IsExchangeToWallet checks if this is a potential accumulation signal
func (wt *WhaleTransaction) IsExchangeToWallet() bool {
	return wt.Direction == WhaleDirectionExchangeToWallet
}

// IsWalletToExchange checks if this is a potential sell signal
func (wt *WhaleTransaction) IsWalletToExchange() bool {
	return wt.Direction == WhaleDirectionWalletToExchange
}

// IsMegaWhale checks if transaction is > $10M
func (wt *WhaleTransaction) IsMegaWhale() bool {
	return wt.AmountUSD >= 10000000
}

// IsLargeWhale checks if transaction is $5M-$10M
func (wt *WhaleTransaction) IsLargeWhale() bool {
	return wt.AmountUSD >= 5000000 && wt.AmountUSD < 10000000
}

// IsMediumWhale checks if transaction is $1M-$5M
func (wt *WhaleTransaction) IsMediumWhale() bool {
	return wt.AmountUSD >= 1000000 && wt.AmountUSD < 5000000
}

// GetWhaleSize returns a human-readable whale size
func (wt *WhaleTransaction) GetWhaleSize() string {
	if wt.IsMegaWhale() {
		return "Mega Whale (>$10M)"
	} else if wt.IsLargeWhale() {
		return "Large Whale ($5M-$10M)"
	} else if wt.IsMediumWhale() {
		return "Medium Whale ($1M-$5M)"
	}
	return "Whale (>$1M)"
}

// GetTimeAgo returns a human-readable time since transaction
func (wt *WhaleTransaction) GetTimeAgo() string {
	txTime := time.Unix(wt.BlockTimestamp, 0)
	duration := time.Since(txTime)

	if duration.Minutes() < 60 {
		return "just now"
	} else if duration.Hours() < 24 {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	}
}

// GetDirectionEmoji returns emoji based on direction
func (wt *WhaleTransaction) GetDirectionEmoji() string {
	switch wt.Direction {
	case WhaleDirectionExchangeToWallet:
		return "📥" // Incoming - potential accumulation
	case WhaleDirectionWalletToExchange:
		return "📤" // Outgoing - potential sell
	case WhaleDirectionWalletToWallet:
		return "↔️" // Transfer
	default:
		return "🔄"
	}
}

// GetSignalInterpretation returns a human-readable interpretation
func (wt *WhaleTransaction) GetSignalInterpretation() string {
	switch wt.Direction {
	case WhaleDirectionExchangeToWallet:
		return "🟢 Potential Accumulation - Bullish Signal"
	case WhaleDirectionWalletToExchange:
		return "🔴 Potential Distribution - Bearish Signal"
	case WhaleDirectionWalletToWallet:
		return "🟡 Whale Transfer - Neutral"
	default:
		return "⚪ Unknown Direction"
	}
}

// WhaleStats represents aggregated whale statistics
type WhaleStats struct {
	Chain              string  `json:"chain"`
	Token              string  `json:"token"`
	Last24hCount       int     `json:"last_24h_count"`
	Last24hVolume      float64 `json:"last_24h_volume_usd"`
	AccumulationCount  int     `json:"accumulation_count"`  // Exchange to wallet
	DistributionCount  int     `json:"distribution_count"`  // Wallet to exchange
	NetFlow            float64 `json:"net_flow_usd"`        // Accumulation - Distribution
	AverageTxSize      float64 `json:"average_tx_size_usd"`
	LargestTx          float64 `json:"largest_tx_usd"`
	MostActiveExchange string  `json:"most_active_exchange,omitempty"`
}

// KnownAddress represents a labeled/known blockchain address
type KnownAddress struct {
	Address     string `json:"address"`
	Label       string `json:"label"`       // "Binance Hot Wallet", "Coinbase", etc.
	Type        string `json:"type"`        // "exchange", "whale", "defi", "other"
	Exchange    string `json:"exchange,omitempty"`
	Description string `json:"description,omitempty"`
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// Helper for formatting (needs to be imported separately if used)
import "fmt"
