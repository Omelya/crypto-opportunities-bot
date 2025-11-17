# План Реалізації Premium Trading Client

## Огляд Системи

### Архітектура
```
┌──────────────────────────────────────────────────────────────┐
│                    Main Server (Go)                          │
│  - Arbitrage Detection (Real-time WebSocket to Exchanges)    │
│  - WebSocket Hub for Premium Clients                         │
│  - Authentication & Premium Validation                       │
│  - Statistics Collection API                                 │
└───────────────────┬──────────────────────────────────────────┘
                    │ WebSocket (Secure)
                    │ Message: arbitrage_opportunity
                    │
    ┌───────────────┴───────────────┬─────────────────┐
    │                               │                 │
┌───▼─────────────┐    ┌───────────▼──────┐    ┌────▼──────────┐
│ Premium Client  │    │ Premium Client   │    │Premium Client │
│  (User Device)  │    │  (User Device)   │    │ (User Device) │
│                 │    │                  │    │               │
│ - Auth (JWT)    │    │ - Auth (JWT)     │    │- Auth (JWT)   │
│ - API Keys      │    │ - API Keys       │    │- API Keys     │
│   (Local Store) │    │   (Local Store)  │    │  (Local Store)│
│ - Trading Bot   │    │ - Trading Bot    │    │- Trading Bot  │
│ - Statistics    │    │ - Statistics     │    │- Statistics   │
└─────┬───────────┘    └─────┬────────────┘    └────┬──────────┘
      │                      │                      │
      │ Direct API Calls     │                      │
      ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────┐
│              Exchanges (Binance, Bybit, OKX)                │
│  - Place Orders                                             │
│  - Check Balances                                           │
│  - Get Order Status                                         │
└─────────────────────────────────────────────────────────────┘
```

---

## 📱 ЧАСТИНА 1: Premium Client Application

### Технологічний Стек

**Варіант 1: Native Go Application (Рекомендований)**
- **Переваги**: Швидкість, низькі ресурси, кросплатформенність
- **Framework**: Go 1.25+ з Fyne UI (опціонально для GUI)
- **WebSocket**: gorilla/websocket
- **Exchange APIs**: ccxt-like library або custom implementations
- **Crypto**: AES-256 для локального шифрування ключів
- **Config**: Viper для конфігурації

**Варіант 2: Electron App (якщо потрібен красивий UI)**
- **Frontend**: React + TypeScript
- **Backend**: Node.js або Go binary
- **Складність**: Вища, більший розмір

### Архітектура Клієнта

```
premium-client/
├── cmd/
│   └── client/
│       └── main.go                 # Entry point
├── internal/
│   ├── auth/
│   │   ├── telegram_auth.go        # Telegram авторизація
│   │   ├── jwt_manager.go          # JWT токени для WebSocket
│   │   └── token_store.go          # Локальне збереження токенів
│   ├── config/
│   │   ├── config.go               # Конфігурація клієнта
│   │   └── exchange_keys.go        # Управління API ключами (encrypted)
│   ├── websocket/
│   │   ├── client.go               # WebSocket клієнт до main server
│   │   ├── handler.go              # Обробка вхідних повідомлень
│   │   ├── reconnect.go            # Auto-reconnect логіка
│   │   └── heartbeat.go            # Heartbeat/ping-pong
│   ├── exchange/
│   │   ├── interface.go            # Інтерфейс для бірж
│   │   ├── binance_client.go       # Binance API клієнт
│   │   ├── bybit_client.go         # Bybit API клієнт
│   │   ├── okx_client.go           # OKX API клієнт
│   │   └── validator.go            # Валідація API ключів
│   ├── trading/
│   │   ├── bot.go                  # Основний trading bot
│   │   ├── executor.go             # Виконання трейдів
│   │   ├── risk_manager.go         # Risk management
│   │   ├── position_manager.go     # Управління позиціями
│   │   └── strategy.go             # Стратегія торгівлі
│   ├── storage/
│   │   ├── secure_store.go         # Encrypted key storage (OS keyring)
│   │   ├── trade_history.go        # Локальна БД трейдів (SQLite)
│   │   └── stats.go                # Локальна статистика
│   ├── notification/
│   │   ├── notifier.go             # Системні повідомлення
│   │   └── telegram_bot.go         # Telegram бот для користувача
│   └── ui/
│       ├── cli.go                  # CLI інтерфейс
│       ├── tray.go                 # System tray (опціонально)
│       └── web_dashboard.go        # Local web dashboard (опціонально)
├── configs/
│   └── client_config.yaml          # Default config
├── go.mod
└── README.md
```

### Основні Компоненти

#### 1. **Authentication Module** (`internal/auth/`)

```go
// telegram_auth.go
type TelegramAuth struct {
    serverURL string
    botToken  string
}

func (ta *TelegramAuth) LoginWithTelegram(telegramID int64, authHash string) (*AuthResponse, error)

// jwt_manager.go
type JWTManager struct {
    tokenStore TokenStore
}

func (jm *JWTManager) SaveToken(token string) error
func (jm *JWTManager) GetToken() (string, error)
func (jm *JWTManager) IsTokenValid() bool
func (jm *JWTManager) RefreshToken() error

// token_store.go
type TokenStore interface {
    Save(key string, value string) error
    Get(key string) (string, error)
    Delete(key string) error
}
```

**Процес авторизації:**
1. Користувач вводить Telegram ID
2. Клієнт відправляє запит на `/api/v1/client/auth/telegram-init`
3. Сервер генерує одноразовий код
4. Користувач підтверджує через Telegram бота
5. Клієнт отримує JWT токен
6. Токен зберігається в OS keyring (secure)

#### 2. **Exchange API Keys Management** (`internal/config/`)

```go
// exchange_keys.go
type ExchangeCredentials struct {
    Exchange  string    // "binance", "bybit", "okx"
    APIKey    string    // Encrypted
    SecretKey string    // Encrypted
    Passphrase string   // Для OKX (encrypted)
    CreatedAt time.Time
    IsValid   bool
}

type KeyManager struct {
    encryptionKey []byte  // Derived from user password or OS keyring
}

func (km *KeyManager) AddExchange(exchange, apiKey, secret, passphrase string) error
func (km *KeyManager) ValidateKeys(exchange string) (bool, error)
func (km *KeyManager) GetCredentials(exchange string) (*ExchangeCredentials, error)
func (km *KeyManager) RemoveExchange(exchange string) error
func (km *KeyManager) ListExchanges() []string

// Зберігання: encrypted JSON file або OS keyring
// ~/.crypto-client/keys.enc (AES-256-GCM)
```

**Схема валідації ключів:**
```go
// validator.go
type KeyValidator struct {
    binanceClient *binance.Client
    bybitClient   *bybit.Client
    okxClient     *okx.Client
}

func (kv *KeyValidator) Validate(exchange, apiKey, secret string) (*ValidationResult, error) {
    // 1. Test API connection
    // 2. Check permissions (trading, withdraw read-only)
    // 3. Get account balance (confirm access)
    // 4. Return result with details
}

type ValidationResult struct {
    IsValid     bool
    Permissions []string // ["SPOT_TRADE", "FUTURES_TRADE"]
    Balance     map[string]float64
    Error       string
}
```

#### 3. **WebSocket Client** (`internal/websocket/`)

```go
// client.go
type Client struct {
    serverURL   string
    jwtToken    string
    conn        *websocket.Conn
    handlers    map[string]MessageHandler
    isConnected bool
    reconnect   *ReconnectManager
}

func NewClient(serverURL, jwtToken string) *Client
func (c *Client) Connect() error
func (c *Client) Disconnect() error
func (c *Client) OnMessage(messageType string, handler MessageHandler)
func (c *Client) Send(messageType string, data interface{}) error

type MessageHandler func(data []byte) error

// handler.go
type Handler struct {
    tradingBot *trading.Bot
}

func (h *Handler) HandleArbitrageOpportunity(data []byte) error {
    var opp ArbitrageOpportunity
    json.Unmarshal(data, &opp)

    // Validate opportunity
    if !h.shouldTrade(opp) {
        return nil
    }

    // Execute trade
    go h.tradingBot.ExecuteArbitrage(opp)
}

func (h *Handler) HandleTradeStatus(data []byte) error
func (h *Handler) HandleServerCommand(data []byte) error
```

**WebSocket Message Types:**
```json
// Server -> Client
{
  "type": "arbitrage_opportunity",
  "data": {
    "id": 123,
    "pair": "BTC/USDT",
    "exchange_buy": "binance",
    "exchange_sell": "bybit",
    "price_buy": 67450.30,
    "price_sell": 67850.50,
    "net_profit_percent": 0.45,
    "recommended_amount": 1000,
    "expires_at": "2024-11-17T12:35:00Z"
  },
  "timestamp": "2024-11-17T12:30:00Z"
}

// Client -> Server
{
  "type": "trade_executed",
  "data": {
    "opportunity_id": 123,
    "status": "success",
    "buy_order": {"id": "...", "price": 67450.30, "amount": 0.0148},
    "sell_order": {"id": "...", "price": 67850.50, "amount": 0.0148},
    "actual_profit": 5.92,
    "actual_profit_percent": 0.42,
    "execution_time_ms": 1250
  }
}

// Server -> Client
{
  "type": "command",
  "data": {
    "action": "pause_trading",  // або "resume_trading", "update_config"
    "reason": "High volatility detected"
  }
}
```

#### 4. **Trading Bot** (`internal/trading/`)

```go
// bot.go
type Bot struct {
    config        *Config
    exchanges     map[string]exchange.Client
    riskManager   *RiskManager
    positionMgr   *PositionManager
    executor      *Executor
    storage       *storage.TradeHistory
    isActive      bool
    mu            sync.RWMutex
}

func NewBot(config *Config, exchanges map[string]exchange.Client) *Bot
func (b *Bot) Start() error
func (b *Bot) Stop() error
func (b *Bot) ExecuteArbitrage(opp *ArbitrageOpportunity) (*TradeResult, error)
func (b *Bot) GetStats() *Stats

// executor.go
type Executor struct {
    exchanges map[string]exchange.Client
}

func (e *Executor) ExecuteArbitrage(opp *ArbitrageOpportunity, amount float64) (*TradeResult, error) {
    // 1. Check balances
    // 2. Place buy order
    // 3. Wait for fill
    // 4. Place sell order
    // 5. Wait for fill
    // 6. Calculate actual profit
    // 7. Return result
}

// risk_manager.go
type RiskManager struct {
    maxPositionSize   float64
    maxDailyLoss      float64
    maxDrawdown       float64
    currentDailyLoss  float64
    openPositions     int
}

func (rm *RiskManager) CanTrade(opp *ArbitrageOpportunity, amount float64) (bool, string)
func (rm *RiskManager) UpdateLoss(loss float64)
func (rm *RiskManager) ResetDaily()

// position_manager.go
type PositionManager struct {
    positions map[string]*Position
}

type Position struct {
    ID            string
    Pair          string
    BuyExchange   string
    SellExchange  string
    Amount        float64
    BuyPrice      float64
    SellPrice     float64
    Status        string // "pending", "partial", "completed", "failed"
    CreatedAt     time.Time
}

func (pm *PositionManager) AddPosition(p *Position)
func (pm *PositionManager) GetPosition(id string) *Position
func (pm *PositionManager) UpdateStatus(id, status string)
func (pm *PositionManager) GetOpenPositions() []*Position
```

**Trading Flow:**
```
1. Receive arbitrage opportunity from server
2. Risk Manager validates:
   - Is within max position size?
   - Is within daily loss limit?
   - Is profit worth the risk?
3. Check balances on both exchanges
4. Calculate optimal trade amount
5. Execute buy order (market or limit)
6. Wait for fill (timeout 30s)
7. Execute sell order
8. Wait for fill (timeout 30s)
9. Send result to server
10. Update local statistics
```

#### 5. **Storage Layer** (`internal/storage/`)

```go
// secure_store.go
type SecureStore struct {
    keyring keyring.Keyring  // github.com/99designs/keyring
}

func (ss *SecureStore) SaveAPIKey(exchange, key, secret string) error
func (ss *SecureStore) GetAPIKey(exchange string) (key, secret string, err error)

// trade_history.go (SQLite)
type TradeHistory struct {
    db *gorm.DB
}

type Trade struct {
    ID                  uint
    OpportunityID       uint
    Pair                string
    BuyExchange         string
    SellExchange        string
    Amount              float64
    BuyPrice            float64
    SellPrice           float64
    ActualProfit        float64
    ActualProfitPercent float64
    Status              string
    Error               string
    ExecutionTimeMs     int
    CreatedAt           time.Time
}

func (th *TradeHistory) Save(trade *Trade) error
func (th *TradeHistory) GetTrades(filters *TradeFilters) ([]*Trade, error)
func (th *TradeHistory) GetStats(period time.Duration) (*Stats, error)
```

#### 6. **Configuration**

```yaml
# client_config.yaml
client:
  name: "Crypto Trading Client"
  version: "1.0.0"
  user_id: 0                    # Set during auth

server:
  url: "wss://api.yourserver.com/v1/client/ws"
  api_url: "https://api.yourserver.com"
  timeout: 30s
  reconnect_delay: 5s
  max_reconnect_attempts: 10

trading:
  enabled: true
  auto_execute: true            # Auto-execute on opportunities
  max_position_size: 1000       # USD
  max_daily_trades: 20
  max_daily_loss: 100           # USD
  min_profit_percent: 0.3       # Only trade if >= 0.3%

risk:
  max_slippage: 0.5             # %
  order_timeout: 30             # seconds
  use_limit_orders: false       # Market orders faster

exchanges:
  binance:
    enabled: true
    trading_fee: 0.1            # 0.1%
  bybit:
    enabled: true
    trading_fee: 0.1
  okx:
    enabled: true
    trading_fee: 0.1

notifications:
  telegram_enabled: true
  telegram_chat_id: 0
  notify_on_trade: true
  notify_on_error: true
  notify_daily_summary: true

storage:
  data_dir: "~/.crypto-client"
  trade_history_days: 90        # Keep 90 days
```

### CLI Commands

```bash
# Initialization
./premium-client init
./premium-client login --telegram-id 123456789

# Exchange Management
./premium-client exchange add --name binance --api-key XXX --secret YYY
./premium-client exchange validate binance
./premium-client exchange list

# Trading
./premium-client start                    # Start trading bot
./premium-client stop                     # Stop trading bot
./premium-client status                   # Show current status

# Statistics
./premium-client stats                    # Overall statistics
./premium-client stats --today            # Today's trades
./premium-client stats --week             # This week
./premium-client history --limit 20       # Last 20 trades

# Configuration
./premium-client config set max_position_size 2000
./premium-client config show
```

---

## 🖥️ ЧАСТИНА 2: Server-Side Changes

### Нові Моделі (`internal/models/`)

```go
// client_session.go
type ClientSession struct {
    BaseModel

    UserID          uint      `gorm:"index;not null"`
    User            *User     `gorm:"foreignKey:UserID"`

    SessionID       string    `gorm:"uniqueIndex;not null"` // UUID
    ConnectionID    string    `gorm:"index"`                // WebSocket connection ID

    ClientVersion   string    // "1.0.0"
    Platform        string    // "windows", "linux", "macos"
    IPAddress       string

    IsActive        bool      `gorm:"default:true"`
    LastHeartbeat   time.Time
    ConnectedAt     time.Time
    DisconnectedAt  *time.Time
}

// client_trade.go
type ClientTrade struct {
    BaseModel

    UserID              uint      `gorm:"index;not null"`
    User                *User     `gorm:"foreignKey:UserID"`

    OpportunityID       uint      `gorm:"index;not null"`
    Opportunity         *ArbitrageOpportunity `gorm:"foreignKey:OpportunityID"`

    Pair                string    `gorm:"index;not null"`
    BuyExchange         string    `gorm:"index;not null"`
    SellExchange        string    `gorm:"index;not null"`

    Amount              float64   `gorm:"type:decimal(20,8)"`
    BuyPrice            float64   `gorm:"type:decimal(20,8)"`
    SellPrice           float64   `gorm:"type:decimal(20,8)"`

    BuyOrderID          string
    SellOrderID         string

    ExpectedProfit      float64   `gorm:"type:decimal(12,2)"`
    ActualProfit        float64   `gorm:"type:decimal(12,2)"`
    ActualProfitPercent float64   `gorm:"type:decimal(5,2)"`

    Status              string    `gorm:"index"` // "pending", "executing", "completed", "failed"
    Error               string

    ExecutionTimeMs     int       // Time to complete trade
    CreatedAt           time.Time
    CompletedAt         *time.Time
}

// client_statistics.go
type ClientStatistics struct {
    BaseModel

    UserID              uint      `gorm:"uniqueIndex;not null"`
    User                *User     `gorm:"foreignKey:UserID"`

    TotalTrades         int       `gorm:"default:0"`
    SuccessfulTrades    int       `gorm:"default:0"`
    FailedTrades        int       `gorm:"default:0"`

    TotalProfit         float64   `gorm:"type:decimal(12,2);default:0"`
    TotalLoss           float64   `gorm:"type:decimal(12,2);default:0"`
    NetProfit           float64   `gorm:"type:decimal(12,2);default:0"`

    BestTrade           float64   `gorm:"type:decimal(12,2);default:0"`
    WorstTrade          float64   `gorm:"type:decimal(12,2);default:0"`
    AvgProfit           float64   `gorm:"type:decimal(12,2);default:0"`

    WinRate             float64   `gorm:"type:decimal(5,2);default:0"` // Percentage

    TotalVolume         float64   `gorm:"type:decimal(15,2);default:0"` // Total USD traded

    LastTradeAt         *time.Time
    LastUpdateAt        time.Time
}
```

### Нові Repositories

```go
// internal/repository/client_session_repository.go
type ClientSessionRepository interface {
    Create(session *models.ClientSession) error
    GetBySessionID(sessionID string) (*models.ClientSession, error)
    GetActiveByUserID(userID uint) (*models.ClientSession, error)
    UpdateHeartbeat(sessionID string) error
    Disconnect(sessionID string) error
    ListActive() ([]*models.ClientSession, error)
    CountActive() (int64, error)
}

// internal/repository/client_trade_repository.go
type ClientTradeRepository interface {
    Create(trade *models.ClientTrade) error
    Update(trade *models.ClientTrade) error
    GetByID(id uint) (*models.ClientTrade, error)
    GetByUserID(userID uint, limit int) ([]*models.ClientTrade, error)
    GetByOpportunityID(oppID uint) ([]*models.ClientTrade, error)
    GetStats(userID uint, period time.Duration) (*TradeStats, error)
}

// internal/repository/client_statistics_repository.go
type ClientStatisticsRepository interface {
    GetByUserID(userID uint) (*models.ClientStatistics, error)
    UpdateFromTrade(trade *models.ClientTrade) error
    GetLeaderboard(limit int) ([]*models.ClientStatistics, error)
}
```

### WebSocket для Premium Clients

```go
// internal/api/websocket/client_hub.go
type ClientHub struct {
    clients    map[string]*PremiumClient  // sessionID -> client
    register   chan *PremiumClient
    unregister chan *PremiumClient
    broadcast  chan *ClientMessage
    mu         sync.RWMutex
}

type PremiumClient struct {
    SessionID    string
    UserID       uint
    User         *models.User
    Conn         *websocket.Conn
    Send         chan *ClientMessage
    Hub          *ClientHub
    LastHeartbeat time.Time
}

type ClientMessage struct {
    Type      string                 `json:"type"`
    Data      interface{}            `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (ch *ClientHub) BroadcastArbitrage(opp *models.ArbitrageOpportunity)
func (ch *ClientHub) SendToUser(userID uint, msg *ClientMessage)
func (ch *ClientHub) SendCommand(userID uint, command string, data interface{})
```

### API Endpoints

```go
// internal/api/handlers/client_handler.go
type ClientHandler struct {
    userRepo      repository.UserRepository
    sessionRepo   repository.ClientSessionRepository
    tradeRepo     repository.ClientTradeRepository
    statsRepo     repository.ClientStatisticsRepository
    jwtManager    *auth.JWTManager
    clientHub     *websocket.ClientHub
}

// POST /api/v1/client/auth/telegram-init
func (h *ClientHandler) InitTelegramAuth(w http.ResponseWriter, r *http.Request)

// POST /api/v1/client/auth/telegram-verify
func (h *ClientHandler) VerifyTelegramAuth(w http.ResponseWriter, r *http.Request)

// POST /api/v1/client/auth/refresh
func (h *ClientHandler) RefreshToken(w http.ResponseWriter, r *http.Request)

// WebSocket endpoint
// WS /api/v1/client/ws
func (h *ClientHandler) WebSocketConnection(w http.ResponseWriter, r *http.Request)

// POST /api/v1/client/trades
func (h *ClientHandler) CreateTrade(w http.ResponseWriter, r *http.Request)

// PATCH /api/v1/client/trades/:id
func (h *ClientHandler) UpdateTrade(w http.ResponseWriter, r *http.Request)

// GET /api/v1/client/trades
func (h *ClientHandler) GetTrades(w http.ResponseWriter, r *http.Request)

// GET /api/v1/client/statistics
func (h *ClientHandler) GetStatistics(w http.ResponseWriter, r *http.Request)

// POST /api/v1/client/heartbeat
func (h *ClientHandler) Heartbeat(w http.ResponseWriter, r *http.Request)
```

### Інтеграція з Arbitrage Detector

```go
// В cmd/api/main.go або cmd/bot/main.go

// Підключити ClientHub до Arbitrage Detector
arbitrageDetector.OnOpportunity(func(opp *models.ArbitrageOpportunity) {
    // Send to Telegram users (existing)
    notificationService.CreateArbitrageNotifications(opp)

    // Send to WebSocket clients (NEW)
    clientHub.BroadcastArbitrage(opp)
})
```

### Routes

```go
// internal/api/server.go

// Premium client endpoints
clientGroup := router.Group("/api/v1/client")
{
    // Public (no auth)
    clientGroup.POST("/auth/telegram-init", clientHandler.InitTelegramAuth)
    clientGroup.POST("/auth/telegram-verify", clientHandler.VerifyTelegramAuth)

    // Protected (JWT required + Premium check)
    protected := clientGroup.Group("")
    protected.Use(middleware.JWTAuth(jwtManager))
    protected.Use(middleware.RequirePremium())
    {
        protected.POST("/auth/refresh", clientHandler.RefreshToken)
        protected.GET("/ws", clientHandler.WebSocketConnection)
        protected.POST("/trades", clientHandler.CreateTrade)
        protected.PATCH("/trades/:id", clientHandler.UpdateTrade)
        protected.GET("/trades", clientHandler.GetTrades)
        protected.GET("/statistics", clientHandler.GetStatistics)
        protected.POST("/heartbeat", clientHandler.Heartbeat)
    }
}
```

### Middleware для Premium Перевірки

```go
// internal/api/middleware/premium.go
func RequirePremium() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := GetUserFromContext(c.Request.Context())
        if claims == nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
            c.Abort()
            return
        }

        // Get user from DB
        user, err := userRepo.GetByID(claims.UserID)
        if err != nil || !user.IsPremium() {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Premium subscription required",
                "upgrade_url": "https://t.me/yourbot?start=upgrade"
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### Database Migrations

```go
// internal/repository/migrations.go
func MigrateClientTables(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.ClientSession{},
        &models.ClientTrade{},
        &models.ClientStatistics{},
    )
}
```

---

## 📊 ЧАСТИНА 3: Telegram Bot Changes

### Нові команди

```go
// internal/bot/premium_handlers.go

// /client - Інформація про клієнт
func (b *Bot) handleClientCommand(message *tgbotapi.Message) {
    user := b.getUser(message.From.ID)

    if !user.IsPremium() {
        // Show upgrade message
        return
    }

    // Show client download links
    msg := "🖥 **Premium Trading Client**\n\n"
    msg += "Завантажте клієнт для вашої платформи:\n\n"
    msg += "🪟 Windows: [Download](https://...)\n"
    msg += "🐧 Linux: [Download](https://...)\n"
    msg += "🍎 MacOS: [Download](https://...)\n\n"
    msg += "📖 [Інструкція з встановлення](https://...)\n"
    msg += "🔐 [Як налаштувати API ключі](https://...)"

    b.sendMessage(message.Chat.ID, msg)
}

// /clientstats - Статистика торгівлі
func (b *Bot) handleClientStatsCommand(message *tgbotapi.Message) {
    user := b.getUser(message.From.ID)
    stats := clientStatsRepo.GetByUserID(user.ID)

    msg := formatClientStats(stats)
    b.sendMessage(message.Chat.ID, msg)
}
```

---

## 🔐 Безпека

### API Keys Storage (Client)

1. **OS Keyring Integration**
   - Windows: Windows Credential Manager
   - macOS: Keychain
   - Linux: Secret Service API (GNOME Keyring, KWallet)

2. **Fallback: Encrypted File**
   - AES-256-GCM encryption
   - Key derived from user password + device ID
   - Salt stored separately

3. **Never Send to Server**
   - API keys NEVER transmitted to server
   - All trading done directly from client to exchanges

### WebSocket Security

1. **JWT Authentication**
   - Token includes: UserID, SessionID, Expiry
   - Tokens refresh automatically
   - Server validates premium status on each connection

2. **TLS/WSS**
   - All WebSocket connections over WSS (secure)
   - Certificate pinning (optional)

3. **Rate Limiting**
   - Max connections per user: 1
   - Heartbeat required every 30s
   - Auto-disconnect on timeout

### Trade Verification

1. **Server-side**
   - Verify opportunity still valid
   - Check user premium status
   - Log all trades

2. **Client-side**
   - Validate opportunity before execution
   - Risk management checks
   - Position size limits

---

## 📈 Моніторинг та Аналітика

### Server Metrics

- Active premium clients (connected)
- Trades per minute
- Success rate
- Average profit per trade
- Most profitable users (leaderboard)

### Client Metrics (Local)

- Total trades today
- Win rate
- Total profit/loss
- Best/worst trade
- Average execution time

---

## 🚀 План Розгортання

### Phase 1: Core Infrastructure (Тиждень 1-2)

**Server:**
- [ ] Додати моделі (ClientSession, ClientTrade, ClientStatistics)
- [ ] Створити repositories
- [ ] Додати ClientHub для WebSocket
- [ ] API endpoints для auth та trades
- [ ] Middleware для premium перевірки
- [ ] Інтеграція з Arbitrage Detector

**Client:**
- [ ] Project structure
- [ ] Authentication module
- [ ] WebSocket client
- [ ] Config management

### Phase 2: Trading Logic (Тиждень 3-4)

**Client:**
- [ ] Exchange API clients (Binance, Bybit, OKX)
- [ ] Trading bot core
- [ ] Risk manager
- [ ] Position manager
- [ ] Trade executor

### Phase 3: Storage & UI (Тиждень 5)

**Client:**
- [ ] Secure key storage
- [ ] SQLite trade history
- [ ] CLI interface
- [ ] Basic web dashboard (опціонально)

### Phase 4: Testing & Polish (Тиждень 6)

- [ ] Integration testing
- [ ] Security audit
- [ ] Performance optimization
- [ ] Documentation
- [ ] Beta release

### Phase 5: Production (Тиждень 7)

- [ ] Production deployment
- [ ] Monitoring setup
- [ ] User onboarding
- [ ] Support documentation

---

## 💡 Додаткові Фічі (Future)

1. **Backtesting**
   - Client може завантажити історичні дані
   - Тестувати стратегії

2. **Custom Strategies**
   - Користувач може налаштувати власні правила
   - Min profit, max slippage, etc.

3. **Portfolio Management**
   - Відстежування всього портфоліо
   - Реалізація PnL

4. **Advanced Risk Management**
   - Stop-loss
   - Take-profit
   - Max drawdown protection

5. **Multi-User Support**
   - Один клієнт для кількох користувачів
   - Corporate accounts

6. **Mobile App**
   - iOS/Android для моніторингу
   - Push notifications

---

## 📝 Технічні Вимоги

### Server Requirements

- Go 1.25+
- PostgreSQL 14+
- Redis (для rate limiting)
- SSL certificate (для WSS)
- Min 2GB RAM
- Min 2 CPU cores

### Client Requirements

- Go 1.25+
- OS: Windows 10+, Linux (Ubuntu 20.04+), macOS 11+
- Internet: Stable connection (низька латентність важлива)
- Storage: 100MB для app + space для trade history
- RAM: 256MB minimum

---

## 🔗 Dependencies

### Server (додати в go.mod)

```go
github.com/gorilla/websocket  // WebSocket
github.com/golang-jwt/jwt/v5  // JWT already there
```

### Client (новий проект)

```go
github.com/gorilla/websocket          // WebSocket
github.com/spf13/viper                // Config
github.com/99designs/keyring          // Secure storage
gorm.io/driver/sqlite                 // Local DB
github.com/adshao/go-binance/v2      // Binance
github.com/hirokisan/bybit/v2        // Bybit
github.com/amir-the-h/okx            // OKX
github.com/urfave/cli/v2             // CLI
```

---

## 📞 Support & Documentation

1. **Client Documentation**
   - Installation guide
   - API keys setup
   - Trading configuration
   - Troubleshooting

2. **Video Tutorials**
   - Як встановити клієнт
   - Налаштування API ключів
   - Перший трейд

3. **FAQ**
   - Security questions
   - Trading strategies
   - Fees and profits

4. **Support Channels**
   - Telegram support group
   - Email support
   - GitHub issues

---

## ✅ Переваги Цього Рішення

1. **Безпека**
   - API ключі залишаються на пристрої користувача
   - Повний контроль над коштами
   - Ніхто не може торгувати без дозволу

2. **Швидкість**
   - Прямі з'єднання до бірж
   - Низька латентність
   - Швидке виконання

3. **Гнучкість**
   - Користувач налаштовує ризики
   - Може вимкнути авто-торгівлю
   - Повний контроль

4. **Масштабованість**
   - Кожен клієнт незалежний
   - Не навантажує сервер
   - Необмежена кількість користувачів

5. **Transparency**
   - Користувач бачить всі трейди
   - Локальна статистика
   - Повна прозорість

---

## ⚠️ Ризики та Обмеження

1. **Технічні**
   - Потрібна стабільна інтернет
   - Клієнт має бути постійно запущений
   - Можливі збої на біржах

2. **Фінансові**
   - Ризик втрати через волатильність
   - Комісії бірж
   - Slippage

3. **Юридичні**
   - Regulatory compliance (KYC/AML)
   - Tax reporting
   - Terms of Service бірж

---

Це повний план реалізації Premium Trading Client. Якщо потрібні деталі з якоїсь частини - дай знати!
