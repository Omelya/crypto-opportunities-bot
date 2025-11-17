package defillama

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	BaseURL         = "https://yields.llama.fi"
	RequestTimeout  = 30 * time.Second
	MaxRetries      = 3
	RetryDelay      = 2 * time.Second
)

// Client для роботи з DeFiLlama API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient створює новий DeFiLlama API client
func NewClient() *Client {
	return &Client{
		baseURL: BaseURL,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

// GetPools отримує всі pools
func (c *Client) GetPools() ([]Pool, error) {
	url := fmt.Sprintf("%s/pools", c.baseURL)

	var response PoolsResponse
	err := c.doRequest(url, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	return response.Data, nil
}

// GetPoolsByChain отримує pools для конкретного chain
func (c *Client) GetPoolsByChain(chain string) ([]Pool, error) {
	// Get all pools and filter by chain
	pools, err := c.GetPools()
	if err != nil {
		return nil, err
	}

	var filtered []Pool
	for _, pool := range pools {
		if pool.Chain == chain {
			filtered = append(filtered, pool)
		}
	}

	return filtered, nil
}

// GetPoolsByProtocol отримує pools для конкретного protocol
func (c *Client) GetPoolsByProtocol(protocol string) ([]Pool, error) {
	// Get all pools and filter by protocol
	pools, err := c.GetPools()
	if err != nil {
		return nil, err
	}

	var filtered []Pool
	for _, pool := range pools {
		if pool.Project == protocol {
			filtered = append(filtered, pool)
		}
	}

	return filtered, nil
}

// GetHighAPYPools отримує pools з високим APY
func (c *Client) GetHighAPYPools(minAPY float64) ([]Pool, error) {
	pools, err := c.GetPools()
	if err != nil {
		return nil, err
	}

	var filtered []Pool
	for _, pool := range pools {
		if pool.APY >= minAPY {
			filtered = append(filtered, pool)
		}
	}

	return filtered, nil
}

// GetStablePools отримує стабільні pools (low IL risk)
func (c *Client) GetStablePools() ([]Pool, error) {
	pools, err := c.GetPools()
	if err != nil {
		return nil, err
	}

	var filtered []Pool
	for _, pool := range pools {
		if pool.Stablecoin || pool.IL7d < 2.0 {
			filtered = append(filtered, pool)
		}
	}

	return filtered, nil
}

// GetProtocols отримує інформацію про всі протоколи
func (c *Client) GetProtocols() ([]Protocol, error) {
	url := "https://api.llama.fi/protocols"

	var response ProtocolsResponse
	err := c.doRequest(url, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocols: %w", err)
	}

	return response, nil
}

// GetProtocol отримує інформацію про конкретний протокол
func (c *Client) GetProtocol(slug string) (*Protocol, error) {
	url := fmt.Sprintf("https://api.llama.fi/protocol/%s", slug)

	var protocol Protocol
	err := c.doRequest(url, &protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol %s: %w", slug, err)
	}

	return &protocol, nil
}

// FilterPools фільтрує pools за критеріями
func (c *Client) FilterPools(filters PoolFilters) ([]Pool, error) {
	pools, err := c.GetPools()
	if err != nil {
		return nil, err
	}

	var filtered []Pool
	for _, pool := range pools {
		if c.matchesFilters(pool, filters) {
			filtered = append(filtered, pool)
		}
	}

	return filtered, nil
}

// PoolFilters критерії фільтрації
type PoolFilters struct {
	Chains       []string
	Protocols    []string
	MinAPY       float64
	MaxAPY       float64
	MinTVL       float64
	MaxTVL       float64
	MaxIL        float64
	OnlyStable   bool
	MinVolume24h float64
}

// matchesFilters перевіряє чи pool відповідає фільтрам
func (c *Client) matchesFilters(pool Pool, filters PoolFilters) bool {
	// Chain filter
	if len(filters.Chains) > 0 {
		found := false
		for _, chain := range filters.Chains {
			if pool.Chain == chain {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Protocol filter
	if len(filters.Protocols) > 0 {
		found := false
		for _, protocol := range filters.Protocols {
			if pool.Project == protocol {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// APY range
	if filters.MinAPY > 0 && pool.APY < filters.MinAPY {
		return false
	}
	if filters.MaxAPY > 0 && pool.APY > filters.MaxAPY {
		return false
	}

	// TVL range
	if filters.MinTVL > 0 && pool.TVL < filters.MinTVL {
		return false
	}
	if filters.MaxTVL > 0 && pool.TVL > filters.MaxTVL {
		return false
	}

	// IL risk
	if filters.MaxIL > 0 && pool.IL7d > filters.MaxIL {
		return false
	}

	// Only stable
	if filters.OnlyStable && !pool.Stablecoin && pool.IL7d > 2.0 {
		return false
	}

	// Min volume
	if filters.MinVolume24h > 0 && pool.Volume1d < filters.MinVolume24h {
		return false
	}

	return true
}

// doRequest виконує HTTP запит з retry логікою
func (c *Client) doRequest(url string, result interface{}) error {
	var lastErr error

	for i := 0; i < MaxRetries; i++ {
		if i > 0 {
			log.Printf("Retrying request to %s (attempt %d/%d)", url, i+1, MaxRetries)
			time.Sleep(RetryDelay * time.Duration(i))
		}

		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)

			// Read error body for debugging
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Error response from %s: %s", url, string(body))

			// Don't retry on 4xx errors
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return lastErr
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		err = json.Unmarshal(body, result)
		if err != nil {
			lastErr = fmt.Errorf("failed to unmarshal response: %w", err)
			continue
		}

		// Success
		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", MaxRetries, lastErr)
}

// GetPoolStats отримує статистику по pools
func (c *Client) GetPoolStats() (*PoolStats, error) {
	pools, err := c.GetPools()
	if err != nil {
		return nil, err
	}

	stats := &PoolStats{
		TotalPools: len(pools),
		ByChain:    make(map[string]int),
		ByProtocol: make(map[string]int),
	}

	var totalTVL, totalAPY float64
	var maxAPY, minAPY float64 = 0, 999999

	for _, pool := range pools {
		stats.ByChain[pool.Chain]++
		stats.ByProtocol[pool.Project]++

		totalTVL += pool.TVL
		totalAPY += pool.APY

		if pool.APY > maxAPY {
			maxAPY = pool.APY
		}
		if pool.APY < minAPY && pool.APY > 0 {
			minAPY = pool.APY
		}
	}

	stats.TotalTVL = totalTVL
	stats.AverageAPY = totalAPY / float64(len(pools))
	stats.MaxAPY = maxAPY
	stats.MinAPY = minAPY

	return stats, nil
}

// PoolStats статистика по pools
type PoolStats struct {
	TotalPools int
	TotalTVL   float64
	AverageAPY float64
	MaxAPY     float64
	MinAPY     float64
	ByChain    map[string]int
	ByProtocol map[string]int
}
