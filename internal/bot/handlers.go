package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleStart(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	telegramID := message.From.ID

	user, err := b.userRepo.GetByTelegramID(telegramID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		b.sendError(chatID)
	}

	if user == nil {
		user = &models.User{
			TelegramID:   telegramID,
			Username:     message.From.UserName,
			FirstName:    message.From.FirstName,
			LastName:     message.From.LastName,
			LanguageCode: message.From.LanguageCode,
		}

		if err := b.userRepo.Create(user); err != nil {
			log.Printf("Error creating user: %v", err)
			b.sendError(chatID)
			return
		}

		b.startOnboarding(chatID, user)
		return
	}

	now := time.Now()
	user.LastActiveAt = &now
	if err := b.userRepo.Update(user); err != nil {
		log.Printf("Error updating user: %v", err)
		b.sendError(chatID)
		return
	}

	b.sendWelcomeBack(chatID, user)
}

func (b *Bot) sendWelcomeBack(chatID int64, user *models.User) {
	text := fmt.Sprintf(
		"👋 З поверненням, %s!\n\n"+
			"Що тебе цікавить?\n\n"+
			"/today - Можливості на сьогодні\n"+
			"/stats - Твоя статистика\n"+
			"/settings - Налаштування\n"+
			"/premium - Дізнатись про Premium",
		user.FirstName,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = b.buildMainMenuKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) handleHelp(message *tgbotapi.Message) {
	text := `
📚 Доступні команди:

/start - Почати роботу з ботом
/help - Показати цю довідку
/today - Можливості на сьогодні
/stats - Твоя статистика
/settings - Налаштування
/premium - Інформація про Premium
/support - Зв'язатись з підтримкою

💡 Підказка: Використовуй кнопки меню для швидкого доступу!
`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.sendMessage(msg)
}

func (b *Bot) handleToday(message *tgbotapi.Message) {
	// TODO: Тут буде логіка показу можливостей
	text := "📊 Можливості на сьогодні:\n\n" +
		"🔜 Скоро тут з'являться актуальні можливості!\n\n" +
		"Зараз я ще навчаюсь їх знаходити 🤖"

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.sendMessage(msg)
}

func (b *Bot) handleStats(message *tgbotapi.Message) {
	user, _ := b.userRepo.GetByTelegramID(message.From.ID)
	if user == nil {
		return
	}

	tier := "🆓 Free"
	if user.IsPremium() {
		tier = "💎 Premium"
	}

	text := fmt.Sprintf(
		"📊 Твоя статистика:\n\n"+
			"Підписка: %s\n"+
			"Реєстрація: %s\n\n"+
			"🔜 Детальна статистика буде скоро!",
		tier,
		user.CreatedAt.Format("02.01.2006"),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.sendMessage(msg)
}

func (b *Bot) handleSettings(message *tgbotapi.Message) {
	text := "⚙️ Налаштування:\n\n🔜 Скоро тут будуть налаштування!"
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.sendMessage(msg)
}

func (b *Bot) handlePremium(message *tgbotapi.Message) {
	text := `
💎 Premium підписка

З Premium ти отримуєш:
⚡ Real-time алерти (0-2 хв затримка)
💰 Арбітражні можливості (10-20/день)
🎯 Персоналізовані фільтри
📊 Детальну аналітику
🔥 DeFi та китові алерти

✨ Перші 7 днів - безкоштовно
💵 Потім: $9/місяць

Користувачі в середньому заробляють $150-300/міс
завдяки Premium функціям.
`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = b.buildPremiumKeyboard()
	b.sendMessage(msg)
}

func (b *Bot) handleSupport(message *tgbotapi.Message) {
	text := "📧 Підтримка:\n\n" +
		"Email: support@cryptobot.com\n" +
		"Telegram: @support_username\n\n" +
		"Ми відповімо протягом 24 годин!"

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.sendMessage(msg)
}

func (b *Bot) handleUnknown(message *tgbotapi.Message) {
	text := "❓ Невідома команда. Використовуй /help для списку команд."
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.sendMessage(msg)
}

func (b *Bot) sendError(chatID int64) {
	text := "❌ Сталася помилка. Спробуй пізніше або напиши в підтримку /support"
	msg := tgbotapi.NewMessage(chatID, text)
	b.sendMessage(msg)
}

func (b *Bot) sendMessage(message tgbotapi.Chattable) {
	_, err := b.api.Send(message)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
