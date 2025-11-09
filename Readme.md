# Crypto Opportunities Bot

Telegram бот для моніторингу та сповіщення про прибуткові можливості в криптовалютному просторі (аірдропи, лаунчпули, арбітраж, DeFi).

## 🚀 Швидкий старт

### Передумови

- Go 1.25.3+
- PostgreSQL 14+
- Redis 7+ (опціонально)
- Telegram Bot Token (отримати у [@BotFather](https://t.me/BotFather))

### Інсталяція

1. **Клонувати репозиторій**
```bash
git clone <repository-url>
cd crypto-opportunities-bot
```

2. **Встановити залежності**
```bash
go mod download
```

3. **Налаштувати БД**
```bash
# Запустити PostgreSQL через Docker
docker-compose -f docker/docker-compose.yml up -d

# Або використати існуючу PostgreSQL
```

4. **Налаштувати конфігурацію**
```bash
# Скопіювати .env.example
cp .env.example .env

# Заповнити необхідні змінні
nano .env
```

Мінімальна конфігурація в `.env`:
```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=crypto_bot_dev
```

5. **Запустити бота**
```bash
go run cmd/bot/main.go
```

## 📁 Структура проекту

```
crypto-opportunities-bot/
├── cmd/
│   ├── bot/          # Telegram Bot entrypoint
│   ├── api/          # REST API (майбутнє)
│   └── worker/       # Background workers (майбутнє)
├── internal/
│   ├── bot/          # Telegram Bot logic
│   ├── config/       # Configuration management
│   ├── models/       # Database models
│   ├── repository/   # Data access layer
│   ├── notification/ # Notification system
│   ├── scraper/      # Exchange scrapers
│   ├── logger/       # Structured logging
│   └── ratelimit/    # Rate limiting
├── configs/          # Configuration files
└── docker/           # Docker configs
```

## 🔧 Конфігурація

### configs/config.yaml

```yaml
app:
  environment: development
  port: 8080
  log_level: debug

telegram:
  bot_token: ""  # Або через змінну оточення
  webhook_url: ""
  debug: true

database:
  host: localhost
  port: 5432
  user: postgres
  password: ""
  dbname: crypto_bot_dev
  sslmode: disable
  max_conns: 25
```

### Environment Variables

Змінні оточення мають пріоритет над config.yaml:

- `TELEGRAM_BOT_TOKEN` - Telegram Bot API token
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` - Redis (опціонально)

## 🎯 Функціонал

### Реалізовано (MVP)

✅ **Telegram Bot**
- Onboarding з налаштуванням профілю
- Команди: /start, /help, /today, /stats, /settings, /premium
- Inline keyboards для навігації
- Персоналізація за капіталом та ризик-профілем

✅ **Scraper System**
- Binance: Launchpool, Airdrops, Learn & Earn
- Bybit: Launchpool, Airdrops, Learn & Earn
- Автоматичний scraping кожні 5 хвилин
- Деактивація застарілих можливостей

✅ **Notification System**
- Створення персоналізованих нотифікацій
- Фільтрація за ROI, капіталом, типами
- Затримка для Free користувачів (20 хв)
- Retry mechanism для failed notifications
- Daily Digest о 09:00 UTC

✅ **Database Layer**
- PostgreSQL з GORM
- Models: User, UserPreferences, Opportunity, Notification
- Repository pattern
- Auto-migrations

### В розробці

🔨 **Stripe Payments**
- Інтеграція з Stripe Checkout
- Webhook обробка
- Автоматична активація Premium

🔨 **Advanced Features**
- Арбітраж моніторинг
- DeFi opportunities
- Whale alerts

## 📊 База даних

### Міграції

Міграції виконуються автоматично при старті застосунку через GORM AutoMigrate.

### Основні таблиці

- `users` - Користувачі бота
- `user_preferences` - Налаштування користувачів
- `opportunities` - Знайдені можливості
- `notifications` - Черга повідомлень

## 🤖 Telegram Bot команди

- `/start` - Початок роботи, onboarding
- `/help` - Довідка по командам
- `/today` - Можливості на сьогодні
- `/stats` - Статистика користувача
- `/settings` - Налаштування профілю
- `/premium` - Інформація про Premium
- `/support` - Контакти підтримки

## 🔐 Безпека

- ✅ Prepared statements (GORM)
- ✅ Input validation
- ✅ Rate limiting (planned)
- ✅ Environment variables для secrets
- ⏳ SSL/TLS для production

## 📈 Моніторинг

### Логування

Всі логи виводяться в stdout з timestamp та log level:

```
[2025-11-09 10:30:15] INFO: ✅ Database initialized
[2025-11-09 10:30:16] INFO: ✅ Scraper scheduler started
```

### Метрики

- Кількість користувачів (Free/Premium)
- Кількість знайдених можливостей
- Успішність відправки нотифікацій
- Uptime scrapers

## 🚢 Deployment

### Development

```bash
go run cmd/bot/main.go
```

### Production

```bash
# Build
go build -o bot cmd/bot/main.go

# Run
./bot
```

### Docker (майбутнє)

```bash
docker-compose up -d
```

## 📝 Contributing

1. Fork репозиторій
2. Створити feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit зміни (`git commit -m 'Add AmazingFeature'`)
4. Push до branch (`git push origin feature/AmazingFeature`)
5. Відкрити Pull Request

## 🗺️ Roadmap

### Phase 1 (MVP) - ✅ Completed
- [x] Базова структура проекту
- [x] Telegram bot з onboarding
- [x] Binance та Bybit scrapers
- [x] Notification system
- [x] Daily digest

### Phase 2 - 🔨 In Progress
- [ ] Stripe payment integration
- [ ] /settings редагування
- [ ] Admin panel (REST API)
- [ ] Детальна статистика

### Phase 3
- [ ] Арбітраж моніторинг
- [ ] DeFi opportunities
- [ ] Whale alerts
- [ ] OKX, Gate.io, Kraken scrapers

### Phase 4
- [ ] Mobile app (PWA)
- [ ] Реферальна програма
- [ ] AI-powered рекомендації

---

**⚠️ Disclaimer**: Цей бот не надає фінансових порад. Всі інвестиційні рішення користувачі приймають на власний ризик.