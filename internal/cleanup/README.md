# Cleanup Package

Автоматичне очищення старих даних з бази даних.

## Опис

Cleanup scheduler автоматично видаляє застарілі дані з бази даних для підтримки продуктивності та економії дискового простору.

## Що очищається

### 1. Opportunities (звичайні можливості)
- **Retention:** 30 днів
- **Причина:** Лаунчпули, аірдропи, та інші opportunities зазвичай короткострокові

### 2. Arbitrage Opportunities
- **Retention:** 7 днів
- **Причина:** Арбітражні можливості мають дуже короткий life cycle (хвилини)
- **Примітка:** Зберігаються довше для історичної аналітики

### 3. DeFi Opportunities
- **Retention:** 7 днів
- **Причина:** APY швидко змінюються, старі дані не актуальні

### 4. Notifications
- **Sent notifications:** 90 днів
- **Failed notifications:** 30 днів
- **Причина:** Зберігаються для аудиту та troubleshooting

## Розклад

**Default:** Щодня о 2:00 AM (UTC)
**Cron expression:** `0 2 * * *`

## Конфігурація

```go
config := &cleanup.Config{
    OpportunitiesRetentionDays:       30,
    ArbitrageRetentionDays:           7,
    DeFiRetentionDays:                7,
    SentNotificationsRetentionDays:   90,
    FailedNotificationsRetentionDays: 30,
    Schedule:                         "0 2 * * *",
}

scheduler := cleanup.NewScheduler(oppRepo, arbRepo, defiRepo, notifRepo, config)
```

## Використання

### Автоматичний запуск

Cleanup scheduler автоматично запускається в `cmd/bot/main.go`:

```go
cleanupScheduler := cleanup.NewScheduler(oppRepo, arbRepo, defiRepo, notifRepo, nil)
if err := cleanupScheduler.Start(); err != nil {
    log.Fatalf("Failed to start cleanup scheduler: %v", err)
}
defer cleanupScheduler.Stop()
```

### Ручний запуск (для тестування)

```go
cleanupScheduler.RunNow()
```

## Логування

Cleanup scheduler логує всі операції з емоджі для зручності моніторингу:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🧹 Cleanup Job Started
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🗑️  Cleaning up opportunities older than 30 days...
✅ Opportunities cleanup completed
🗑️  Cleaning up arbitrage opportunities older than 7 days...
✅ Arbitrage cleanup completed
🗑️  Cleaning up DeFi opportunities older than 7 days...
✅ DeFi cleanup completed
🗑️  Cleaning up old notifications...
✅ Notifications cleanup completed
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Cleanup completed in 234ms
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Моніторинг

Рекомендується моніторити:
- Час виконання cleanup операцій
- Кількість видалених записів
- Помилки під час cleanup

## Покращення (TODO)

- [ ] Додати метрики (кількість видалених записів)
- [ ] Додати backup before cleanup (optional)
- [ ] Налаштування через config.yaml
- [ ] Email/Telegram алерти при помилках
- [ ] Прогресивне видалення (batches) для великих таблиць
