package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"log"
	"strings"
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
	chatID := message.Chat.ID
	telegramID := message.From.ID

	user, err := b.userRepo.GetByTelegramID(telegramID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		text := "⚠️ Спочатку налаштуй свій профіль через /start"
		msg := tgbotapi.NewMessage(chatID, text)
		b.sendMessage(msg)
		return
	}

	opportunities, err := b.getFilteredOpportunities(user, prefs, 0)
	if err != nil {
		log.Printf("Error getting opportunities: %v", err)
		b.sendError(chatID)
		return
	}

	if len(opportunities) == 0 {
		text := "🔍 На жаль, зараз немає можливостей, які відповідають твоїм критеріям.\n\n" +
			"💡 Спробуй:\n" +
			"• Розширити фільтри у /settings\n" +
			"• Додати більше бірж\n" +
			"• Знизити мінімальний ROI"

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = b.buildMainMenuKeyboard()
		b.sendMessage(msg)
		return
	}

	b.sendOpportunitiesList(chatID, user, opportunities, 0, "all")
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

func (b *Bot) getFilteredOpportunities(user *models.User, prefs *models.UserPreferences, offset int) ([]*models.Opportunity, error) {
	limit := 20

	opportunities, err := b.oppRepo.ListActive(1000, 0)
	if err != nil {
		return nil, err
	}

	var filtered []*models.Opportunity

	for _, opp := range opportunities {
		if !b.shouldShowOpportunity(user, prefs, opp) {
			continue
		}
		filtered = append(filtered, opp)
	}

	start := offset
	end := offset + limit
	if start > len(filtered) {
		return []*models.Opportunity{}, nil
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], nil
}

func (b *Bot) shouldShowOpportunity(user *models.User, prefs *models.UserPreferences, opp *models.Opportunity) bool {
	if !opp.IsActive || opp.IsExpired() {
		return false
	}

	isPremiumOpp := opp.Type == models.OpportunityTypeArbitrage || opp.Type == models.OpportunityTypeDeFi
	if isPremiumOpp && !user.IsPremium() {
		return false
	}

	if len(prefs.OpportunityTypes) > 0 {
		found := false
		for _, t := range prefs.OpportunityTypes {
			if t == opp.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(prefs.Exchanges) > 0 {
		found := false
		for _, ex := range prefs.Exchanges {
			if ex == opp.Exchange {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if opp.EstimatedROI > 0 && opp.EstimatedROI < prefs.MinROI {
		return false
	}

	if prefs.MaxInvestment > 0 && opp.MinInvestment > float64(prefs.MaxInvestment) {
		return false
	}

	return true
}

func (b *Bot) sendOpportunitiesList(chatID int64, user *models.User, opportunities []*models.Opportunity, page int, filter string) {
	if len(opportunities) == 0 {
		text := "🔍 Можливостей не знайдено"
		msg := tgbotapi.NewMessage(chatID, text)
		b.sendMessage(msg)
		return
	}

	grouped := b.groupOpportunitiesByType(opportunities)

	var text strings.Builder
	text.WriteString("💰 <b>Доступні можливості</b>\n\n")

	total := len(opportunities)
	text.WriteString(fmt.Sprintf("Знайдено: <b>%d</b>\n\n", total))

	for oppType, opps := range grouped {
		if len(opps) == 0 {
			continue
		}

		emoji := b.getTypeEmoji(oppType)
		typeName := b.getTypeName(oppType)

		text.WriteString(fmt.Sprintf("%s <b>%s</b> (%d)\n", emoji, typeName, len(opps)))

		for i, opp := range opps {
			if i >= 3 {
				text.WriteString(fmt.Sprintf("   ... і ще %d\n", len(opps)-3))
				break
			}

			roi := ""
			if opp.EstimatedROI > 0 {
				roi = fmt.Sprintf(" • %.1f%% ROI", opp.EstimatedROI)
			}

			text.WriteString(fmt.Sprintf("   • %s%s\n", b.truncate(opp.Title, 50), roi))
		}
		text.WriteString("\n")
	}

	text.WriteString("👇 Обери категорію для детального перегляду")

	msg := tgbotapi.NewMessage(chatID, text.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildOpportunitiesFilterKeyboard(filter, len(opportunities) > 20, page)

	b.sendMessage(msg)
}

func (b *Bot) groupOpportunitiesByType(opportunities []*models.Opportunity) map[string][]*models.Opportunity {
	result := make(map[string][]*models.Opportunity)

	for _, opp := range opportunities {
		result[opp.Type] = append(result[opp.Type], opp)
	}

	return result
}

func (b *Bot) getTypeEmoji(oppType string) string {
	switch oppType {
	case models.OpportunityTypeLaunchpool:
		return "🚀"
	case models.OpportunityTypeLaunchpad:
		return "🆕"
	case models.OpportunityTypeAirdrop:
		return "🎁"
	case models.OpportunityTypeLearnEarn:
		return "📚"
	case models.OpportunityTypeStaking:
		return "💎"
	case models.OpportunityTypeArbitrage:
		return "🔥"
	case models.OpportunityTypeDeFi:
		return "🌾"
	default:
		return "💰"
	}
}

func (b *Bot) getTypeName(oppType string) string {
	switch oppType {
	case models.OpportunityTypeLaunchpool:
		return "Launchpool"
	case models.OpportunityTypeLaunchpad:
		return "Launchpad"
	case models.OpportunityTypeAirdrop:
		return "Airdrops"
	case models.OpportunityTypeLearnEarn:
		return "Learn & Earn"
	case models.OpportunityTypeStaking:
		return "Staking"
	case models.OpportunityTypeArbitrage:
		return "Арбітраж"
	case models.OpportunityTypeDeFi:
		return "DeFi"
	default:
		return "Інше"
	}
}

func (b *Bot) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
