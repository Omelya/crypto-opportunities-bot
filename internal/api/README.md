# Admin Panel API

REST API для управління Crypto Opportunities Bot.

## 🚀 Запуск

### Development Mode

```bash
# З кореневої директорії проекту
go run cmd/api/main.go
```

### Production Mode

```bash
# Build binary
go build -o bin/admin-api cmd/api/main.go

# Run
./bin/admin-api
```

## ⚙️ Конфігурація

Додайте секцію `admin` в `configs/config.yaml`:

```yaml
admin:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  jwt_secret: "${ADMIN_JWT_SECRET}"  # Встановіть через env variable
  allowed_origins:
    - "http://localhost:3000"  # Frontend URL
    - "https://admin.yourbot.com"
  rate_limit: 100  # requests per minute
```

### Environment Variables

```bash
# Required
ADMIN_JWT_SECRET=your-secret-key-here  # Мінімум 32 символи

# Optional (overrides config.yaml)
ADMIN_PORT=8080
ADMIN_HOST=0.0.0.0
```

## 📡 API Endpoints

### Health Check

```bash
# Перевірка статусу сервера
GET /api/v1/health

# Response
{
  "status": "healthy",
  "uptime": "1h23m45s",
  "version": "1.0.0",
  "go_version": "go1.25.3"
}
```

```bash
# Simple ping
GET /api/v1/ping

# Response
{"message": "pong"}
```

### User Management

```bash
# Список користувачів (з пагінацією та фільтрами)
GET /api/v1/users?page=1&limit=20&tier=premium&is_active=true

# Query Parameters:
# - page: номер сторінки (default: 1)
# - limit: кількість на сторінці (default: 20, max: 100)
# - tier: фільтр за підпискою (free, premium)
# - is_active: фільтр за статусом (true, false)

# Response
{
  "users": [
    {
      "id": 1,
      "telegram_id": 123456789,
      "username": "john_doe",
      "subscription_tier": "premium",
      "is_active": true,
      "created_at": "2024-01-15T10:00:00Z",
      ...
    }
  ],
  "total": 150,
  "page": 1,
  "limit": 20,
  "total_pages": 8
}
```

```bash
# Отримати користувача
GET /api/v1/users/:id

# Response
{
  "id": 1,
  "telegram_id": 123456789,
  "username": "john_doe",
  "subscription_tier": "premium",
  "is_active": true,
  "capital_range": "1000-5000",
  "risk_profile": "moderate",
  ...
}
```

```bash
# Оновити користувача
PUT /api/v1/users/:id

# Request body
{
  "is_blocked": false,
  "subscription_tier": "premium"
}

# Response
{
  "id": 1,
  "telegram_id": 123456789,
  "is_blocked": false,
  "subscription_tier": "premium",
  ...
}
```

```bash
# Видалити користувача (soft delete)
DELETE /api/v1/users/:id

# Response
{
  "message": "User deleted successfully"
}
```

```bash
# Статистика користувача
GET /api/v1/users/:id/stats

# Response
{
  "user_id": 1,
  "notifications_sent": 45,
  "actions_count": 23,
  "subscription_tier": "premium",
  "is_premium": true,
  "capital_range": "1000-5000",
  "risk_profile": "moderate"
}
```

```bash
# Дії користувача
GET /api/v1/users/:id/actions?page=1&limit=20

# Response
{
  "user_id": 1,
  "actions": [
    {
      "id": 1,
      "type": "opportunity_viewed",
      "opportunity_id": 15,
      "created_at": "2024-01-20T15:30:00Z"
    }
  ],
  "total": 45,
  "page": 1,
  "limit": 20
}
```

### Statistics

```bash
# Dashboard статистика
GET /api/v1/stats/dashboard

# Response
{
  "users": {
    "total": 1000,
    "active": 750,
    "premium": 150,
    "free": 850
  },
  "opportunities": {
    "active": 25,
    "arbitrage": 10,
    "defi": 15
  },
  "notifications": {
    "pending": 50,
    "sent": 10000,
    "failed": 25,
    "total": 10075
  }
}
```

```bash
# User статистика
GET /api/v1/stats/users

# Response
{
  "total": 1000,
  "active": 750,
  "premium": 150,
  "free": 850
}
```

### Opportunities Management

```bash
# Список opportunities (з пагінацією та фільтрами)
GET /api/v1/opportunities?page=1&limit=20&exchange=binance&type=launchpool&is_active=true

# Query Parameters:
# - page, limit: пагінація
# - exchange: фільтр за біржею (binance, bybit, okx, ...)
# - type: фільтр за типом (launchpool, airdrop, learn_earn, staking, arbitrage, defi)
# - is_active: активні/неактивні

# Response
{
  "opportunities": [
    {
      "id": 1,
      "external_id": "binance:launchpool:123",
      "exchange": "binance",
      "type": "launchpool",
      "title": "BNB Launchpool: XYZ Token",
      "estimated_roi": 12.5,
      "is_active": true,
      "end_date": "2024-02-01T00:00:00Z",
      ...
    }
  ],
  "total": 45,
  "page": 1,
  "limit": 20
}
```

```bash
# Отримати opportunity
GET /api/v1/opportunities/:id

# Response
{
  "id": 1,
  "external_id": "binance:launchpool:123",
  "exchange": "binance",
  "type": "launchpool",
  "title": "BNB Launchpool: XYZ Token",
  "description": "Stake BNB to earn XYZ tokens",
  "reward": "Up to 15% APR",
  "estimated_roi": 12.5,
  "pool_size": 1000000,
  "min_investment": 0.1,
  "url": "https://...",
  "is_active": true,
  ...
}
```

```bash
# Створити opportunity
POST /api/v1/opportunities

# Request body
{
  "exchange": "binance",
  "type": "launchpool",
  "title": "New Launchpool",
  "description": "Description here",
  "estimated_roi": 15.0,
  "url": "https://..."
}

# Response
{
  "id": 123,
  "external_id": "binance:launchpool:new123",
  ...
}
```

```bash
# Оновити opportunity
PUT /api/v1/opportunities/:id

# Request body
{
  "title": "Updated Title",
  "estimated_roi": 20.0
}
```

```bash
# Деактивувати opportunity
POST /api/v1/opportunities/:id/deactivate

# Response
{
  "message": "Opportunity deactivated successfully"
}
```

```bash
# Видалити opportunity (soft delete)
DELETE /api/v1/opportunities/:id

# Response
{
  "message": "Opportunity deleted successfully"
}
```

### Arbitrage Management

```bash
# Список arbitrage opportunities
GET /api/v1/arbitrage?page=1&limit=20&pair=BTC/USDT&min_profit=1.0

# Query Parameters:
# - pair: торговельна пара
# - min_profit: мінімальний прибуток (%)
# - exchange_buy, exchange_sell: фільтр за біржами

# Response
{
  "arbitrage": [
    {
      "id": 1,
      "pair": "BTC/USDT",
      "exchange_buy": "binance",
      "exchange_sell": "bybit",
      "buy_price": 42000.50,
      "sell_price": 42500.75,
      "profit_percent": 1.19,
      "is_active": true,
      "detected_at": "2024-01-20T15:30:00Z"
    }
  ],
  "total": 15,
  "page": 1,
  "limit": 20
}
```

```bash
# Отримати arbitrage opportunity
GET /api/v1/arbitrage/:id

# Response
{
  "id": 1,
  "pair": "BTC/USDT",
  "exchange_buy": "binance",
  "exchange_sell": "bybit",
  "buy_price": 42000.50,
  "sell_price": 42500.75,
  "profit_percent": 1.19,
  "spread": 500.25,
  "volume_24h": 1000000,
  "is_active": true,
  ...
}
```

```bash
# Статистика arbitrage
GET /api/v1/arbitrage/stats

# Response
{
  "active_count": 5,
  "total_count": 120,
  "average_profit_percent": 0.85,
  "max_profit_percent": 3.5,
  "top_pairs": [
    {"pair": "BTC/USDT", "count": 45},
    {"pair": "ETH/USDT", "count": 32}
  ]
}
```

```bash
# Статус бірж для arbitrage
GET /api/v1/arbitrage/exchanges

# Response
{
  "exchanges": [
    {
      "name": "binance",
      "is_active": true,
      "last_update": "2024-01-20T15:30:00Z",
      "opportunities_count": 25
    },
    {
      "name": "bybit",
      "is_active": true,
      "last_update": "2024-01-20T15:29:00Z",
      "opportunities_count": 18
    }
  ]
}
```

### DeFi Management

```bash
# Список DeFi opportunities
GET /api/v1/defi?page=1&limit=20&chain=ethereum&protocol=aave&min_apy=5.0

# Query Parameters:
# - chain: фільтр за блокчейном (ethereum, bsc, polygon, ...)
# - protocol: фільтр за протоколом (aave, compound, uniswap, ...)
# - min_apy: мінімальний APY (%)
# - risk_level: рівень ризику (low, medium, high)

# Response
{
  "defi": [
    {
      "id": 1,
      "protocol": "aave",
      "chain": "ethereum",
      "asset": "USDC",
      "apy": 8.5,
      "tvl": 1500000000,
      "risk_level": "low",
      "is_active": true,
      "updated_at": "2024-01-20T15:30:00Z"
    }
  ],
  "total": 30,
  "page": 1,
  "limit": 20
}
```

```bash
# Отримати DeFi opportunity
GET /api/v1/defi/:id

# Response
{
  "id": 1,
  "protocol": "aave",
  "chain": "ethereum",
  "asset": "USDC",
  "apy": 8.5,
  "tvl": 1500000000,
  "risk_level": "low",
  "url": "https://app.aave.com",
  "description": "Lend USDC on Aave V3",
  "is_active": true,
  ...
}
```

```bash
# Статистика DeFi
GET /api/v1/defi/stats

# Response
{
  "active_count": 25,
  "total_count": 150,
  "average_apy": 6.8,
  "max_apy": 45.2,
  "total_tvl": 5000000000,
  "by_chain": [
    {"chain": "ethereum", "count": 80, "avg_apy": 5.5},
    {"chain": "bsc", "count": 40, "avg_apy": 12.3}
  ]
}
```

```bash
# Список протоколів
GET /api/v1/defi/protocols

# Response
{
  "protocols": [
    {"name": "aave", "count": 45},
    {"name": "compound", "count": 32},
    {"name": "uniswap", "count": 28}
  ]
}
```

```bash
# Список блокчейнів
GET /api/v1/defi/chains

# Response
{
  "chains": [
    {"name": "ethereum", "count": 80},
    {"name": "bsc", "count": 40},
    {"name": "polygon", "count": 30}
  ]
}
```

```bash
# Запустити DeFi scraping вручну
POST /api/v1/defi/scrape

# Response
{
  "message": "DeFi scraping triggered successfully",
  "status": "running"
}
```

### Authentication

```bash
# Login
POST /api/v1/auth/login

# Request body
{
  "username": "admin",
  "password": "secure_password"
}

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 86400,
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "super_admin",
    "is_active": true
  }
}
```

```bash
# Logout
POST /api/v1/auth/logout
Authorization: Bearer <token>

# Response
{
  "message": "Logged out successfully"
}
```

```bash
# Отримати поточного користувача
GET /api/v1/auth/me
Authorization: Bearer <token>

# Response
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "role": "super_admin",
  "is_active": true,
  "last_login_at": "2024-01-20T10:00:00Z"
}
```

```bash
# Оновити токен
POST /api/v1/auth/refresh
Authorization: Bearer <old_token>

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 86400
}
```

## 🔧 Middleware

### Logging

Автоматично логує всі HTTP запити:

```
📡 GET /api/v1/users | Status: 200 | Duration: 15ms | Size: 1024 bytes | IP: 127.0.0.1
```

### CORS

Налаштовується через `allowed_origins` в конфігурації.

Підтримує:
- Exact match: `https://admin.example.com`
- Wildcard subdomains: `*.example.com`
- Development wildcard: `*` (не використовуйте в production!)

### Recovery

Ловить паніки та повертає 500 помилку з деталями (stack trace в логах).

### Rate Limiting

Обмежує кількість запитів з одного IP:
- Default: 100 requests/minute
- Використовує Token Bucket algorithm
- Автоматичне очищення старих записів

### Authentication (JWT)

Всі protected endpoints вимагають JWT токен в Authorization header:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Role-Based Access Control (RBAC):**
- `viewer` - може тільки читати дані
- `admin` - може читати та модифікувати users та opportunities
- `super_admin` - повний доступ до всіх операцій

**Token Expiration:** 24 години

## 📦 Структура

```
internal/api/
├── server.go                    # HTTP server
├── middleware/                  # Middleware
│   ├── logging.go               # Request logging
│   ├── recovery.go              # Panic recovery
│   ├── cors.go                  # CORS headers
│   ├── auth.go                  # JWT authentication
│   └── ratelimit.go             # Rate limiting
├── handlers/                    # Request handlers
│   ├── health_handler.go        # Health check
│   ├── user_handler.go          # User management
│   ├── stats_handler.go         # Statistics
│   ├── auth_handler.go          # Authentication (login/logout)
│   ├── opportunity_handler.go   # Opportunities management
│   ├── arbitrage_handler.go     # Arbitrage management
│   └── defi_handler_api.go      # DeFi management
├── auth/                        # Authentication
│   ├── jwt.go                   # JWT manager
│   └── token.go                 # Token helpers
└── websocket/                   # WebSocket (TODO)
    └── monitor.go               # Real-time monitoring
```

## 🧪 Testing

```bash
# Run tests
go test ./internal/api/... -v

# Test with curl
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/ping
curl http://localhost:8080/api/v1/users
```

## 📝 TODO

### Phase 1 - ✅ Completed
- [x] HTTP Server setup (gorilla/mux)
- [x] Middleware (logging, CORS, recovery)
- [x] Health check endpoints
- [x] User management endpoints (basic)
- [x] Statistics endpoints (basic)

### Phase 1.2 - ✅ Completed
- [x] JWT Authentication (HMAC-SHA256, 24h expiration)
- [x] Admin User model + repository
- [x] Role-Based Access Control (viewer/admin/super_admin)
- [x] Rate limiting middleware (Token Bucket, 100 req/min)
- [x] Auth endpoints (login, logout, me, refresh)

### Phase 2 - ✅ Completed
- [x] User management (pagination, filters, delete, actions)
- [x] Opportunities management (full CRUD)
- [x] Arbitrage management (list, stats, exchanges)
- [x] DeFi management (list, stats, protocols, chains, manual scrape)

### Phase 3 - Planned
- [ ] Notification management endpoints
- [ ] WebSocket real-time monitoring
- [ ] Broadcast system
- [ ] Payment management (Stripe integration)
- [ ] System control endpoints (restart scrapers, clear cache)

### Phase 4 - Future
- [ ] Swagger/OpenAPI documentation
- [ ] Unit + Integration tests
- [ ] Frontend dashboard (React/Vue/Retool)
- [ ] Docker deployment configuration
- [ ] Performance monitoring and metrics

## 🔐 Security

1. **HTTPS Required** in production
2. **JWT Tokens** для authentication
3. **CORS** - whitelist тільки дозволені origins
4. **Rate Limiting** - захист від brute force
5. **Input Validation** - всі входи валідуються
6. **Audit Logs** - логування всіх admin операцій

## 📚 Resources

- [Gorilla Mux Documentation](https://github.com/gorilla/mux)
- [GORM Documentation](https://gorm.io/docs/)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)

## 🐛 Troubleshooting

### Port already in use

```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>
```

### CORS errors

Переконайтеся що frontend URL додано в `allowed_origins` в config.yaml.

### Database connection fails

Перевірте що PostgreSQL запущений та credentials правильні в config.yaml.
