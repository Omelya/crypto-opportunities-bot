package defillama

// Pool представляє liquidity pool від DeFiLlama API
type Pool struct {
	Chain          string   `json:"chain"`
	Project        string   `json:"project"`
	Symbol         string   `json:"symbol"`
	PoolID         string   `json:"pool"`
	TVL            float64  `json:"tvlUsd"`
	APY            float64  `json:"apy"`
	APYBase        float64  `json:"apyBase"`
	APYReward      float64  `json:"apyReward"`
	APYMean30d     float64  `json:"apyMean30d"`
	Volume1d       float64  `json:"volumeUsd1d"`
	Volume7d       float64  `json:"volumeUsd7d"`
	IL7d           float64  `json:"il7d"`
	ILRisk         string   `json:"ilRisk"`
	RewardTokens   []string `json:"rewardTokens"`
	UnderlyingTokens []string `json:"underlyingTokens"`
	PoolMeta       string   `json:"poolMeta"`
	PredictedClass string   `json:"predictedClass"`
	Stablecoin     bool     `json:"stablecoin"`
	Count          int      `json:"count"`
}

// PoolsResponse відповідь від /pools endpoint
type PoolsResponse struct {
	Status string `json:"status"`
	Data   []Pool `json:"data"`
}

// Protocol представляє DeFi протокол
type Protocol struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Address        string  `json:"address"`
	Symbol         string  `json:"symbol"`
	URL            string  `json:"url"`
	Description    string  `json:"description"`
	Chain          string  `json:"chain"`
	Logo           string  `json:"logo"`
	Audits         string  `json:"audits"`
	AuditNote      string  `json:"audit_note"`
	Gecko          string  `json:"gecko_id"`
	CMC            string  `json:"cmcId"`
	Category       string  `json:"category"`
	Chains         []string `json:"chains"`
	Module         string  `json:"module"`
	Twitter        string  `json:"twitter"`
	Forked         string  `json:"forkedFrom"`
	OracleSource   string  `json:"oracles"`
	LiquiditySource string `json:"liquisity_mining"`
	TVL            float64 `json:"tvl"`
	ChainTVLs      map[string]float64 `json:"chainTvls"`
	Change1h       float64 `json:"change_1h"`
	Change1d       float64 `json:"change_1d"`
	Change7d       float64 `json:"change_7d"`
}

// ProtocolsResponse відповідь від /protocols endpoint
type ProtocolsResponse []Protocol

// Yield представляє yield farming opportunity
type Yield struct {
	Chain         string   `json:"chain"`
	Project       string   `json:"project"`
	Symbol        string   `json:"symbol"`
	TVL           float64  `json:"tvlUsd"`
	APY           float64  `json:"apy"`
	APYBase       float64  `json:"apyBase"`
	APYReward     float64  `json:"apyReward"`
	PoolID        string   `json:"pool"`
	RewardTokens  []string `json:"rewardTokens"`
}

// YieldsResponse відповідь від /yields endpoint
type YieldsResponse struct {
	Status string  `json:"status"`
	Data   []Yield `json:"data"`
}

// ChainResponse відповідь для chain-specific запитів
type ChainResponse struct {
	Chain string `json:"chain"`
	Pools []Pool `json:"pools"`
}
