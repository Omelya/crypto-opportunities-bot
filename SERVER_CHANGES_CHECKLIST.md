# Server Changes Checklist для Premium Client

## 📋 Огляд Змін

Цей документ містить список усіх змін, які потрібно внести на сервері для підтримки Premium Trading Client.

---

## 1️⃣ Database Models

### Нові моделі в `internal/models/`

- [ ] **client_session.go** - Сесії підключених клієнтів
  ```go
  type ClientSession struct {
      BaseModel
      UserID          uint
      SessionID       string    // UUID
      ConnectionID    string
      ClientVersion   string
      Platform        string
      IPAddress       string
      IsActive        bool
      LastHeartbeat   time.Time
      ConnectedAt     time.Time
      DisconnectedAt  *time.Time
  }
  ```

- [ ] **client_trade.go** - Трейди виконані клієнтами
  ```go
  type ClientTrade struct {
      BaseModel
      UserID              uint
      OpportunityID       uint
      Pair                string
      BuyExchange         string
      SellExchange        string
      Amount              float64
      BuyPrice            float64
      SellPrice           float64
      BuyOrderID          string
      SellOrderID         string
      ExpectedProfit      float64
      ActualProfit        float64
      ActualProfitPercent float64
      Status              string // pending, executing, completed, failed
      Error               string
      ExecutionTimeMs     int
      CompletedAt         *time.Time
  }
  ```

- [ ] **client_statistics.go** - Статистика користувачів
  ```go
  type ClientStatistics struct {
      BaseModel
      UserID              uint
      TotalTrades         int
      SuccessfulTrades    int
      FailedTrades        int
      TotalProfit         float64
      TotalLoss           float64
      NetProfit           float64
      BestTrade           float64
      WorstTrade          float64
      AvgProfit           float64
      WinRate             float64
      TotalVolume         float64
      LastTradeAt         *time.Time
      LastUpdateAt        time.Time
  }
  ```

---

## 2️⃣ Repositories

### Нові repository в `internal/repository/`

- [ ] **client_session_repository.go**
  ```go
  type ClientSessionRepository interface {
      Create(session *models.ClientSession) error
      GetBySessionID(sessionID string) (*models.ClientSession, error)
      GetActiveByUserID(userID uint) (*models.ClientSession, error)
      UpdateHeartbeat(sessionID string) error
      Disconnect(sessionID string) error
      ListActive() ([]*models.ClientSession, error)
      CountActive() (int64, error)
  }
  ```

- [ ] **client_trade_repository.go**
  ```go
  type ClientTradeRepository interface {
      Create(trade *models.ClientTrade) error
      Update(trade *models.ClientTrade) error
      GetByID(id uint) (*models.ClientTrade, error)
      GetByUserID(userID uint, limit int) ([]*models.ClientTrade, error)
      GetByOpportunityID(oppID uint) ([]*models.ClientTrade, error)
      GetStats(userID uint, period time.Duration) (*TradeStats, error)
  }
  ```

- [ ] **client_statistics_repository.go**
  ```go
  type ClientStatisticsRepository interface {
      GetByUserID(userID uint) (*models.ClientStatistics, error)
      UpdateFromTrade(trade *models.ClientTrade) error
      GetLeaderboard(limit int) ([]*models.ClientStatistics, error)
      RecalculateStats(userID uint) error
  }
  ```

---

## 3️⃣ WebSocket Infrastructure

### Новий пакет `internal/api/websocket/`

- [ ] **client_hub.go** - Hub для premium клієнтів
  ```go
  type ClientHub struct {
      clients    map[string]*PremiumClient
      register   chan *PremiumClient
      unregister chan *PremiumClient
      broadcast  chan *ClientMessage
      mu         sync.RWMutex
  }

  func NewClientHub() *ClientHub
  func (ch *ClientHub) Run()
  func (ch *ClientHub) BroadcastArbitrage(opp *models.ArbitrageOpportunity)
  func (ch *ClientHub) SendToUser(userID uint, msg *ClientMessage)
  func (ch *ClientHub) SendCommand(userID uint, command string, data interface{})
  func (ch *ClientHub) GetConnectedClients() int
  ```

- [ ] **premium_client.go** - Окремий клієнт для кожного підключення
  ```go
  type PremiumClient struct {
      SessionID     string
      UserID        uint
      User          *models.User
      Conn          *websocket.Conn
      Send          chan *ClientMessage
      Hub           *ClientHub
      LastHeartbeat time.Time
      mu            sync.Mutex
  }

  func (c *PremiumClient) ReadPump()
  func (c *PremiumClient) WritePump()
  func (c *PremiumClient) HandleMessage(msg *ClientMessage)
  ```

- [ ] **client_message.go** - Message types
  ```go
  type ClientMessage struct {
      Type      string                 `json:"type"`
      Data      interface{}            `json:"data"`
      Timestamp time.Time              `json:"timestamp"`
      Metadata  map[string]interface{} `json:"metadata,omitempty"`
  }

  // Message types:
  // - arbitrage_opportunity
  // - trade_executed
  // - trade_failed
  // - command (pause/resume/update_config)
  // - heartbeat
  // - stats_update
  ```

---

## 4️⃣ API Handlers

### Новий handler `internal/api/handlers/client_handler.go`

- [ ] **ClientHandler struct**
  ```go
  type ClientHandler struct {
      userRepo      repository.UserRepository
      sessionRepo   repository.ClientSessionRepository
      tradeRepo     repository.ClientTradeRepository
      statsRepo     repository.ClientStatisticsRepository
      jwtManager    *auth.JWTManager
      clientHub     *websocket.ClientHub
  }
  ```

- [ ] **Authentication Endpoints**
  - [ ] `POST /api/v1/client/auth/telegram-init` - Ініціалізація авторизації
  - [ ] `POST /api/v1/client/auth/telegram-verify` - Верифікація через Telegram
  - [ ] `POST /api/v1/client/auth/refresh` - Оновлення токена

- [ ] **WebSocket Endpoint**
  - [ ] `WS /api/v1/client/ws` - WebSocket підключення
    - Приймає JWT токен
    - Валідує premium статус
    - Створює ClientSession
    - Підключає до ClientHub

- [ ] **Trade Endpoints**
  - [ ] `POST /api/v1/client/trades` - Створення трейду (коли клієнт починає)
  - [ ] `PATCH /api/v1/client/trades/:id` - Оновлення статусу трейду
  - [ ] `GET /api/v1/client/trades` - Список трейдів користувача

- [ ] **Statistics Endpoints**
  - [ ] `GET /api/v1/client/statistics` - Статистика користувача
  - [ ] `GET /api/v1/client/statistics/leaderboard` - Топ трейдерів

- [ ] **Session Management**
  - [ ] `POST /api/v1/client/heartbeat` - Heartbeat для підтримки з'єднання
  - [ ] `POST /api/v1/client/disconnect` - Graceful disconnect

---

## 5️⃣ Middleware

### Новий middleware `internal/api/middleware/premium.go`

- [ ] **RequirePremium() middleware**
  ```go
  func RequirePremium() gin.HandlerFunc {
      return func(c *gin.Context) {
          claims := GetUserFromContext(c.Request.Context())
          user, _ := userRepo.GetByID(claims.UserID)

          if !user.IsPremium() {
              c.JSON(403, gin.H{
                  "error": "Premium subscription required",
                  "upgrade_url": "..."
              })
              c.Abort()
              return
          }

          c.Next()
      }
  }
  ```

---

## 6️⃣ Routes

### Оновити `internal/api/server.go`

- [ ] **Додати ClientHub ініціалізацію**
  ```go
  clientHub := websocket.NewClientHub()
  go clientHub.Run()
  ```

- [ ] **Додати Client routes**
  ```go
  clientGroup := router.Group("/api/v1/client")
  {
      // Public
      clientGroup.POST("/auth/telegram-init", clientHandler.InitTelegramAuth)
      clientGroup.POST("/auth/telegram-verify", clientHandler.VerifyTelegramAuth)

      // Protected (JWT + Premium)
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
          protected.GET("/statistics/leaderboard", clientHandler.GetLeaderboard)
          protected.POST("/heartbeat", clientHandler.Heartbeat)
      }
  }
  ```

---

## 7️⃣ Integration з Arbitrage Detector

### Оновити `cmd/api/main.go` або `cmd/bot/main.go`

- [ ] **Передати ClientHub до ArbitrageDetector**
  ```go
  // При створенні arbitrage detector
  detector := arbitrage.NewDetector(...)

  // Додати callback для відправки в WebSocket
  detector.OnOpportunity(func(opp *models.ArbitrageOpportunity) {
      // Existing: Send to Telegram users
      notificationService.CreateArbitrageNotifications(opp)

      // NEW: Send to WebSocket clients
      if clientHub != nil {
          clientHub.BroadcastArbitrage(opp)
      }
  })
  ```

---

## 8️⃣ Telegram Bot Changes

### Оновити `internal/bot/premium_handlers.go`

- [ ] **Нова команда `/client`**
  ```go
  func (b *Bot) handleClientCommand(message *tgbotapi.Message) {
      user := b.getUser(message.From.ID)

      if !user.IsPremium() {
          b.sendUpgradeMessage(message.Chat.ID)
          return
      }

      // Show client download links
      msg := "🖥 **Premium Trading Client**\n\n"
      msg += "Завантажте клієнт:\n"
      msg += "🪟 Windows: [Download](https://...)\n"
      msg += "🐧 Linux: [Download](https://...)\n"
      msg += "🍎 MacOS: [Download](https://...)\n\n"
      msg += "📖 [Інструкція](https://...)"

      b.sendMessage(message.Chat.ID, msg)
  }
  ```

- [ ] **Нова команда `/clientstats`**
  ```go
  func (b *Bot) handleClientStatsCommand(message *tgbotapi.Message) {
      user := b.getUser(message.From.ID)
      stats := b.statsRepo.GetByUserID(user.ID)

      msg := formatClientStats(stats)
      b.sendMessage(message.Chat.ID, msg)
  }
  ```

- [ ] **Додати команди в commands.go**
  ```go
  case "client":
      b.handleClientCommand(message)
  case "clientstats":
      b.handleClientStatsCommand(message)
  ```

---

## 9️⃣ Configuration

### Оновити `configs/config.yaml`

- [ ] **Додати секцію client**
  ```yaml
  client:
    enabled: true
    websocket_path: "/api/v1/client/ws"
    heartbeat_interval: 30           # seconds
    heartbeat_timeout: 90            # seconds
    max_connections_per_user: 1     # Один клієнт на користувача
    rate_limit: 100                 # requests per minute
    allowed_origins:
      - "http://localhost:*"        # Dev
      - "https://yourdomain.com"    # Prod
  ```

### Оновити `internal/config/config.go`

- [ ] **Додати ClientConfig struct**
  ```go
  type ClientConfig struct {
      Enabled              bool     `mapstructure:"enabled"`
      WebSocketPath        string   `mapstructure:"websocket_path"`
      HeartbeatInterval    int      `mapstructure:"heartbeat_interval"`
      HeartbeatTimeout     int      `mapstructure:"heartbeat_timeout"`
      MaxConnectionsPerUser int     `mapstructure:"max_connections_per_user"`
      RateLimit            int      `mapstructure:"rate_limit"`
      AllowedOrigins       []string `mapstructure:"allowed_origins"`
  }
  ```

---

## 🔟 Database Migrations

### Оновити `internal/repository/db.go`

- [ ] **Додати міграцію client tables**
  ```go
  func AutoMigrate(db *gorm.DB) error {
      return db.AutoMigrate(
          // Existing models
          &models.User{},
          &models.Opportunity{},
          ...

          // NEW: Client models
          &models.ClientSession{},
          &models.ClientTrade{},
          &models.ClientStatistics{},
      )
  }
  ```

---

## 1️⃣1️⃣ Services

### Новий service `internal/service/client_trade_service.go`

- [ ] **ClientTradeService**
  ```go
  type ClientTradeService struct {
      tradeRepo repository.ClientTradeRepository
      statsRepo repository.ClientStatisticsRepository
      oppRepo   repository.ArbitrageRepository
  }

  func (s *ClientTradeService) CreateTrade(userID uint, oppID uint) (*models.ClientTrade, error)
  func (s *ClientTradeService) UpdateTradeStatus(tradeID uint, status string, data *UpdateData) error
  func (s *ClientTradeService) CalculateStats(userID uint) (*models.ClientStatistics, error)
  ```

---

## 1️⃣2️⃣ Monitoring & Logging

### Додати метрики в `internal/api/handlers/stats_handler.go`

- [ ] **Client metrics endpoint**
  ```go
  GET /api/v1/admin/stats/clients
  {
      "connected_clients": 15,
      "active_sessions": 15,
      "trades_today": 234,
      "total_volume_24h": 45678.90,
      "success_rate": 87.5,
      "avg_profit_per_trade": 3.42
  }
  ```

### Додати логи

- [ ] Client connection/disconnection
- [ ] Trade execution start/complete
- [ ] Errors та failures
- [ ] Premium validation failures

---

## 1️⃣3️⃣ Security

### Rate Limiting

- [ ] **WebSocket connections** - 1 per user
- [ ] **API calls** - 100 per minute per user
- [ ] **Trade creation** - Max 20 trades per day (configurable)

### Validation

- [ ] Validate JWT on every WebSocket message
- [ ] Check premium status on each trade
- [ ] Validate opportunity exists and is active
- [ ] Check opportunity not expired

### IP Restrictions (optional)

- [ ] Whitelist IPs for production
- [ ] Block suspicious IPs
- [ ] Log all authentication attempts

---

## 1️⃣4️⃣ Testing

### Unit Tests

- [ ] `client_hub_test.go` - Hub operations
- [ ] `premium_client_test.go` - Client operations
- [ ] `client_handler_test.go` - API endpoints
- [ ] Repository tests

### Integration Tests

- [ ] WebSocket connection flow
- [ ] Authentication flow
- [ ] Trade creation and update
- [ ] Statistics calculation

### Load Tests

- [ ] 100 concurrent WebSocket connections
- [ ] 1000 trades per minute
- [ ] Hub broadcast performance

---

## 1️⃣5️⃣ Documentation

### API Documentation

- [ ] Додати OpenAPI/Swagger specs для client endpoints
- [ ] WebSocket message formats
- [ ] Error codes та handling

### Deployment

- [ ] Environment variables
- [ ] SSL/TLS configuration для WSS
- [ ] Nginx configuration (WebSocket proxy)
- [ ] Firewall rules

---

## 📦 Dependencies

### Додати в `go.mod`

```bash
go get github.com/gorilla/websocket  # Можливо вже є
```

---

## 🚀 Deployment Steps

### 1. Database Migration

```bash
# Backup database
make db-backup

# Run migrations (auto-migrate on startup)
# або manual:
psql -U postgres -d crypto_bot -c "
CREATE TABLE client_sessions (...);
CREATE TABLE client_trades (...);
CREATE TABLE client_statistics (...);
"
```

### 2. Server Restart

```bash
# Stop server
make stop

# Build with new code
make build

# Start server
make start
```

### 3. Verify

```bash
# Check WebSocket endpoint
wscat -c wss://api.yourserver.com/v1/client/ws -H "Authorization: Bearer YOUR_JWT"

# Check API
curl https://api.yourserver.com/v1/client/statistics \
  -H "Authorization: Bearer YOUR_JWT"
```

---

## ✅ Final Checklist

- [ ] Всі моделі створені та мігровані
- [ ] Repositories implemented
- [ ] ClientHub працює
- [ ] API endpoints тестуються
- [ ] WebSocket з'єднання працюють
- [ ] Інтеграція з ArbitrageDetector
- [ ] Premium validation працює
- [ ] Telegram bot команди додані
- [ ] Logging працює
- [ ] Metrics збираються
- [ ] Documentation оновлена
- [ ] Tests написані та passed
- [ ] Security review пройдено
- [ ] Load testing пройдено
- [ ] Production deployment успішний

---

## 📞 Support

При виникненні проблем:

1. Перевірити logs: `tail -f /var/log/crypto-bot/api.log`
2. Перевірити WebSocket connections: `netstat -an | grep :8080`
3. Перевірити database: `psql -U postgres -d crypto_bot`
4. Перевірити metrics: `GET /api/v1/admin/stats/clients`

---

**Estimated Development Time**: 2-3 тижні для повної реалізації server-side змін

**Priority**: High (critical для premium users)

**Complexity**: Medium-High
