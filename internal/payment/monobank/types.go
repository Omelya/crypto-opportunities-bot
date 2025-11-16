package monobank

import "time"

// Invoice створення рахунку для оплати
type InvoiceRequest struct {
	Amount          int64              `json:"amount"`          // Сума в мінімальних одиницях (копійки для UAH)
	Ccy             int                `json:"ccy,omitempty"`   // ISO 4217 (980 = UAH, 840 = USD)
	MerchantPaymInfo MerchantPaymInfo  `json:"merchantPaymInfo"`
	RedirectUrl     string             `json:"redirectUrl"`
	WebHookUrl      string             `json:"webHookUrl"`
	Validity        int64              `json:"validity,omitempty"` // Час дії в секундах
	PaymentType     string             `json:"paymentType,omitempty"` // debit, hold
	QrId            string             `json:"qrId,omitempty"`
	SaveCardData    *SaveCardData      `json:"saveCardData,omitempty"` // Для recurring
}

type MerchantPaymInfo struct {
	Reference       string `json:"reference"`       // Унікальний ID платежу
	Destination     string `json:"destination"`     // Призначення платежу
	Comment         string `json:"comment,omitempty"`
	CustomerEmails  []string `json:"customerEmails,omitempty"`
	BasketOrder     []BasketItem `json:"basketOrder,omitempty"`
}

type BasketItem struct {
	Name     string  `json:"name"`
	Qty      float64 `json:"qty"`
	Sum      int64   `json:"sum"`
	Icon     string  `json:"icon,omitempty"`
	Unit     string  `json:"unit,omitempty"`
	Code     string  `json:"code,omitempty"`
	BarCode  string  `json:"barCode,omitempty"`
	Header   string  `json:"header,omitempty"`
	Footer   string  `json:"footer,omitempty"`
	Tax      []int   `json:"tax,omitempty"`
	ZeroTax  bool    `json:"zeroTax,omitempty"`
}

type SaveCardData struct {
	SaveCard bool   `json:"saveCard"` // Зберегти картку для recurring
	WalletId string `json:"walletId,omitempty"` // ID гаманця для recurring
}

// InvoiceResponse відповідь при створенні рахунку
type InvoiceResponse struct {
	InvoiceId  string `json:"invoiceId"`
	PageUrl    string `json:"pageUrl"`
	ErrCode    string `json:"errCode,omitempty"`
	ErrText    string `json:"errText,omitempty"`
}

// WebhookPayload дані від webhook
type WebhookPayload struct {
	InvoiceId       string    `json:"invoiceId"`
	Status          string    `json:"status"` // created, processing, hold, success, failure, reversed, expired
	FailureReason   string    `json:"failureReason,omitempty"`
	Amount          int64     `json:"amount"`
	Ccy             int       `json:"ccy"`
	FinalAmount     int64     `json:"finalAmount"`
	CreatedDate     time.Time `json:"createdDate"`
	ModifiedDate    time.Time `json:"modifiedDate"`
	Reference       string    `json:"reference"` // Наш унікальний ID
	CancelList      []Cancel  `json:"cancelList,omitempty"`
	TranId          string    `json:"tranId,omitempty"`

	// Для recurring
	PaymentInfo     *PaymentInfo `json:"paymentInfo,omitempty"`
}

type Cancel struct {
	Status        string    `json:"status"`
	Amount        int64     `json:"amount"`
	Ccy           int       `json:"ccy"`
	CreatedDate   time.Time `json:"createdDate"`
	ModifiedDate  time.Time `json:"modifiedDate"`
	ApprovalCode  string    `json:"approvalCode"`
	Rrn           string    `json:"rrn"`
	ExtRef        string    `json:"extRef"`
}

type PaymentInfo struct {
	Rrn              string `json:"rrn,omitempty"`
	ApprovalCode     string `json:"approvalCode,omitempty"`
	TransAmount      int64  `json:"transAmount,omitempty"`
	TransCcy         int    `json:"transCcy,omitempty"`
	Fee              int64  `json:"fee,omitempty"`
	WalletId         string `json:"walletId,omitempty"` // Для recurring
	CardPan          string `json:"cardPan,omitempty"`
	PaymentSystem    string `json:"paymentSystem,omitempty"`
}

// Invoice Status Constants
const (
	StatusCreated    = "created"
	StatusProcessing = "processing"
	StatusHold       = "hold"
	StatusSuccess    = "success"
	StatusFailure    = "failure"
	StatusReversed   = "reversed"
	StatusExpired    = "expired"
)

// Currency ISO 4217
const (
	CurrencyUAH = 980
	CurrencyUSD = 840
	CurrencyEUR = 978
)

// Subscription Plans
const (
	PlanPremiumMonthly = "premium_monthly"
	PlanPremiumWeekly  = "premium_weekly"
	PlanPremiumYearly  = "premium_yearly"
)

// Plan Prices (в мінімальних одиницях)
var PlanPrices = map[string]int64{
	PlanPremiumMonthly: 24900, // 249 UAH
	PlanPremiumWeekly:  9900,  // 99 UAH
	PlanPremiumYearly:  249900, // 2499 UAH (знижка ~16%)
}

// Plan Durations
var PlanDurations = map[string]time.Duration{
	PlanPremiumMonthly: 30 * 24 * time.Hour,
	PlanPremiumWeekly:  7 * 24 * time.Hour,
	PlanPremiumYearly:  365 * 24 * time.Hour,
}

// ErrorResponse помилка API
type ErrorResponse struct {
	ErrCode string `json:"errCode"`
	ErrText string `json:"errText"`
}

func (e *ErrorResponse) Error() string {
	return e.ErrText
}
