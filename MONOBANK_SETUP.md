# 🏦 Monobank Integration - Інструкція з налаштування

## 📋 Зміст
1. [Отримання API ключів](#1-отримання-api-ключів)
2. [Налаштування проекту](#2-налаштування-проекту)
3. [Тестування](#3-тестування)
4. [Production deployment](#4-production-deployment)
5. [Troubleshooting](#5-troubleshooting)

---

## 1. Отримання API ключів

### Крок 1.1: Реєстрація Acquiring
1. Відкрийте https://api.monobank.ua/
2. Натисніть "Зареєструватись"
3. Заповніть форму (потрібен ФОП або ТОВ)
4. Очікуйте підтвердження (1-2 робочі дні)

### Крок 1.2: Отримання токенів
Після активації Acquiring:

1. Перейдіть в особистий кабінет: https://web.monobank.ua/
2. Розділ "Acquiring" → "Налаштування" → "API"
3. Згенеруйте **API Token** (X-Token)
4. Збережіть **Public Key** для верифікації webhook

**❗ ВАЖЛИВО:** Токени показуються тільки раз! Збережіть їх в безпечному місці.

---

## 2. Налаштування проекту

### Крок 2.1: Environment Variables

Створіть `.env` файл або додайте в існуючий:

```bash
# Monobank Configuration
MONOBANK_TOKEN=your_monobank_api_token_here
MONOBANK_PUBLIC_KEY=your_public_key_here
PAYMENT_WEBHOOK_URL=https://yourdomain.com/webhook/monobank
PAYMENT_REDIRECT_URL=https://t.me/your_bot_username
PAYMENT_WEBHOOK_PORT=8081
```

**Де взяти значення:**
- `MONOBANK_TOKEN` - з особистого кабінету Monobank (X-Token)
- `MONOBANK_PUBLIC_KEY` - з особистого кабінету Monobank
- `PAYMENT_WEBHOOK_URL` - ваш публічний домен + `/webhook/monobank`
- `PAYMENT_REDIRECT_URL` - посилання на ваш Telegram бот
- `PAYMENT_WEBHOOK_PORT` - порт для webhook сервера (8081 за замовчуванням)

### Крок 2.2: Config файл

Перевірте `configs/config.yaml`:

```yaml
payment:
  monobank_token: ""  # Буде взято з .env
  monobank_public_key: ""  # Буде взято з .env
  webhook_url: "https://yourdomain.com/webhook/monobank"
  redirect_url: "https://t.me/your_bot_username"
  webhook_port: "8081"
```

---

## 3. Тестування

### Крок 3.1: Локальне тестування з ngrok

Для тестування webhook локально використовуйте ngrok:

```bash
# 1. Встановіть ngrok
# macOS: brew install ngrok
# Linux: https://ngrok.com/download

# 2. Запустіть ngrok
ngrok http 8081

# 3. Скопіюйте HTTPS URL (наприклад: https://abc123.ngrok.io)

# 4. Додайте в .env
PAYMENT_WEBHOOK_URL=https://abc123.ngrok.io/webhook/monobank
```

### Крок 3.2: Запуск проекту

```bash
# 1. Встановіть dependencies
make install-deps

# 2. Запустіть базу даних
make docker-up

# 3. Запустіть бота
make run
```

### Крок 3.3: Тестовий платіж

1. Відкрийте Telegram бот
2. Відправте `/buy_premium`
3. Оберіть план (наприклад, Тижнева)
4. Натисніть кнопку "Оплатити"
5. Використайте **тестову картку**:

```
Номер: 5375 4112 3456 7890
Термін: будь-який майбутній
CVV: будь-який 3-значний
```

6. Перевірте логи бота - має з'явитись webhook callback
7. Перевірте статус: `/subscription`

### Крок 3.4: Перевірка webhook

Після успішного платежу:

```bash
# У логах бота має з'явитись:
📥 Monobank webhook: invoice=xxx, status=success, amount=9900
✅ Payment successful: subscription=1, invoice=xxx
🎉 User 123 upgraded to Premium until 2025-12-16
```

---

## 4. Production Deployment

### Крок 4.1: Домен та SSL

**Обов'язково:** Monobank вимагає HTTPS для webhook!

```bash
# Варіант 1: VPS з Let's Encrypt
sudo certbot --nginx -d yourdomain.com

# Варіант 2: Cloudflare (безкоштовний SSL)
# 1. Додайте домен в Cloudflare
# 2. Увімкніть Proxy (оранжева хмарка)
# 3. SSL буде автоматично
```

### Крок 4.2: Nginx конфігурація

```nginx
server {
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    # Webhook endpoint
    location /webhook/monobank {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Sign $http_x_sign;  # ВАЖЛИВО!
    }
}
```

### Крок 4.3: Production ENV

```bash
# Production .env
MONOBANK_TOKEN=prod_token_here
MONOBANK_PUBLIC_KEY=prod_public_key_here
PAYMENT_WEBHOOK_URL=https://yourdomain.com/webhook/monobank
PAYMENT_REDIRECT_URL=https://t.me/your_bot_username
PAYMENT_WEBHOOK_PORT=8081
```

### Крок 4.4: Запуск з systemd

`/etc/systemd/system/crypto-bot.service`:

```ini
[Unit]
Description=Crypto Opportunities Bot
After=network.target postgresql.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/crypto-opportunities-bot
ExecStart=/home/ubuntu/crypto-opportunities-bot/crypto-bot
Restart=always
RestartSec=10
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
EnvironmentFile=/home/ubuntu/crypto-opportunities-bot/.env

[Install]
WantedBy=multi-user.target
```

```bash
# Активація
sudo systemctl daemon-reload
sudo systemctl enable crypto-bot
sudo systemctl start crypto-bot
sudo systemctl status crypto-bot
```

### Крок 4.5: Моніторинг webhook в Monobank

1. Зайдіть в особистий кабінет Monobank
2. Розділ "Acquiring" → "Webhook"
3. Перевірте що URL правильний
4. Статус має бути "Active"
5. Переглянте історію webhook calls

---

## 5. Troubleshooting

### Проблема: Webhook не приходить

**Рішення:**

1. **Перевірте URL:**
   ```bash
   # Має бути HTTPS!
   echo $PAYMENT_WEBHOOK_URL
   # Правильно: https://yourdomain.com/webhook/monobank
   # Неправильно: http://localhost:8081/webhook/monobank
   ```

2. **Перевірте доступність:**
   ```bash
   curl https://yourdomain.com/webhook/monobank
   # Має повернути {"status":"success"} або error
   ```

3. **Перевірте порт:**
   ```bash
   # Порт 8081 має слухатись
   netstat -tuln | grep 8081
   ```

4. **Перевірте логи:**
   ```bash
   # У логах бота має бути:
   🌐 Webhook server starting on port 8081
   ```

### Проблема: Invalid signature

**Рішення:**

1. Перевірте `MONOBANK_PUBLIC_KEY` в .env
2. Public key має співпадати з ключем в особистому кабінеті
3. У production обов'язково перевіряйте підпис

### Проблема: Payment створюється але не активується

**Рішення:**

1. Перевірте логи webhook:
   ```bash
   tail -f /var/log/crypto-bot/app.log | grep webhook
   ```

2. Перевірте БД:
   ```sql
   SELECT * FROM subscriptions WHERE status = 'pending';
   SELECT * FROM payments WHERE status = 'pending';
   ```

3. Перевірте що webhook обробляється:
   ```bash
   # У логах має бути:
   📥 Webhook received: invoice=xxx, status=success
   ✅ Payment successful
   🎉 User X upgraded to Premium
   ```

### Проблема: Subscription не продовжується автоматично

**Рішення:**

1. Перевірте що збережена картка:
   ```sql
   SELECT monobank_wallet_id FROM subscriptions WHERE user_id = X;
   # Має бути не NULL
   ```

2. Перевірте subscription checker:
   ```bash
   # У логах має бути кожну годину:
   ✅ Subscription checker started (every 1h)
   ```

3. Логи renewal:
   ```bash
   tail -f /var/log/crypto-bot/app.log | grep renewal
   ```

---

## 📊 API Endpoints

### Webhook Endpoint
```
POST /webhook/monobank
Content-Type: application/json
X-Sign: <signature>

Body: {
  "invoiceId": "abc123",
  "status": "success",
  "amount": 24900,
  "reference": "sub_xxx_123",
  ...
}
```

### Health Check
```
GET /health

Response: {
  "status": "healthy"
}
```

---

## 🔒 Безпека

1. **Ніколи не коммітьте** `.env` файл в git
2. **Завжди перевіряйте** webhook signature в production
3. **Використовуйте HTTPS** для webhook URL
4. **Зберігайте токени** в безпечному місці (1Password, Vault)
5. **Регулярно ротуйте** API токени (раз на 6 місяців)

---

## 💰 Ціни та комісії

### Комісії Monobank:
- **Українські картки:** 1.5-1.8%
- **Міжнародні картки:** 2.5%
- **Recurring платежі:** така ж комісія

### Рекомендовані ціни:
- **Тижнева:** 99 UAH (психологічний поріг <100)
- **Місячна:** 249 UAH (оптимальна для UA)
- **Річна:** 2499 UAH (знижка 16%, психологічний поріг <2500)

---

## 📞 Підтримка

**Monobank Acquiring:**
- Email: acquiring@monobank.ua
- Telegram: @monobank_acquiring
- Документація: https://api.monobank.ua/docs/

**Проблеми з інтеграцією:**
- Перевірте логи: `make logs`
- Перевірте БД: `make db-shell`
- Issue tracker: GitHub Issues

---

## ✅ Чеклист перед production

- [ ] Monobank Acquiring активовано
- [ ] API токени отримані та збережені
- [ ] Домен налаштовано з SSL
- [ ] Webhook URL HTTPS
- [ ] `.env` файл налаштовано
- [ ] Тестовий платіж пройшов успішно
- [ ] Webhook приходить та обробляється
- [ ] Subscription автоматично активується
- [ ] Systemd service налаштовано
- [ ] Моніторинг працює
- [ ] Backup БД налаштовано

---

**Готово! 🚀 Ваш бот готовий приймати платежі через Monobank!**
