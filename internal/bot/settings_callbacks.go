package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleSetCapital встановлює капітал користувача
func (b *Bot) handleSetCapital(callback *tgbotapi.CallbackQuery, capitalRange string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	user.CapitalRange = capitalRange
	if err := b.userRepo.Update(user); err != nil {
		log.Printf("Error updating user capital: %v", err)
		b.api.Send(tgbotapi.NewCallback(callback.ID, "❌ Помилка"))
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleSetRisk встановлює ризик-профіль користувача
func (b *Bot) handleSetRisk(callback *tgbotapi.CallbackQuery, riskProfile string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	user.RiskProfile = riskProfile
	if err := b.userRepo.Update(user); err != nil {
		log.Printf("Error updating user risk: %v", err)
		b.api.Send(tgbotapi.NewCallback(callback.ID, "❌ Помилка"))
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleToggleType перемикає вибір типу можливості
func (b *Bot) handleToggleType(callback *tgbotapi.CallbackQuery, oppType string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	// Toggle selection
	found := false
	newTypes := []string{}
	for _, t := range prefs.OpportunityTypes {
		if t == oppType {
			found = true
			// Skip this type (remove it)
			continue
		}
		newTypes = append(newTypes, t)
	}

	if !found {
		// Add the type
		newTypes = append(newTypes, oppType)
	}

	prefs.OpportunityTypes = newTypes

	// Update immediately for toggle
	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences types: %v", err)
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	// Refresh types menu with updated state
	b.handleSettingsTypes(callback)
}

// handleToggleExchange перемикає вибір біржі
func (b *Bot) handleToggleExchange(callback *tgbotapi.CallbackQuery, exchange string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	// Toggle selection
	found := false
	newExchanges := []string{}
	for _, ex := range prefs.Exchanges {
		if ex == exchange {
			found = true
			// Skip this exchange (remove it)
			continue
		}
		newExchanges = append(newExchanges, ex)
	}

	if !found {
		// Add the exchange
		newExchanges = append(newExchanges, exchange)
	}

	prefs.Exchanges = newExchanges

	// Update immediately for toggle
	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences exchanges: %v", err)
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	// Refresh exchanges menu with updated state
	b.handleSettingsExchanges(callback)
}

// handleSetROI встановлює мінімальний ROI
func (b *Bot) handleSetROI(callback *tgbotapi.CallbackQuery, roiStr string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	roi, err := strconv.ParseFloat(roiStr, 64)
	if err != nil {
		log.Printf("Error parsing ROI: %v", err)
		b.api.Send(tgbotapi.NewCallback(callback.ID, "❌ Помилка"))
		return
	}

	prefs.MinROI = roi
	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences ROI: %v", err)
		b.api.Send(tgbotapi.NewCallback(callback.ID, "❌ Помилка"))
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleSetInvestment встановлює максимальну інвестицію
func (b *Bot) handleSetInvestment(callback *tgbotapi.CallbackQuery, amountStr string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		log.Printf("Error parsing investment amount: %v", err)
		b.api.Send(tgbotapi.NewCallback(callback.ID, "❌ Помилка"))
		return
	}

	prefs.MaxInvestment = amount
	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences investment: %v", err)
		b.api.Send(tgbotapi.NewCallback(callback.ID, "❌ Помилка"))
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleToggleNotification перемикає сповіщення
func (b *Bot) handleToggleNotification(callback *tgbotapi.CallbackQuery, notifyType string) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	switch notifyType {
	case "instant":
		prefs.NotifyInstant = !prefs.NotifyInstant
	case "daily":
		prefs.NotifyDaily = !prefs.NotifyDaily
	case "weekly":
		prefs.NotifyWeekly = !prefs.NotifyWeekly
	}

	// Update immediately for toggle
	if err := b.prefsRepo.Update(prefs); err != nil {
		log.Printf("Error updating preferences notifications: %v", err)
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	// Refresh notifications menu with updated state
	b.handleSettingsNotifications(callback)
}

// handleSaveTypes зберігає обрані типи (вже збережено при toggle, просто повертаємо до налаштувань)
func (b *Bot) handleSaveTypes(callback *tgbotapi.CallbackQuery) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleSaveExchanges зберігає обрані біржі
func (b *Bot) handleSaveExchanges(callback *tgbotapi.CallbackQuery) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleSaveNotifications зберігає налаштування сповіщень
func (b *Bot) handleSaveNotifications(callback *tgbotapi.CallbackQuery) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, "✅ Збережено"))

	// Refresh settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// handleBackSettings повертає до головного меню налаштувань
func (b *Bot) handleBackSettings(callback *tgbotapi.CallbackQuery) {
	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	// Show settings menu
	text := b.formatSettingsText(user, prefs)
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &b.buildSettingsKeyboard()

	b.api.Send(edit)
}

// Routing для всіх settings callbacks
func (b *Bot) handleSettingsCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Навігація по меню settings
	switch data {
	case "settings_capital":
		b.handleSettingsCapital(callback)
		return
	case "settings_risk":
		b.handleSettingsRisk(callback)
		return
	case "settings_types":
		b.handleSettingsTypes(callback)
		return
	case "settings_exchanges":
		b.handleSettingsExchanges(callback)
		return
	case "settings_roi":
		b.handleSettingsROI(callback)
		return
	case "settings_investment":
		b.handleSettingsInvestment(callback)
		return
	case "settings_notifications":
		b.handleSettingsNotifications(callback)
		return
	case "back_settings":
		b.handleBackSettings(callback)
		return
	case "save_types":
		b.handleSaveTypes(callback)
		return
	case "save_exchanges":
		b.handleSaveExchanges(callback)
		return
	case "save_notifications":
		b.handleSaveNotifications(callback)
		return
	}

	// Set capital
	if strings.HasPrefix(data, "set_capital_") {
		capitalRange := strings.TrimPrefix(data, "set_capital_")
		b.handleSetCapital(callback, capitalRange)
		return
	}

	// Set risk
	if strings.HasPrefix(data, "set_risk_") {
		riskProfile := strings.TrimPrefix(data, "set_risk_")
		b.handleSetRisk(callback, riskProfile)
		return
	}

	// Toggle opportunity type
	if strings.HasPrefix(data, "toggle_type_") {
		oppType := strings.TrimPrefix(data, "toggle_type_")
		b.handleToggleType(callback, oppType)
		return
	}

	// Toggle exchange
	if strings.HasPrefix(data, "toggle_exchange_") {
		exchange := strings.TrimPrefix(data, "toggle_exchange_")
		b.handleToggleExchange(callback, exchange)
		return
	}

	// Set ROI
	if strings.HasPrefix(data, "set_roi_") {
		roiStr := strings.TrimPrefix(data, "set_roi_")
		b.handleSetROI(callback, roiStr)
		return
	}

	// Set investment
	if strings.HasPrefix(data, "set_investment_") {
		amountStr := strings.TrimPrefix(data, "set_investment_")
		b.handleSetInvestment(callback, amountStr)
		return
	}

	// Toggle notifications
	if strings.HasPrefix(data, "toggle_notify_") {
		notifyType := strings.TrimPrefix(data, "toggle_notify_")
		b.handleToggleNotification(callback, notifyType)
		return
	}
}
