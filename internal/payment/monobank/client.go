package monobank

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// API endpoints
	baseURL        = "https://api.monobank.ua"
	createInvoice  = "/api/merchant/invoice/create"
	invoiceStatus  = "/api/merchant/invoice/status"
	invoiceCancel  = "/api/merchant/invoice/cancel"
	invoiceRemove  = "/api/merchant/invoice/remove"

	// Підписка на statement (для перевірки recurring платежів)
	merchantDetails = "/api/merchant/details"
)

// Client Monobank API клієнт
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient створює новий Monobank клієнт
func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateInvoice створює рахунок для оплати
func (c *Client) CreateInvoice(req *InvoiceRequest) (*InvoiceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+createInvoice, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, &errResp
	}

	var invoiceResp InvoiceResponse
	if err := json.Unmarshal(respBody, &invoiceResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &invoiceResp, nil
}

// GetInvoiceStatus отримує статус рахунку
func (c *Client) GetInvoiceStatus(invoiceID string) (*WebhookPayload, error) {
	url := fmt.Sprintf("%s%s?invoiceId=%s", c.baseURL, invoiceStatus, invoiceID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("X-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, &errResp
	}

	var status WebhookPayload
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &status, nil
}

// CancelInvoice скасовує рахунок
func (c *Client) CancelInvoice(invoiceID string) error {
	req := map[string]string{"invoiceId": invoiceID}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+invoiceCancel, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &errResp
	}

	return nil
}

// VerifyWebhookSignature перевіряє підпис webhook
func (c *Client) VerifyWebhookSignature(pubKey, body, signature string) bool {
	// Створюємо підпис: base64(sha256(pubkey + body))
	hash := sha256.Sum256([]byte(pubKey + body))
	expectedSignature := base64.StdEncoding.EncodeToString(hash[:])

	return expectedSignature == signature
}

// CreateSubscriptionInvoice створює рахунок з можливістю збереження картки
func (c *Client) CreateSubscriptionInvoice(req *InvoiceRequest, saveCard bool) (*InvoiceResponse, error) {
	if saveCard {
		req.SaveCardData = &SaveCardData{
			SaveCard: true,
		}
	}

	return c.CreateInvoice(req)
}

// CreateRecurringPayment створює recurring платіж по збереженій картці
// Примітка: Monobank не має прямого API для recurring, це треба робити через збережені картки
// і створення нових invoice з walletId
func (c *Client) CreateRecurringPayment(walletID string, req *InvoiceRequest) (*InvoiceResponse, error) {
	if req.SaveCardData == nil {
		req.SaveCardData = &SaveCardData{}
	}
	req.SaveCardData.WalletId = walletID

	return c.CreateInvoice(req)
}
