package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// showSettingsMenu показує головне меню налаштувань
func (b *Bot) showSettingsMenu(chatID int64, user *models.User, prefs *models.UserPreferences) {
	text := b.formatSettingsText(user, prefs)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildSettingsKeyboard()

	b.sendMessage(msg)
}

// formatSettingsText форматує текст з поточними налаштуваннями
func (b *Bot) formatSettingsText(user *models.User, prefs *models.UserPreferences) string {
	var text strings.Builder

	text.WriteString("⚙️ <b>Налаштування</b>\n\n")

	// Subscription tier
	tier := "🆓 Free"
	if user.IsPremium() {
		tier = fmt.Sprintf("💎 Premium (до %s)", user.SubscriptionExpiresAt.Format("02.01.2006"))
	}
	text.WriteString(fmt.Sprintf("Підписка: %s\n\n", tier))

	// Capital range
	capitalDisplay := user.CapitalRange
	if capitalDisplay == "" {
		capitalDisplay = "не вказано"
	}
	text.WriteString(fmt.Sprintf("💰 Капітал: <b>$%s</b>\n", capitalDisplay))

	// Risk profile
	riskEmoji := "🟢"
	riskDisplay := user.RiskProfile
	switch user.RiskProfile {
	case "low":
		riskDisplay = "Низький"
	case "medium":
		riskDisplay = "Середній"
		riskEmoji = "🟡"
	case "high":
		riskDisplay = "Високий"
		riskEmoji = "🔴"
	}
	text.WriteString(fmt.Sprintf("%s Ризик: <b>%s</b>\n\n", riskEmoji, riskDisplay))

	// Opportunity types
	text.WriteString("📋 <b>Типи можливостей:</b>\n")
	if len(prefs.OpportunityTypes) == 0 {
		text.WriteString("  Всі типи\n")
	} else {
		for _, oppType := range prefs.OpportunityTypes {
			text.WriteString(fmt.Sprintf("  • %s\n", b.getTypeName(oppType)))
		}
	}
	text.WriteString("\n")

	// Exchanges
	text.WriteString("🏦 <b>Біржі:</b>\n")
	if len(prefs.Exchanges) == 0 {
		text.WriteString("  Всі біржі\n")
	} else {
		for _, ex := range prefs.Exchanges {
			text.WriteString(fmt.Sprintf("  • %s\n", strings.Title(ex)))
		}
	}
	text.WriteString("\n")

	// Min ROI
	text.WriteString(fmt.Sprintf("📈 Мінімальний ROI: <b>%.1f%%</b>\n", prefs.MinROI))

	// Max Investment
	if prefs.MaxInvestment > 0 {
		text.WriteString(fmt.Sprintf("💵 Макс. інвестиція: <b>$%d</b>\n", prefs.MaxInvestment))
	} else {
		text.WriteString("💵 Макс. інвестиція: <b>без обмежень</b>\n")
	}
	text.WriteString("\n")

	// Notifications
	text.WriteString("🔔 <b>Сповіщення:</b>\n")
	text.WriteString(fmt.Sprintf("  • Миттєві: %s\n", b.formatBool(prefs.NotifyInstant)))
	text.WriteString(fmt.Sprintf("  • Щоденний дайджест: %s\n", b.formatBool(prefs.NotifyDaily)))
	text.WriteString(fmt.Sprintf("  • Щотижневий: %s\n", b.formatBool(prefs.NotifyWeekly)))

	text.WriteString("\n👇 Обери що хочеш змінити")

	return text.String()
}

// buildSettingsKeyboard створює клавіатуру налаштувань
func (b *Bot) buildSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Капітал", "settings_capital"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Ризик", "settings_risk"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Типи", "settings_types"),
			tgbotapi.NewInlineKeyboardButtonData("🏦 Біржі", "settings_exchanges"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 ROI", "settings_roi"),
			tgbotapi.NewInlineKeyboardButtonData("💵 Інвестиції", "settings_investment"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Сповіщення", "settings_notifications"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Головне меню", "menu_main"),
		),
	)
}

// handleSettingsCapital показує вибір капіталу
func (b *Bot) handleSettingsCapital(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	text := "💰 <b>Виберi свій капітал</b>\n\n" +
		"Це допоможе підібрати можливості\n" +
		"відповідно до твоїх можливостей"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.CapitalRange == "100-500")+"$100-500",
				"set_capital_100-500",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.CapitalRange == "500-2000")+"$500-2000",
				"set_capital_500-2000",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.CapitalRange == "2000-5000")+"$2000-5000",
				"set_capital_2000-5000",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.CapitalRange == "5000+")+"$5000+",
				"set_capital_5000+",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад до налаштувань", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// handleSettingsRisk показує вибір ризику
func (b *Bot) handleSettingsRisk(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	text := "📊 <b>Виберi рівень ризику</b>\n\n" +
		"🟢 <b>Низький</b> - стабільні, перевірені проекти\n" +
		"🟡 <b>Середній</b> - баланс між ризиком та прибутком\n" +
		"🔴 <b>Високий</b> - можливість високих прибутків"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.RiskProfile == "low")+"🟢 Низький",
				"set_risk_low",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.RiskProfile == "medium")+"🟡 Середній",
				"set_risk_medium",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(user.RiskProfile == "high")+"🔴 Високий",
				"set_risk_high",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад до налаштувань", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// handleSettingsTypes показує вибір типів можливостей
func (b *Bot) handleSettingsTypes(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	text := "📋 <b>Виберi типи можливостей</b>\n\n" +
		"Обери які типи тебе цікавлять.\n" +
		"Можеш вибрати декілька."

	isSelected := func(oppType string) bool {
		for _, t := range prefs.OpportunityTypes {
			if t == oppType {
				return true
			}
		}
		return false
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected(models.OpportunityTypeLaunchpool))+"🚀 Launchpool",
				"toggle_type_"+models.OpportunityTypeLaunchpool,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected(models.OpportunityTypeAirdrop))+"🎁 Airdrops",
				"toggle_type_"+models.OpportunityTypeAirdrop,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected(models.OpportunityTypeLearnEarn))+"📚 Learn & Earn",
				"toggle_type_"+models.OpportunityTypeLearnEarn,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected(models.OpportunityTypeStaking))+"💎 Staking",
				"toggle_type_"+models.OpportunityTypeStaking,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Зберегти", "save_types"),
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// handleSettingsExchanges показує вибір бірж
func (b *Bot) handleSettingsExchanges(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	text := "🏦 <b>Виберi біржі</b>\n\n" +
		"Обери з яких бірж тебе цікавлять можливості.\n" +
		"Можеш вибрати декілька."

	isSelected := func(exchange string) bool {
		for _, ex := range prefs.Exchanges {
			if ex == exchange {
				return true
			}
		}
		return false
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected("binance"))+"Binance",
				"toggle_exchange_binance",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected("bybit"))+"Bybit",
				"toggle_exchange_bybit",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(isSelected("okx"))+"OKX",
				"toggle_exchange_okx",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Зберегти", "save_exchanges"),
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// handleSettingsROI показує вибір мінімального ROI
func (b *Bot) handleSettingsROI(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	text := fmt.Sprintf(
		"📈 <b>Мінімальний ROI</b>\n\n"+
			"Поточне значення: <b>%.1f%%</b>\n\n"+
			"Виберi мінімальний ROI який тебе цікавить:",
		prefs.MinROI,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MinROI == 0)+"Без обмежень",
				"set_roi_0",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MinROI == 5)+"5%",
				"set_roi_5",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MinROI == 10)+"10%",
				"set_roi_10",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MinROI == 20)+"20%",
				"set_roi_20",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MinROI == 50)+"50%",
				"set_roi_50",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад до налаштувань", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// handleSettingsInvestment показує вибір максимальної інвестиції
func (b *Bot) handleSettingsInvestment(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	currentDisplay := "без обмежень"
	if prefs.MaxInvestment > 0 {
		currentDisplay = fmt.Sprintf("$%d", prefs.MaxInvestment)
	}

	text := fmt.Sprintf(
		"💵 <b>Максимальна інвестиція</b>\n\n"+
			"Поточне значення: <b>%s</b>\n\n"+
			"Фільтрує можливості, які потребують\n"+
			"більше цієї суми для участі:",
		currentDisplay,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MaxInvestment == 0)+"Без обмежень",
				"set_investment_0",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MaxInvestment == 100)+"$100",
				"set_investment_100",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MaxInvestment == 500)+"$500",
				"set_investment_500",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MaxInvestment == 1000)+"$1000",
				"set_investment_1000",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.MaxInvestment == 5000)+"$5000",
				"set_investment_5000",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Назад до налаштувань", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// handleSettingsNotifications показує налаштування сповіщень
func (b *Bot) handleSettingsNotifications(callback *tgbotapi.CallbackQuery) {
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	user, prefs := b.getUserAndPrefs(callback.From.ID)
	if user == nil || prefs == nil {
		return
	}

	text := "🔔 <b>Налаштування сповіщень</b>\n\n" +
		"Обери як хочеш отримувати сповіщення:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.NotifyInstant)+"⚡ Миттєві алерти",
				"toggle_notify_instant",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.NotifyDaily)+"📅 Щоденний дайджест",
				"toggle_notify_daily",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.checkmark(prefs.NotifyWeekly)+"📊 Щотижневий звіт",
				"toggle_notify_weekly",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Зберегти", "save_notifications"),
			tgbotapi.NewInlineKeyboardButtonData("« Назад", "back_settings"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.api.Send(edit)
}

// Helper методи

func (b *Bot) checkmark(isSelected bool) string {
	if isSelected {
		return "✅ "
	}
	return ""
}

func (b *Bot) formatBool(value bool) string {
	if value {
		return "✅ Увімкнено"
	}
	return "❌ Вимкнено"
}
