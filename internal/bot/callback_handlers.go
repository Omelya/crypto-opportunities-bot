package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleMenuCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
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

	switch callback.Data {
	case CallbackMenuToday:
		opportunities, err := b.getFilteredOpportunities(user, prefs, 0)
		if err != nil {
			log.Printf("Error getting opportunities: %v", err)
			b.sendError(chatID)
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.sendOpportunitiesList(chatID, user, opportunities, 0, "all")

	case CallbackMenuAll:
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.sendWelcomeBack(chatID, user)

	case CallbackMenuSettings:
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.showSettingsMenu(chatID, user, prefs)

	case CallbackMenuStats:
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.showStats(chatID, user)

	case CallbackMenuPremium:
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.showPremiumInfo(chatID, user)
	}
}

func (b *Bot) handleFilterCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	var filterType string
	var opportunities []*models.Opportunity

	switch callback.Data {
	case CallbackFilterAll:
		filterType = "all"
		opportunities, err = b.getFilteredOpportunities(user, prefs, 0)

	case CallbackFilterLaunchpool:
		filterType = "launchpool"
		opportunities, err = b.getFilteredOpportunitiesByType(user, prefs, models.OpportunityTypeLaunchpool, 0)

	case CallbackFilterAirdrop:
		filterType = "airdrop"
		opportunities, err = b.getFilteredOpportunitiesByType(user, prefs, models.OpportunityTypeAirdrop, 0)

	case CallbackFilterLearnEarn:
		filterType = "learn_earn"
		opportunities, err = b.getFilteredOpportunitiesByType(user, prefs, models.OpportunityTypeLearnEarn, 0)

	case CallbackFilterStaking:
		filterType = "staking"
		opportunities, err = b.getFilteredOpportunitiesByType(user, prefs, models.OpportunityTypeStaking, 0)

	default:
		return
	}

	if err != nil {
		log.Printf("Error filtering opportunities: %v", err)
		b.sendError(chatID)
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	b.sendOpportunitiesList(chatID, user, opportunities, 0, filterType)
}

func (b *Bot) handlePaginationCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	var page int
	var err2 error

	if strings.HasPrefix(callback.Data, CallbackPageNext) {
		pageStr := strings.TrimPrefix(callback.Data, CallbackPageNext)
		page, err2 = strconv.Atoi(pageStr)
	} else if strings.HasPrefix(callback.Data, CallbackPagePrev) {
		pageStr := strings.TrimPrefix(callback.Data, CallbackPagePrev)
		page, err2 = strconv.Atoi(pageStr)
	}

	if err2 != nil {
		log.Printf("Error parsing page: %v", err2)
		return
	}

	opportunities, err := b.getFilteredOpportunities(user, prefs, page*20)
	if err != nil {
		log.Printf("Error getting opportunities: %v", err)
		b.sendError(chatID)
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	b.sendOpportunitiesList(chatID, user, opportunities, page, "all")
}

func (b *Bot) handleSettingsCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	switch callback.Data {
	case CallbackSettingsCapital:
		b.showCapitalSelection(chatID, user)

	case CallbackSettingsRisk:
		b.showRiskSelection(chatID, user)

	case CallbackSettingsExchanges:
		b.showExchangeSelection(chatID, prefs)

	case CallbackSettingsTypes:
		b.showTypeSelection(chatID, prefs)

	case CallbackSettingsLanguage:
		b.showLanguageSelection(chatID, user)

	case CallbackSettingsDigest:
		b.showDigestSettings(chatID, prefs)
	}
}

func (b *Bot) handleExchangeToggle(callback *tgbotapi.CallbackQuery, exchange string) {
	userID := callback.From.ID
	chatID := callback.Message.Chat.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	found := false
	var newExchanges []string

	for _, ex := range prefs.Exchanges {
		if ex == exchange {
			found = true
		} else {
			newExchanges = append(newExchanges, ex)
		}
	}

	if !found {
		newExchanges = append(newExchanges, exchange)
	}

	prefs.Exchanges = newExchanges
	if prefsErr := b.prefsRepo.Update(prefs); prefsErr != nil {
		log.Printf("Error updating preferences: %v", prefsErr)
		b.sendError(chatID)
		return
	}

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		callback.Message.MessageID,
		b.buildExchangeSelectionKeyboard(prefs.Exchanges),
	)
	b.sendMessage(editMsg)
}

func (b *Bot) handleTypeToggle(callback *tgbotapi.CallbackQuery, oppType string) {
	userID := callback.From.ID
	chatID := callback.Message.Chat.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	found := false
	var newTypes []string

	for _, t := range prefs.OpportunityTypes {
		if t == oppType {
			found = true
		} else {
			newTypes = append(newTypes, t)
		}
	}

	if !found {
		newTypes = append(newTypes, oppType)
	}

	prefs.OpportunityTypes = newTypes
	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences: %v", err)
		b.sendError(chatID)
		return
	}

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		callback.Message.MessageID,
		b.buildTypeSelectionKeyboard(prefs.OpportunityTypes),
	)
	b.sendMessage(editMsg)
}

func (b *Bot) handleDigestToggle(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	chatID := callback.Message.Chat.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	prefs.DailyDigestEnabled = !prefs.DailyDigestEnabled

	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences: %v", err)
		b.sendError(chatID)
		return
	}

	status := "вимкнено"
	if prefs.DailyDigestEnabled {
		status = "ввімкнено"
	}

	text := fmt.Sprintf("✅ Щоденний дайджест %s", status)
	keyboard := b.buildDigestSettingsKeyboard(prefs)

	editMsg := tgbotapi.NewEditMessageText(
		chatID,
		callback.Message.MessageID,
		text,
	)
	editMsg.ReplyMarkup = &keyboard

	b.sendMessage(editMsg)
}

func (b *Bot) showSettingsMenu(chatID int64, user *models.User, prefs *models.UserPreferences) {
	text := "⚙️ <b>Налаштування</b>\n\n"
	text += fmt.Sprintf("💰 Капітал: <b>%s</b>\n", b.formatCapitalRange(user.CapitalRange))
	text += fmt.Sprintf("⚖️ Ризик-профіль: <b>%s</b>\n", b.formatRiskProfile(user.RiskProfile))
	text += fmt.Sprintf("🏦 Біржі: <b>%d обрано</b>\n", len(prefs.Exchanges))
	text += fmt.Sprintf("📊 Типи можливостей: <b>%d обрано</b>\n", len(prefs.OpportunityTypes))
	text += fmt.Sprintf("🌐 Мова: <b>%s</b>\n", b.formatLanguage(user.LanguageCode))
	text += fmt.Sprintf("📬 Щоденний дайджест: <b>%s</b>\n", b.formatBool(prefs.DailyDigestEnabled))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildSettingsKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) showStats(chatID int64, user *models.User) {
	tier := "🆓 Free"
	if user.IsPremium() {
		tier = "💎 Premium"
	}

	text := fmt.Sprintf(
		"📊 <b>Твоя статистика</b>\n\n"+
			"Підписка: %s\n"+
			"Реєстрація: %s\n\n"+
			"🔜 Детальна статистика буде скоро!",
		tier,
		user.CreatedAt.Format("02.01.2006"),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildMainMenuKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) showPremiumInfo(chatID int64, user *models.User) {
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
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Головне меню", CallbackMenuAll),
			),
		)

		b.sendMessage(msg)

		return
	}

	text := `💎 <b>Premium підписка</b>

З Premium ти отримуєш:
⚡ Real-time алерти (0-2 хв затримка)
💰 Арбітражні можливості (10-20/день)
🎯 Персоналізовані фільтри
📊 Детальну аналітику
🔥 DeFi та китові алерти

✨ Перші 7 днів - безкоштовно

Користувачі в середньому заробляють $150-300/міс
завдяки Premium функціям.`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildPremiumKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) showCapitalSelection(chatID int64, user *models.User) {
	text := "💰 <b>Обери свій капітал для інвестицій:</b>\n\n"
	text += "Поточний вибір: " + b.formatCapitalRange(user.CapitalRange)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildCapitalSelectionKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) showRiskSelection(chatID int64, user *models.User) {
	text := "⚖️ <b>Обери свій ризик-профіль:</b>\n\n"
	text += "🟢 <b>Низький</b> - Консервативні інвестиції\n"
	text += "🟡 <b>Середній</b> - Баланс ризику та прибутку\n"
	text += "🔴 <b>Високий</b> - Агресивні стратегії\n\n"
	text += "Поточний вибір: " + b.formatRiskProfile(user.RiskProfile)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildRiskSelectionKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) showExchangeSelection(chatID int64, prefs *models.UserPreferences) {
	text := "🏦 <b>Обери біржі для моніторингу:</b>\n\n"
	text += "Можеш вибрати кілька варіантів:"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildExchangeSelectionKeyboard(prefs.Exchanges)

	b.sendMessage(msg)
}

func (b *Bot) showTypeSelection(chatID int64, prefs *models.UserPreferences) {
	text := "📊 <b>Обери типи можливостей:</b>\n\n"
	text += "Можеш вибрати кілька варіантів:"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildTypeSelectionKeyboard(prefs.OpportunityTypes)

	b.sendMessage(msg)
}

func (b *Bot) showLanguageSelection(chatID int64, user *models.User) {
	text := "🌐 <b>Обери мову інтерфейсу:</b>\n\n"
	text += "Поточна мова: " + b.formatLanguage(user.LanguageCode)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildLanguageSelectionKeyboard()

	b.sendMessage(msg)
}

func (b *Bot) showDigestSettings(chatID int64, prefs *models.UserPreferences) {
	text := "📬 <b>Налаштування щоденного дайджесту</b>\n\n"
	text += fmt.Sprintf("Статус: <b>%s</b>\n", b.formatBool(prefs.DailyDigestEnabled))
	text += fmt.Sprintf("Час відправки: <b>%s</b>", prefs.DailyDigestTime)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildDigestSettingsKeyboard(prefs)

	b.sendMessage(msg)
}

func (b *Bot) getFilteredOpportunitiesByType(user *models.User, prefs *models.UserPreferences, oppType string, offset int) ([]*models.Opportunity, error) {
	limit := 20

	opportunities, err := b.oppRepo.ListByType(oppType, 1000, 0)
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

func (b *Bot) formatCapitalRange(capital string) string {
	if capital == "" {
		return "не вказано"
	}
	return "$" + capital
}

func (b *Bot) formatRiskProfile(risk string) string {
	switch risk {
	case "low":
		return "🟢 Низький"
	case "medium":
		return "🟡 Середній"
	case "high":
		return "🔴 Високий"
	default:
		return "не вказано"
	}
}

func (b *Bot) formatLanguage(lang string) string {
	switch lang {
	case "uk":
		return "🇺🇦 Українська"
	case "en":
		return "🇬🇧 English"
	default:
		return "🇺🇦 Українська"
	}
}

func (b *Bot) formatBool(value bool) string {
	if value {
		return "✅ Ввімкнено"
	}
	return "❌ Вимкнено"
}
