# CLAUDE.md - AI Assistant Development Guide

## Project Overview

**Crypto Opportunities Bot** is a Telegram bot that monitors and notifies users about profitable opportunities in the cryptocurrency space, including airdrops, launchpools, arbitrage, and DeFi opportunities.

### Key Features
- Real-time scraping of crypto opportunities from exchanges (Binance, Bybit)
- Personalized notifications based on user preferences
- Telegram bot interface with onboarding flow
- Daily digest system
- Freemium subscription model (Free vs Premium)
- PostgreSQL database with GORM ORM
- Automated scraping every 5 minutes

### Tech Stack
- **Language**: Go 1.25.3+
- **Database**: PostgreSQL 14+ (GORM ORM)
- **Cache**: Redis 7+ (optional)
- **Bot Framework**: go-telegram-bot-api/v5
- **Config**: Viper (YAML + env vars)
- **Scheduling**: robfig/cron/v3
- **Payment**: Stripe (planned)

---

## Architecture & Structure

### Directory Layout

```
crypto-opportunities-bot/
├── cmd/                    # Application entrypoints
│   ├── bot/               # Main Telegram bot (active)
│   ├── api/               # REST API server (planned)
│   └── worker/            # Background workers (planned)
├── internal/              # Private application code
│   ├── bot/              # Telegram bot logic
│   │   ├── bot.go        # Bot initialization
│   │   ├── handlers.go   # Message/command handlers
│   │   ├── keyboards.go  # Inline keyboards
│   │   ├── onboarding.go # User onboarding flow
│   │   └── commands.go   # Command routing
│   ├── config/           # Configuration management
│   ├── models/           # Database models (GORM)
│   ├── repository/       # Data access layer (Repository pattern)
│   ├── notification/     # Notification system
│   │   ├── service.go    # Notification creation/sending
│   │   ├── filter.go     # User preference filtering
│   │   ├── formatter.go  # Message formatting
│   │   └── digest_scheduler.go # Daily digest cron
│   ├── scraper/          # Exchange scrapers
│   │   ├── binance_scraper.go
│   │   ├── bybit_scraper.go
│   │   ├── scheduler.go  # Scraper cron jobs
│   │   └── parser.go     # HTML/JSON parsing utilities
│   ├── logger/           # Structured logging
│   └── ratelimit/        # Rate limiting (planned)
├── configs/              # Configuration files
│   ├── config.yaml       # Development config
│   └── config.prod.yaml  # Production config
├── docker/               # Docker setup
│   ├── Dockerfile
│   └── docker-compose.yml
├── Makefile              # Build and dev commands
├── go.mod                # Go modules
└── Readme.md             # User documentation
```

### Application Flow

1. **Startup** (`cmd/bot/main.go`):
   - Load config from `configs/config.yaml` + env vars
   - Initialize database connection
   - Run GORM auto-migrations
   - Initialize repositories
   - Start scraper scheduler (every 5 min)
   - Start notification dispatcher (every 10 sec)
   - Start daily digest scheduler (09:00 UTC)
   - Start Telegram bot polling

2. **Scraping Flow**:
   - Scheduler triggers scrapers (Binance, Bybit)
   - Scrapers fetch data from exchange APIs
   - Parse JSON responses into `Opportunity` models
   - Check for duplicates using `ExternalID` (MD5 hash)
   - Save new opportunities to database
   - Trigger notification creation for new opportunities

3. **Notification Flow**:
   - New opportunity detected → create notifications for eligible users
   - Filter users by preferences (capital, risk, types, exchanges, ROI)
   - Apply delays (20 min for free users, instant for premium)
   - Notification dispatcher sends pending notifications
   - Retry failed notifications with exponential backoff

---

## Key Components

### Models (`internal/models/`)

All models extend `BaseModel` which provides:
- `ID` (uint, primary key)
- `CreatedAt`, `UpdatedAt`, `DeletedAt` (GORM timestamps)

#### Core Models:

**User** (`user.go`):
- Telegram user information
- Subscription tier (free/premium)
- Capital range and risk profile
- Methods: `IsPremium()`, `IsSubscriptionActive()`

**UserPreferences** (`user_preferences.go`):
- Notification settings
- Opportunity type filters (array)
- Exchange filters (array)
- Min ROI threshold
- Max investment limit

**Opportunity** (`opportunity.go`):
- External ID (unique MD5 hash: `exchange:type:id`)
- Exchange (binance, bybit, okx, etc.)
- Type (launchpool, airdrop, learn_earn, staking, arbitrage, defi)
- Title, description, reward
- Estimated ROI, pool size, min investment
- Start/end dates, duration
- URL, image URL
- Active status
- Methods: `IsExpired()`, `DaysLeft()`, `IsHighROI()`

**Notification** (`notification.go`):
- Links to User and Opportunity
- Status (pending, sent, failed)
- Send after timestamp (for delays)
- Retry count
- Error message

### Repositories (`internal/repository/`)

**Pattern**: Repository pattern for data access abstraction

Each repository provides CRUD operations:
- `Create(model)` - Insert new record
- `Update(model)` - Update existing record
- `GetByID(id)` - Find by primary key
- `Delete(id)` - Soft delete (GORM DeletedAt)
- Custom queries (e.g., `GetByTelegramID`, `ListActive`)

**Available Repositories**:
- `UserRepository`
- `UserPreferencesRepository`
- `OpportunityRepository`
- `NotificationRepository`
- `UserActionRepository`

### Scrapers (`internal/scraper/`)

**Interface**:
```go
type Scraper interface {
    GetExchange() string
    ScrapeAll() ([]*models.Opportunity, error)
}
```

**Implemented Scrapers**:
1. **BinanceScraper** (`binance_scraper.go`):
   - Launchpool: `/bapi/earn/v1/public/launchpool/project/list`
   - Airdrops: CMS article API (catalog 128)
   - Learn & Earn: CMS article API (catalog 220)

2. **BybitScraper** (`bybit_scraper.go`):
   - Similar structure to Binance
   - Fetches launchpool, airdrops, learn & earn

**Scraper Scheduler** (`scheduler.go`):
- Uses `robfig/cron/v3`
- Default: every 5 minutes (`*/5 * * * *`)
- Runs all registered scrapers
- Deactivates expired opportunities
- Callback system for new opportunities

### Notification System (`internal/notification/`)

**Service** (`service.go`):
- `CreateOpportunityNotifications(opp)` - Create notifications for eligible users
- `SendPendingNotifications(limit)` - Send batch of pending notifications
- `RetryFailedNotifications(limit)` - Retry failed notifications
- `SendDailyDigest(user)` - Send daily summary

**Filter** (`filter.go`):
- Filters opportunities based on user preferences
- Capital range matching
- Risk profile matching
- Type/exchange filtering
- ROI threshold

**Formatter** (`formatter.go`):
- Formats opportunities into Telegram messages
- HTML parsing for rich text
- Inline keyboards for actions

**Digest Scheduler** (`digest_scheduler.go`):
- Daily cron job at 09:00 UTC
- Sends personalized daily digest to active users

---

## Development Workflow

### Initial Setup

```bash
# 1. Clone repository
git clone <repository-url>
cd crypto-opportunities-bot

# 2. Install dependencies
make install-deps

# 3. Setup environment
make .env
# Edit .env with your credentials

# 4. Start PostgreSQL + Redis
make docker-up

# 5. Run the bot
make run
```

### Configuration

**Priority**: Environment variables > `config.yaml`

**Required Environment Variables**:
- `TELEGRAM_BOT_TOKEN` - Get from @BotFather
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`

**Optional**:
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`

**Config Validation**:
- Production mode requires webhook URL and Stripe credentials
- Development mode can skip Redis and Stripe

### Common Commands

```bash
# Development
make run              # Run the bot
make build            # Build binary
make dev              # Start docker + run bot
make test             # Run tests
make test-coverage    # Generate coverage report

# Database
make db-shell         # Open PostgreSQL shell
make db-backup        # Backup database
make db-restore       # Restore from backup
make db-reset         # Drop and recreate DB

# Docker
make docker-up        # Start containers
make docker-down      # Stop containers
make docker-logs      # View logs

# Code Quality
make fmt              # Format code
make vet              # Run go vet
make lint             # Run golangci-lint
make check            # Format + vet + test

# Production
make prod-build       # Build for production (static binary)
```

### Git Workflow

**Branching**:
- `main` - Production branch
- `develop` - Development branch
- `feature/*` - Feature branches
- `bugfix/*` - Bug fix branches

**Commit Messages**:
- Use conventional commits: `feat:`, `fix:`, `chore:`, `docs:`
- Be descriptive: "Added Bybit scraper for launchpool opportunities"

---

## Code Conventions

### Go Style

1. **Follow Go idioms**:
   - Use `gofmt` for formatting
   - Run `go vet` before commits
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)

2. **Error Handling**:
   ```go
   // Always check errors
   if err != nil {
       log.Printf("Error: %v", err)
       return err
   }

   // Wrap errors with context
   return fmt.Errorf("failed to create user: %w", err)
   ```

3. **Logging**:
   ```go
   // Use structured logging with emojis for readability
   log.Printf("✅ Database initialized")
   log.Printf("❌ Failed to send notification: %v", err)
   log.Printf("📢 Creating notifications for: %s", opp.Title)
   ```

4. **Nil Checks**:
   ```go
   // Always check for nil pointers
   if user == nil {
       return fmt.Errorf("user not found")
   }
   ```

### Database

1. **GORM Best Practices**:
   - Use prepared statements (default with GORM)
   - Avoid N+1 queries with `Preload`
   - Use transactions for multi-step operations
   - Let auto-migration handle schema changes

2. **Model Conventions**:
   - Embed `BaseModel` for timestamps
   - Use `gorm` tags for constraints
   - Implement `TableName()` method
   - Add helper methods (e.g., `IsExpired()`)

3. **Repository Pattern**:
   - Keep SQL logic in repositories
   - Return errors, don't handle them
   - Use meaningful method names

### Telegram Bot

1. **Message Structure**:
   - Use HTML parsing (`ParseMode = "HTML"`)
   - Keep messages concise
   - Use emojis for visual hierarchy
   - Provide inline keyboards for actions

2. **Command Handlers**:
   - Validate user input
   - Check user authentication
   - Handle errors gracefully
   - Provide helpful error messages

3. **Callback Queries**:
   - Use prefixed callback data: `action:param`
   - Always answer callback queries
   - Update message after callback

### Scrapers

1. **Error Handling**:
   - Don't fail if one scraper fails
   - Log individual scraper errors
   - Return partial results

2. **Data Parsing**:
   - Use regex for extracting structured data
   - Provide fallback values
   - Clean and validate data

3. **Unique IDs**:
   - Generate consistent external IDs
   - Format: `MD5(exchange:type:id)`
   - Prevents duplicate opportunities

---

## Database Schema

### Key Tables

**users**:
- `telegram_id` (unique) - Telegram user ID
- `subscription_tier` - "free" or "premium"
- `subscription_expires_at` - Premium expiration
- `capital_range` - User's investment capital
- `risk_profile` - User's risk tolerance

**user_preferences**:
- `user_id` (foreign key)
- `opportunity_types` (JSON array)
- `exchanges` (JSON array)
- `min_roi` (float)
- `max_investment` (int)
- `notify_instant`, `notify_daily`, `notify_weekly`

**opportunities**:
- `external_id` (unique) - MD5 hash
- `exchange`, `type` (indexed)
- `estimated_roi` (indexed)
- `end_date` (indexed)
- `is_active` (indexed)
- `metadata` (JSONB)

**notifications**:
- `user_id`, `opportunity_id` (foreign keys)
- `status` (pending, sent, failed)
- `send_after` - Scheduled send time
- `retry_count` - Failed retry attempts
- `error_message` - Last error

### Migrations

- Auto-migration runs on startup (`repository.AutoMigrate()`)
- Add new fields to models, restart app
- GORM handles non-destructive changes
- For destructive changes, manual migration needed

---

## Testing Guidelines

### Writing Tests

```go
// Test naming: TestFunctionName
func TestUserIsPremium(t *testing.T) {
    user := &models.User{
        SubscriptionTier: "premium",
        SubscriptionExpiresAt: &futureDate,
    }

    if !user.IsPremium() {
        t.Error("Expected user to be premium")
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./internal/models -v

# Run specific test
go test ./internal/models -run TestUserIsPremium -v
```

---

## Common Tasks for AI Assistants

### Adding a New Scraper

1. Create `internal/scraper/exchange_scraper.go`
2. Implement `Scraper` interface
3. Register in `cmd/bot/main.go`:
   ```go
   scraperService.RegisterScraper(scraper.NewExchangeScraper())
   ```

### Adding a New Command

1. Add handler in `internal/bot/handlers.go`:
   ```go
   func (b *Bot) handleNewCommand(message *tgbotapi.Message) {
       // Implementation
   }
   ```
2. Register in `internal/bot/commands.go`:
   ```go
   case "newcommand":
       b.handleNewCommand(message)
   ```

### Adding a New Model Field

1. Add field to model in `internal/models/`
2. Add GORM tags if needed
3. Restart app (auto-migration will run)
4. Update repository methods if needed

### Modifying Notification Logic

1. Update filter logic in `internal/notification/filter.go`
2. Update formatter in `internal/notification/formatter.go`
3. Update service methods in `internal/notification/service.go`

### Adding API Endpoints (Future)

1. Create handlers in `cmd/api/`
2. Use repository pattern for data access
3. Add authentication middleware
4. Document endpoints

---

## Troubleshooting

### Database Connection Issues

```bash
# Check if PostgreSQL is running
docker-compose -f docker/docker-compose.yml ps

# Check connection
docker-compose -f docker/docker-compose.yml exec postgres psql -U postgres -c "SELECT 1"

# Reset database
make db-reset
```

### Telegram Bot Not Responding

1. Check bot token in `.env`
2. Verify bot is running: `ps aux | grep bot`
3. Check logs for errors
4. Test token: `curl https://api.telegram.org/bot<TOKEN>/getMe`

### Scrapers Failing

1. Check exchange API status
2. Verify URLs haven't changed
3. Check rate limiting
4. Review error logs

### Notifications Not Sending

1. Check `notifications` table: `SELECT * FROM notifications WHERE status = 'failed'`
2. Verify user preferences allow notifications
3. Check Telegram bot permissions
4. Review `send_after` timestamps

---

## Important Notes for AI Assistants

### When Making Changes

1. **Read Before Writing**: Always read the file before editing
2. **Preserve Existing Patterns**: Follow established code patterns
3. **Test Thoroughly**: Run `make test` after changes
4. **Update Documentation**: Update this file if architecture changes
5. **Check Imports**: Ensure all imports are used

### Security Considerations

1. **Never commit secrets**: Use environment variables
2. **Validate user input**: Always sanitize Telegram input
3. **Use prepared statements**: GORM does this by default
4. **Rate limiting**: Implement for API endpoints
5. **SQL injection**: GORM prevents this, but validate anyway

### Performance

1. **Database queries**: Use indexes, avoid N+1 queries
2. **Telegram API**: Respect rate limits (30 msg/sec)
3. **Scraper frequency**: Balance freshness vs load
4. **Notification batching**: Process in chunks

### Code Review Checklist

- [ ] All errors are handled
- [ ] Logs include context
- [ ] No hardcoded credentials
- [ ] Tests pass
- [ ] Code is formatted (`make fmt`)
- [ ] No unused imports/variables
- [ ] Database queries are efficient
- [ ] Telegram messages are clear

---

## Project Roadmap

### Phase 1 (MVP) - ✅ Completed
- [x] Basic project structure
- [x] Telegram bot with onboarding
- [x] Binance and Bybit scrapers
- [x] Notification system
- [x] Daily digest

### Phase 2 - 🔨 In Progress
- [ ] Stripe payment integration
- [ ] /settings editing functionality
- [ ] Admin panel (REST API)
- [ ] Detailed statistics

### Phase 3 - Planned
- [ ] Arbitrage monitoring
- [ ] DeFi opportunities
- [ ] Whale alerts
- [ ] OKX, Gate.io, Kraken scrapers

### Phase 4 - Future
- [ ] Mobile app (PWA)
- [ ] Referral program
- [ ] AI-powered recommendations

---

## Resources

- **Telegram Bot API**: https://core.telegram.org/bots/api
- **GORM Documentation**: https://gorm.io/docs/
- **Go Documentation**: https://golang.org/doc/
- **Viper Config**: https://github.com/spf13/viper

---

## Contact

For questions or issues:
- Check README.md for user documentation
- Review logs for debugging
- Consult this file for architecture questions

---

**Last Updated**: 2025-11-16
**Version**: 1.0
**Maintained by**: AI Assistant
