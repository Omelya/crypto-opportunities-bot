package whale

import (
	"crypto-opportunities-bot/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// BlockchainClient provides interface for different blockchain explorers
type BlockchainClient interface {
	GetRecentTransactions(minValueUSD float64) ([]*Transaction, error)
	GetChain() string
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash           string
	From           string
	To             string
	Value          string  // In wei or smallest unit
	ValueDecimal   float64 // In human-readable format
	Token          string
	TokenAddress   string
	BlockNumber    uint64
	BlockTimestamp int64
	GasUsed        uint64
	GasPrice       uint64
}

// EtherscanClient for Ethereum blockchain
type EtherscanClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	chain      string
}

func NewEtherscanClient(apiKey string) *EtherscanClient {
	return &EtherscanClient{
		apiKey:  apiKey,
		baseURL: "https://api.etherscan.io/api",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		chain: "ethereum",
	}
}

func (c *EtherscanClient) GetChain() string {
	return c.chain
}

func (c *EtherscanClient) GetRecentTransactions(minValueUSD float64) ([]*Transaction, error) {
	// Note: Etherscan free API has limitations
	// For production, consider using:
	// 1. Etherscan Pro API
	// 2. Alchemy/Infura webhooks
	// 3. The Graph protocol

	// Get latest block
	latestBlock, err := c.getLatestBlock()
	if err != nil {
		return nil, err
	}

	// Scan recent blocks for large transactions
	transactions := []*Transaction{}

	// Check last 10 blocks
	for i := 0; i < 10; i++ {
		blockNum := latestBlock - uint64(i)
		blockTxs, err := c.getBlockTransactions(blockNum)
		if err != nil {
			continue // Skip failed blocks
		}
		transactions = append(transactions, blockTxs...)
	}

	return transactions, nil
}

func (c *EtherscanClient) getLatestBlock() (uint64, error) {
	url := fmt.Sprintf("%s?module=proxy&action=eth_blockNumber&apikey=%s",
		c.baseURL, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	// Convert hex to uint64
	blockNum, err := strconv.ParseUint(result.Result[2:], 16, 64)
	return blockNum, err
}

func (c *EtherscanClient) getBlockTransactions(blockNum uint64) ([]*Transaction, error) {
	url := fmt.Sprintf("%s?module=proxy&action=eth_getBlockByNumber&tag=0x%x&boolean=true&apikey=%s",
		c.baseURL, blockNum, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Transactions []struct {
				Hash        string `json:"hash"`
				From        string `json:"from"`
				To          string `json:"to"`
				Value       string `json:"value"`
				Gas         string `json:"gas"`
				GasPrice    string `json:"gasPrice"`
				BlockNumber string `json:"blockNumber"`
			} `json:"transactions"`
			Timestamp string `json:"timestamp"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	transactions := []*Transaction{}
	timestamp, _ := strconv.ParseInt(result.Result.Timestamp[2:], 16, 64)

	for _, tx := range result.Result.Transactions {
		// Convert hex value to decimal
		valueWei, _ := strconv.ParseUint(tx.Value[2:], 16, 64)

		// Skip small transactions (< 0.1 ETH)
		if valueWei < 100000000000000000 {
			continue
		}

		valueEth := float64(valueWei) / 1e18

		transactions = append(transactions, &Transaction{
			Hash:           tx.Hash,
			From:           tx.From,
			To:             tx.To,
			Value:          tx.Value,
			ValueDecimal:   valueEth,
			Token:          "ETH",
			TokenAddress:   "",
			BlockNumber:    blockNum,
			BlockTimestamp: timestamp,
		})
	}

	return transactions, nil
}

// BSCScanClient for Binance Smart Chain
type BSCScanClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	chain      string
}

func NewBSCScanClient(apiKey string) *BSCScanClient {
	return &BSCScanClient{
		apiKey:  apiKey,
		baseURL: "https://api.bscscan.com/api",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		chain: "bsc",
	}
}

func (c *BSCScanClient) GetChain() string {
	return c.chain
}

func (c *BSCScanClient) GetRecentTransactions(minValueUSD float64) ([]*Transaction, error) {
	// Similar implementation to Etherscan
	// BSCScan API is compatible with Etherscan API

	latestBlock, err := c.getLatestBlock()
	if err != nil {
		return nil, err
	}

	transactions := []*Transaction{}

	for i := 0; i < 10; i++ {
		blockNum := latestBlock - uint64(i)
		blockTxs, err := c.getBlockTransactions(blockNum)
		if err != nil {
			continue
		}
		transactions = append(transactions, blockTxs...)
	}

	return transactions, nil
}

func (c *BSCScanClient) getLatestBlock() (uint64, error) {
	url := fmt.Sprintf("%s?module=proxy&action=eth_blockNumber&apikey=%s",
		c.baseURL, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	blockNum, err := strconv.ParseUint(result.Result[2:], 16, 64)
	return blockNum, err
}

func (c *BSCScanClient) getBlockTransactions(blockNum uint64) ([]*Transaction, error) {
	url := fmt.Sprintf("%s?module=proxy&action=eth_getBlockByNumber&tag=0x%x&boolean=true&apikey=%s",
		c.baseURL, blockNum, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Transactions []struct {
				Hash        string `json:"hash"`
				From        string `json:"from"`
				To          string `json:"to"`
				Value       string `json:"value"`
				BlockNumber string `json:"blockNumber"`
			} `json:"transactions"`
			Timestamp string `json:"timestamp"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	transactions := []*Transaction{}
	timestamp, _ := strconv.ParseInt(result.Result.Timestamp[2:], 16, 64)

	for _, tx := range result.Result.Transactions {
		valueWei, _ := strconv.ParseUint(tx.Value[2:], 16, 64)

		// Skip small transactions (< 0.1 BNB)
		if valueWei < 100000000000000000 {
			continue
		}

		valueBNB := float64(valueWei) / 1e18

		transactions = append(transactions, &Transaction{
			Hash:           tx.Hash,
			From:           tx.From,
			To:             tx.To,
			Value:          tx.Value,
			ValueDecimal:   valueBNB,
			Token:          "BNB",
			TokenAddress:   "",
			BlockNumber:    blockNum,
			BlockTimestamp: timestamp,
		})
	}

	return transactions, nil
}

// Known exchange addresses database
var knownAddresses = map[string]models.KnownAddress{
	// Ethereum - Binance
	"0x28c6c06298d514db089934071355e5743bf21d60": {
		Address:  "0x28c6c06298d514db089934071355e5743bf21d60",
		Label:    "Binance Hot Wallet",
		Type:     "exchange",
		Exchange: "binance",
	},
	"0xdfd5293d8e347dfe59e90efd55b2956a1343963d": {
		Address:  "0xdfd5293d8e347dfe59e90efd55b2956a1343963d",
		Label:    "Binance Cold Wallet",
		Type:     "exchange",
		Exchange: "binance",
	},
	// Coinbase
	"0x503828976d22510aad0201ac7ec88293211d23da": {
		Address:  "0x503828976d22510aad0201ac7ec88293211d23da",
		Label:    "Coinbase Hot Wallet",
		Type:     "exchange",
		Exchange: "coinbase",
	},
	// Kraken
	"0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0": {
		Address:  "0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0",
		Label:    "Kraken Exchange",
		Type:     "exchange",
		Exchange: "kraken",
	},
	// Add more known addresses as needed
}

func GetAddressLabel(address string) (string, bool) {
	if known, exists := knownAddresses[address]; exists {
		return known.Label, true
	}
	return "", false
}

func IsExchangeAddress(address string) bool {
	if known, exists := knownAddresses[address]; exists {
		return known.Type == "exchange"
	}
	return false
}
