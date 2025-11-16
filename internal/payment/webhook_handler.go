package payment

import (
	"crypto-opportunities-bot/internal/payment/monobank"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// WebhookHandler обробник webhook від Monobank
type WebhookHandler struct {
	paymentService *Service
	publicKey      string // Monobank public key для верифікації підпису
}

func NewWebhookHandler(paymentService *Service, publicKey string) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		publicKey:      publicKey,
	}
}

// HandleMonobankWebhook HTTP handler для Monobank webhook
func (h *WebhookHandler) HandleMonobankWebhook(w http.ResponseWriter, r *http.Request) {
	// Тільки POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаємо body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read webhook body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Верифікуємо підпис (якщо є public key)
	if h.publicKey != "" {
		signature := r.Header.Get("X-Sign")
		if signature == "" {
			log.Printf("⚠️ Webhook without signature")
			http.Error(w, "Missing signature", http.StatusUnauthorized)
			return
		}

		if !h.paymentService.monoClient.VerifyWebhookSignature(h.publicKey, string(body), signature) {
			log.Printf("❌ Invalid webhook signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Парсимо JSON
	var payload monobank.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("❌ Failed to parse webhook JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Monobank webhook: invoice=%s, status=%s, amount=%d",
		payload.InvoiceId, payload.Status, payload.Amount)

	// Обробляємо webhook
	if err := h.paymentService.HandleWebhook(&payload); err != nil {
		log.Printf("❌ Failed to handle webhook: %v", err)
		// Все одно повертаємо 200, щоб Monobank не retry
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Успішна відповідь
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// StartWebhookServer запускає HTTP сервер для webhooks
func StartWebhookServer(handler *WebhookHandler, port string) error {
	mux := http.NewServeMux()

	// Monobank webhook endpoint
	mux.HandleFunc("/webhook/monobank", handler.HandleMonobankWebhook)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	})

	addr := fmt.Sprintf(":%s", port)
	log.Printf("🌐 Webhook server starting on %s", addr)
	log.Printf("   POST /webhook/monobank - Monobank webhook handler")
	log.Printf("   GET  /health - Health check")

	return http.ListenAndServe(addr, mux)
}
