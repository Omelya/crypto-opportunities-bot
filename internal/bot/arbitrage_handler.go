package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleArbitrage обробляє команду /arbitrage (тільки для Premium)
func (b *Bot) handleArbitrage(message *tgbotapi.Message) {
	user, _ := b.getUserAndPrefs(message.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendPremiumRequired(message.Chat.ID)

		return
	}

	// Отримати активні арбітражні можливості
	opportunities, err := b.arbRepo.GetActive(5)
	if err != nil {
		b.sendError(message.Chat.ID)

		return
	}

	if len(opportunities) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🔍 <b>Арбітражні можливості</b>\n\n"+
				"На даний момент немає прибуткових арбітражних можливостей.\n\n"+
				"💡 Моніторинг активний, ви отримаєте алерт коли з'явиться можливість!\n\n"+
				"⏱️ Перевірка відбувається кожні 1-2 хвилини")

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackMenuAll),
			),
		)

		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard

		b.sendMessage(msg)
		return
	}

	// Форматувати повідомлення
	text := fmt.Sprintf("🔥 <b>Топ %d арбітражних можливостей</b>\n\n", len(opportunities))
	for i, opp := range opportunities {
		text += formatArbitrageOpportunity(opp, i+1)
		text += "\n"
	}

	text += "⏰ <i>Актуально: ~3-5 хвилин</i>\n"
	text += "⚠️ <i>Це інформація, не гарантія прибутку. Ціни змінюються швидко.</i>"

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = getArbitrageKeyboard()

	b.sendMessage(msg)
}

// handleArbitrageRefresh обробляє callback для оновлення арбітражних можливостей
func (b *Bot) handleArbitrageRefresh(callback *tgbotapi.CallbackQuery) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, "🔄 Оновлюю..."))

	user, _ := b.getUserAndPrefs(callback.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendPremiumRequired(callback.Message.Chat.ID)

		return
	}

	// Отримати активні арбітражні можливості
	opportunities, err := b.arbRepo.GetActive(5)
	if err != nil {
		b.sendError(callback.Message.Chat.ID)

		return
	}

	var text string
	if len(opportunities) == 0 {
		text = "🔍 <b>Арбітражні можливості</b>\n\n" +
			"На даний момент немає прибуткових арбітражних можливостей.\n\n" +
			"💡 Моніторинг активний, ви отримаєте алерт коли з'явиться можливість!\n\n" +
			"⏱️ Перевірка відбувається кожні 1-2 хвилини"
	} else {
		text = fmt.Sprintf("🔥 <b>Топ %d арбітражних можливостей</b>\n\n", len(opportunities))

		for i, opp := range opportunities {
			text += formatArbitrageOpportunity(opp, i+1)
			text += "\n"
		}

		text += "⏰ <i>Актуально: ~3-5 хвилин</i>\n"
		text += "⚠️ <i>Це інформація, не гарантія прибутку. Ціни змінюються швидко.</i>"
	}

	// Update message
	keyboard := getArbitrageKeyboard()
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// formatArbitrageOpportunity форматує одну арбітражну можливість
func formatArbitrageOpportunity(opp *models.ArbitrageOpportunity, index int) string {
	emoji := "💰"
	if opp.NetProfitPercent >= 1.0 {
		emoji = "🔥🔥"
	} else if opp.NetProfitPercent >= 0.5 {
		emoji = "🔥"
	}

	// Capitalize exchange names
	buyExchangeCap := strings.ToUpper(string(opp.ExchangeBuy[0])) + opp.ExchangeBuy[1:]
	sellExchangeCap := strings.ToUpper(string(opp.ExchangeSell[0])) + opp.ExchangeSell[1:]

	timeLeft := opp.TimeLeft()
	minutesLeft := int(timeLeft.Minutes())
	if minutesLeft < 0 {
		minutesLeft = 0
	}

	return fmt.Sprintf(
		"%s <b>%d. %s</b>\n"+
			"├ 🟢 Купити: <b>%s</b> @ <code>$%.2f</code>\n"+
			"├ 🔴 Продати: <b>%s</b> @ <code>$%.2f</code>\n"+
			"├ 💵 Валовий profit: <b>%.2f%%</b>\n"+
			"├ 💸 На $1000: <b>$%.2f</b>\n"+
			"├ 📊 Рекомендовано: <b>$%.0f-%.0f</b>\n"+
			"├ ⚠️ Fees: -%.2f%% (trading + withdrawal)\n"+
			"├ 📉 Slippage: -%.2f%% (buy+sell)\n"+
			"├ ✅ Чистий profit: <b>%.2f%%</b> (<b>$%.2f</b> на $1000)\n"+
			"└ ⏰ Залишилось: ~%d хв\n",
		emoji, index, opp.Pair,
		buyExchangeCap, opp.PriceBuy,
		sellExchangeCap, opp.PriceSell,
		opp.ProfitPercent,
		opp.ProfitUSD,
		opp.MinTradeAmount, opp.RecommendedAmount,
		opp.TotalFeesPercent,
		opp.SlippageBuy+opp.SlippageSell,
		opp.NetProfitPercent, opp.NetProfitUSD,
		minutesLeft,
	)
}

// getArbitrageKeyboard створює клавіатуру для арбітражу
func getArbitrageKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Оновити", CallbackRefreshArbitrage),
			tgbotapi.NewInlineKeyboardButtonData("📊 Головне меню", CallbackMenuAll),
		),
	)
}

// sendPremiumRequired відправляє повідомлення про необхідність Premium
func (b *Bot) sendPremiumRequired(chatID int64) {
	text := "🔒 <b>Арбітраж - Premium функція</b>\n\n" +
		"Моніторинг арбітражних можливостей доступний тільки для Premium користувачів.\n\n" +
		"💎 <b>З Premium ви отримаєте:</b>\n" +
		"• Real-time арбітражні алерти (0-2 хв затримка)\n" +
		"• Точний розрахунок profit з fees та slippage\n" +
		"• Рекомендації по обсягу торгівлі\n" +
		"• Моніторинг 15-20 пар на 3+ біржах\n" +
		"• DeFi можливості та китові алерти\n\n" +
		"💰 Користувачі в середньому заробляють $150-300/міс завдяки арбітражу\n\n" +
		"⚡ Спробуйте Premium зараз!"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💎 Переглянути Premium", CallbackMenuPremium),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackMenuAll),
		),
	)

	msg.ReplyMarkup = keyboard

	b.sendMessage(msg)
}
