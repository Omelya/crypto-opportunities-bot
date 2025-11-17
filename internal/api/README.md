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
# Список користувачів
GET /api/v1/users
# TODO: Add pagination (?page=1&limit=20)
# TODO: Add filters (?tier=premium&is_active=true)

# Response
{
  "users": [...],
  "total": 150
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
  ...
}
```

```bash
# Статистика користувача
GET /api/v1/users/:id/stats

# Response
{
  "user_id": 1,
  "notifications_sent": 45,
  "opportunities_viewed": 120,
  "subscription_tier": "premium",
  "is_premium": true,
  ...
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

## 🔒 Authentication (TODO)

JWT authentication буде додано в Phase 1.2:

```bash
# Login (planned)
POST /api/v1/auth/login
{
  "username": "admin",
  "password": "secure_password"
}

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-12-25T10:00:00Z"
}

# Protected endpoints
GET /api/v1/users
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

## 📦 Структура

```
internal/api/
├── server.go              # HTTP server
├── middleware/            # Middleware
│   ├── logging.go         # Request logging
│   ├── recovery.go        # Panic recovery
│   ├── cors.go            # CORS headers
│   └── auth.go            # JWT auth (TODO)
├── handlers/              # Request handlers
│   ├── health_handler.go  # Health check
│   ├── user_handler.go    # User management
│   └── stats_handler.go   # Statistics
├── auth/                  # Authentication (TODO)
│   └── jwt.go             # JWT helpers
└── websocket/             # WebSocket (TODO)
    └── monitor.go         # Real-time monitoring
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

### Phase 1 (Current)
- [x] HTTP Server setup
- [x] Middleware (logging, CORS, recovery)
- [x] Health check endpoints
- [x] User management endpoints (basic)
- [x] Statistics endpoints (basic)
- [ ] JWT Authentication
- [ ] Admin User model + repository
- [ ] Rate limiting middleware

### Phase 2
- [ ] Opportunities management endpoints
- [ ] Arbitrage management endpoints
- [ ] DeFi management endpoints
- [ ] Notification management

### Phase 3
- [ ] WebSocket real-time monitoring
- [ ] Broadcast system
- [ ] Payment management
- [ ] System control endpoints

### Phase 4
- [ ] Swagger/OpenAPI documentation
- [ ] Unit + Integration tests
- [ ] Frontend dashboard (React/Retool)
- [ ] Docker deployment

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
