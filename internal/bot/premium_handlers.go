package bot

import (
	_ "crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/payment/monobank"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleBuyPremium(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)

		return
	}

	if user.IsPremium() {
		text := fmt.Sprintf(
			"💎 У тебе вже є Premium підписка!\n\n"+
				"Активна до: %s\n"+
				"Залишилось: %d днів\n\n"+
				"Хочеш керувати підпискою? /subscription",
			user.SubscriptionExpiresAt.Format("02.01.2006"),
			b.daysUntil(*user.SubscriptionExpiresAt),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.buildSubscriptionManagementKeyboard()

		b.sendMessage(msg)

		return
	}

	text := `💎 <b>Premium Підписка</b>
З Premium ти отримуєш:
⚡ Real-time алерти (0-2 хв затримка)
💰 Арбітражні можливості (10-20/день)
🎯 Персоналізовані фільтри
📊 Детальну аналітику
🔥 DeFi та китові алерти
🎁 Unlimited алерти (Free: 5/день)

<b>Обери план:</b>

`
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildPremiumPlansKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) handleSubscription(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)

		return
	}

	if !user.IsPremium() {
		text := "⚠️ У тебе немає активної Premium підписки.\n\n" +
			"Хочеш спробувати Premium? /buy_premium"

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.buildPremiumKeyboard()

		b.sendMessage(msg)

		return
	}

	subscription, err := b.subsRepo.GetActiveByUserID(user.ID)
	if err != nil {
		log.Printf("Failed to get subscription: %v", err)
		b.sendError(chatID)

		return
	}

	planName := b.getPlanNameUA(subscription.Plan)
	priceUAH := float64(subscription.Amount) / 100

	text := fmt.Sprintf(
		"💎 <b>Твоя Premium підписка</b>\n\n"+
			"📋 План: %s\n"+
			"💵 Ціна: %.2f UAH\n"+
			"📅 Активна до: %s\n"+
			"⏰ Залишилось: %d днів\n"+
			"🔄 Автопродовження: %s\n\n",
		planName,
		priceUAH,
		subscription.CurrentPeriodEnd.Format("02.01.2006 15:04"),
		subscription.DaysLeft(),
		b.getAutoRenewStatus(subscription.AutoRenew),
	)

	if subscription.CancelAtPeriodEnd {
		text += "⚠️ Підписка буде скасована після закінчення періоду\n"
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildSubscriptionManagementKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) handlePremiumCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	// Витягнути plan з callback data (формат: "premium:plan_name")
	plan := callback.Data[8:] // Пропустити "premium:"

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)

		return
	}

	if user.IsPremium() {
		_, errMsg := b.api.Request(tgbotapi.NewCallbackWithAlert(callback.ID, "У тебе вже Premium!"))
		if errMsg != nil {
			b.sendError(chatID)
		}

		return
	}

	// Відобразити wait message
	b.sendMessage(tgbotapi.NewCallback(callback.ID, "Створюю рахунок для оплати..."))

	trialDays := 0

	if plan == monobank.PlanPremiumWeekly {
		trialDays = 7
	}

	subscription, paymentURL, err := b.paymentService.CreateSubscription(user.ID, plan, trialDays)
	if err != nil {
		log.Printf("Failed to create subscription: %v", err)

		b.sendMessage(tgbotapi.NewCallbackWithAlert(callback.ID, "Помилка при створенні підписки. Спробуй пізніше."))

		return
	}

	planName := b.getPlanNameUA(plan)

	priceUAH := float64(monobank.PlanPrices[plan]) / 100

	if trialDays > 0 {
		text := fmt.Sprintf(
			"🎉 <b>Trial активовано!</b>\n\n"+
				"Ти отримав %d днів Premium <b>безкоштовно</b>!\n\n"+
				"💎 Всі Premium функції доступні прямо зараз:\n"+
				"⚡ Real-time алерти\n"+
				"💰 Арбітражні можливості\n"+
				"📊 Детальна аналітика\n\n"+
				"📅 Trial закінчується: %s\n\n"+
				"Насолоджуйся! 🚀",
			trialDays,
			subscription.CurrentPeriodEnd.Format("02.01.2006"),
		)

		newMsg := tgbotapi.NewMessage(chatID, text)
		newMsg.ParseMode = "HTML"
		newMsg.ReplyMarkup = b.buildMainMenuKeyboard()
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)

		b.sendMessage(deleteMsg)
		b.sendMessage(newMsg)

		return
	}

	// Платна підписка
	text := fmt.Sprintf(
		"💳 <b>Оплата підписки</b>\n\n"+
			"📋 План: %s\n"+
			"💵 Ціна: %.2f UAH\n\n"+
			"Натисни кнопку нижче для оплати через Monobank.\n\n"+
			"✅ Безпечна оплата\n"+
			"💳 Приймаються всі картки\n"+
			"🔒 Захищено Monobank",
		planName,
		priceUAH,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("💳 Оплатити "+fmt.Sprintf("%.0f UAH", priceUAH), paymentURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Скасувати", "cancel_payment"),
		),
	)

	newMsg := tgbotapi.NewMessage(chatID, text)
	newMsg.ParseMode = "HTML"
	newMsg.ReplyMarkup = keyboard
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)

	b.sendMessage(deleteMsg)
	b.sendMessage(newMsg)

	log.Printf("✅ Payment link sent to user %d: %s", user.ID, paymentURL)
}

// handleCancelSubscription скасовує підписку
func (b *Bot) handleCancelSubscription(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)

		return
	}

	// Скасувати підписку через payment service
	if err := b.paymentService.CancelSubscription(user.ID, false, "Скасовано користувачем"); err != nil {
		log.Printf("Failed to cancel subscription: %v", err)

		b.sendMessage(tgbotapi.NewCallbackWithAlert(callback.ID, "Помилка при скасуванні підписки"))

		return
	}

	b.sendMessage(tgbotapi.NewCallback(callback.ID, "Підписку скасовано"))

	text := "⏸️ <b>Підписку скасовано</b>\n\n" +
		"Твоя Premium підписка залишиться активною до кінця оплаченого періоду.\n\n" +
		"Автопродовження вимкнено."

	newMsg := tgbotapi.NewMessage(chatID, text)
	newMsg.ParseMode = "HTML"
	newMsg.ReplyMarkup = b.buildMainMenuKeyboard()
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)

	b.sendMessage(deleteMsg)
	b.sendMessage(newMsg)

	log.Printf("✅ User %d canceled subscription", user.ID)
}

func (b *Bot) getPlanNameUA(plan string) string {
	names := map[string]string{
		monobank.PlanPremiumMonthly: "💎 Місячна",
		monobank.PlanPremiumWeekly:  "⚡ Тижнева",
		monobank.PlanPremiumYearly:  "👑 Річна",
	}

	if name, ok := names[plan]; ok {
		return name
	}

	return "Premium"
}

func (b *Bot) getAutoRenewStatus(autoRenew bool) string {
	if autoRenew {
		return "✅ Увімкнено"
	}

	return "❌ Вимкнено"
}

func (b *Bot) daysUntil(t time.Time) int {
	d := t.Sub(time.Now())
	if d < 0 {
		return 0
	}

	return int(d.Hours() / 24)
}

// handleClient показує інформацію про Premium Trading Client
func (b *Bot) handleClient(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	if !user.IsPremium() {
		text := "⚠️ Premium Trading Client доступний тільки для Premium користувачів.\n\n" +
			"Хочеш спробувати Premium? /buy_premium"

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.buildPremiumKeyboard()
		b.sendMessage(msg)
		return
	}

	text := `🖥 <b>Premium Trading Client</b>

Desktop додаток для автоматичної торгівлі арбітражем на твоїх пристроях!

<b>Переваги:</b>
🔐 API ключі зберігаються на твоєму пристрої
⚡ Миттєве виконання трейдів
💰 Автоматична торгівля 24/7
📊 Детальна статистика
🎯 Повний контроль над коштами

<b>Завантаження:</b>
🪟 Windows: bit.ly/client-win
🐧 Linux: bit.ly/client-linux
🍎 MacOS: bit.ly/client-mac

📖 Інструкція: bit.ly/client-docs
🔑 Налаштування API ключів: bit.ly/client-api

<b>Статистика:</b>
Подивись свою статистику торгівлі: /clientstats`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.sendMessage(msg)

	log.Printf("✅ User %d requested client info", user.ID)
}

// handleClientStats показує статистику торгівлі користувача
func (b *Bot) handleClientStats(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	if !user.IsPremium() {
		text := "⚠️ Статистика доступна тільки для Premium користувачів.\n\n" +
			"Хочеш спробувати Premium? /buy_premium"

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.buildPremiumKeyboard()
		b.sendMessage(msg)
		return
	}

	// TODO: Отримати статистику через clientStatsRepo коли він буде доданий до Bot
	// Поки що показуємо заглушку
	text := `📊 <b>Твоя Статистика Торгівлі</b>

🔄 Всього трейдів: 0
✅ Успішних: 0
❌ Провалених: 0

💰 Чистий прибуток: $0.00
📈 Win rate: 0%
🏆 Кращий трейд: $0.00

⏰ Остання торгівля: Ніколи

<i>Статистика оновиться після першого трейду через Premium Client</i>

Завантажити клієнт: /client`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.sendMessage(msg)

	log.Printf("✅ User %d requested client stats", user.ID)
}
