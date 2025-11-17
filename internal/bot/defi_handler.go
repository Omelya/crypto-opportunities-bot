package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleDeFi обробляє команду /defi (тільки для Premium)
func (b *Bot) handleDeFi(message *tgbotapi.Message) {
	user, _ := b.getUserAndPrefs(message.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendDeFiPremiumRequired(message.Chat.ID)
		return
	}

	// Отримати топ DeFi opportunities за APY
	opportunities, err := b.defiRepo.GetTopByAPY(10)
	if err != nil {
		b.sendError(message.Chat.ID)
		return
	}

	if len(opportunities) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"🌾 <b>DeFi Opportunities</b>\n\n"+
				"На даний момент немає активних DeFi можливостей.\n\n"+
				"💡 Моніторинг активний, ви отримаєте алерт коли з'явиться можливість!\n\n"+
				"⏱️ Перевірка відбувається кожні 30 хвилин")
		msg.ParseMode = "HTML"
		b.sendMessage(msg)
		return
	}

	// Форматувати повідомлення
	text := b.formatDeFiList(opportunities)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = getDeFiKeyboard()

	b.sendMessage(msg)
}

// formatDeFiList форматує список DeFi opportunities
func (b *Bot) formatDeFiList(opportunities []*models.DeFiOpportunity) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("🌾 <b>Топ %d DeFi opportunities</b>\n\n", len(opportunities)))

	for i, opp := range opportunities {
		builder.WriteString(formatDeFiOpportunity(opp, i+1))
		builder.WriteString("\n")
	}

	builder.WriteString("⏰ <i>Дані оновлюються кожні 30 хвилин</i>\n")
	builder.WriteString("⚠️ <i>DeFi involves risks. DYOR before investing.</i>")

	return builder.String()
}

// formatDeFiOpportunity форматує одну DeFi opportunity (коротка версія для списку)
func formatDeFiOpportunity(opp *models.DeFiOpportunity, index int) string {
	emoji := "🌾"
	if opp.APY >= 50 {
		emoji = "🔥🌾"
	} else if opp.APY >= 30 {
		emoji = "⭐🌾"
	}

	riskEmoji := "✅"
	switch opp.RiskLevel {
	case "medium":
		riskEmoji = "⚡"
	case "high":
		riskEmoji = "⚠️"
	}

	return fmt.Sprintf(
		"%s <b>%d. %s</b>\n"+
			"├ 🏦 Protocol: <b>%s</b> ⛓️ %s\n"+
			"├ 📈 APY: <b>%.2f%%</b> (%.2f%% base + %.2f%% rewards)\n"+
			"├ 💵 Daily: <b>$%.2f</b> | Monthly: <b>$%.2f</b> (на $1000)\n"+
			"├ 📊 TVL: <b>$%.2fM</b>\n"+
			"├ %s Risk: <b>%s</b> | IL: %.1f%%\n"+
			"└ 💼 Min Deposit: <b>$%.0f</b>\n",
		emoji, index, opp.GetDisplayName(),
		strings.Title(opp.Protocol), strings.Title(opp.Chain),
		opp.APY, opp.APYBase, opp.APYReward,
		opp.DailyReturnUSD(1000), opp.MonthlyReturnUSD(1000),
		opp.TVL/1_000_000,
		riskEmoji, strings.Title(opp.RiskLevel), opp.ILRisk,
		opp.MinDeposit,
	)
}

// getDeFiKeyboard створює клавіатуру для DeFi
func getDeFiKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔥 По APY", "defi_filter_apy"),
			tgbotapi.NewInlineKeyboardButtonData("💎 По TVL", "defi_filter_tvl"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Low Risk", "defi_filter_low"),
			tgbotapi.NewInlineKeyboardButtonData("⚡ Med Risk", "defi_filter_med"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⛓️ За Chain", "defi_filter_chain"),
			tgbotapi.NewInlineKeyboardButtonData("🏦 За Protocol", "defi_filter_protocol"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Оновити", "refresh_defi"),
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Головне меню", CallbackMenuAll),
		),
	)
}

// sendDeFiPremiumRequired відправляє повідомлення про необхідність Premium
func (b *Bot) sendDeFiPremiumRequired(chatID int64) {
	text := "🔒 <b>DeFi Opportunities - Premium функція</b>\n\n" +
		"Моніторинг DeFi можливостей доступний тільки для Premium користувачів.\n\n" +
		"💎 <b>З Premium ви отримаєте:</b>\n" +
		"• Real-time DeFi opportunities з 1000+ протоколів\n" +
		"• Фільтрація за APY, TVL, ризиком, chain\n" +
		"• Автоматичний розрахунок ризиків та IL\n" +
		"• Алерти для пулів з APY 30%+\n" +
		"• Інформація про аудити та безпеку\n" +
		"• Прямі посилання на протоколи\n\n" +
		"📊 DeFiLlama API: 1000+ протоколів, 50+ блокчейнів\n\n" +
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

// handleDeFiRefresh обробляє callback для оновлення DeFi opportunities
func (b *Bot) handleDeFiRefresh(callback *tgbotapi.CallbackQuery) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, "🔄 Оновлюю..."))

	user, _ := b.getUserAndPrefs(callback.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendDeFiPremiumRequired(callback.Message.Chat.ID)
		return
	}

	// Отримати топ DeFi opportunities
	opportunities, err := b.defiRepo.GetTopByAPY(10)
	if err != nil {
		b.sendError(callback.Message.Chat.ID)
		return
	}

	var text string

	if len(opportunities) == 0 {
		text = "🌾 <b>DeFi Opportunities</b>\n\n" +
			"На даний момент немає активних DeFi можливостей.\n\n" +
			"💡 Моніторинг активний, ви отримаєте алерт коли з'явиться можливість!\n\n" +
			"⏱️ Перевірка відбувається кожні 30 хвилин"
	} else {
		text = b.formatDeFiList(opportunities)
	}

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	keyboard := getDeFiKeyboard()
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// handleDeFiFilterByRisk обробляє фільтрацію DeFi за рівнем ризику
func (b *Bot) handleDeFiFilterByRisk(callback *tgbotapi.CallbackQuery, riskLevel string) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Фільтрую за ризиком: %s", riskLevel)))

	user, _ := b.getUserAndPrefs(callback.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendDeFiPremiumRequired(callback.Message.Chat.ID)
		return
	}

	// Отримати DeFi opportunities за рівнем ризику
	opportunities, err := b.defiRepo.GetByRiskLevel(riskLevel, 10)
	if err != nil {
		b.sendError(callback.Message.Chat.ID)
		return
	}

	var text string

	if len(opportunities) == 0 {
		text = fmt.Sprintf("🌾 <b>DeFi Opportunities - %s risk</b>\n\n"+
			"Немає активних можливостей з таким рівнем ризику.\n\n"+
			"Спробуйте інший фільтр або оновіть список.", strings.Title(riskLevel))
	} else {
		text = fmt.Sprintf("🌾 <b>DeFi Opportunities - %s risk</b>\n\n", strings.Title(riskLevel))

		for i, opp := range opportunities {
			text += formatDeFiOpportunity(opp, i+1)
			text += "\n"
		}

		text += "⏰ <i>Дані оновлюються кожні 30 хвилин</i>\n"
		text += "⚠️ <i>DeFi involves risks. DYOR before investing.</i>"
	}

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	keyboard := getDeFiKeyboard()
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// handleDeFiFilterByTVL обробляє фільтрацію DeFi за TVL
func (b *Bot) handleDeFiFilterByTVL(callback *tgbotapi.CallbackQuery) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, "Фільтрую за TVL"))

	user, _ := b.getUserAndPrefs(callback.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendDeFiPremiumRequired(callback.Message.Chat.ID)
		return
	}

	// Отримати топ DeFi opportunities за TVL
	opportunities, err := b.defiRepo.GetTopByTVL(10)
	if err != nil {
		b.sendError(callback.Message.Chat.ID)
		return
	}

	var text string

	if len(opportunities) == 0 {
		text = "🌾 <b>DeFi Opportunities - By TVL</b>\n\n" +
			"Немає активних можливостей.\n\n" +
			"Спробуйте оновити список."
	} else {
		text = "🌾 <b>DeFi Opportunities - Top by TVL</b>\n\n"

		for i, opp := range opportunities {
			text += formatDeFiOpportunity(opp, i+1)
			text += "\n"
		}

		text += "⏰ <i>Дані оновлюються кожні 30 хвилин</i>\n"
		text += "⚠️ <i>DeFi involves risks. DYOR before investing.</i>"
	}

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	keyboard := getDeFiKeyboard()
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// handleDeFiFilterChain показує список chains для вибору
func (b *Bot) handleDeFiFilterChain(callback *tgbotapi.CallbackQuery) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, "Виберіть chain"))

	text := "⛓️ <b>Виберіть blockchain</b>\n\n" +
		"Оберіть chain для фільтрації DeFi можливостей:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ethereum", "defi_chain_ethereum"),
			tgbotapi.NewInlineKeyboardButtonData("BSC", "defi_chain_bsc"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Polygon", "defi_chain_polygon"),
			tgbotapi.NewInlineKeyboardButtonData("Arbitrum", "defi_chain_arbitrum"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Optimism", "defi_chain_optimism"),
			tgbotapi.NewInlineKeyboardButtonData("Avalanche", "defi_chain_avalanche"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "refresh_defi"),
		),
	)

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// handleDeFiByChain обробляє фільтрацію за конкретним chain
func (b *Bot) handleDeFiByChain(callback *tgbotapi.CallbackQuery, chain string) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Фільтрую за chain: %s", chain)))

	user, _ := b.getUserAndPrefs(callback.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendDeFiPremiumRequired(callback.Message.Chat.ID)
		return
	}

	// Отримати DeFi opportunities за chain
	opportunities, err := b.defiRepo.GetByChain(chain, 10)
	if err != nil {
		b.sendError(callback.Message.Chat.ID)
		return
	}

	var text string

	if len(opportunities) == 0 {
		text = fmt.Sprintf("🌾 <b>DeFi Opportunities - %s</b>\n\n"+
			"Немає активних можливостей на цьому chain.\n\n"+
			"Спробуйте інший chain або оновіть список.", strings.Title(chain))
	} else {
		text = fmt.Sprintf("🌾 <b>DeFi Opportunities - %s</b>\n\n", strings.Title(chain))

		for i, opp := range opportunities {
			text += formatDeFiOpportunity(opp, i+1)
			text += "\n"
		}

		text += "⏰ <i>Дані оновлюються кожні 30 хвилин</i>\n"
		text += "⚠️ <i>DeFi involves risks. DYOR before investing.</i>"
	}

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	keyboard := getDeFiKeyboard()
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// handleDeFiFilterProtocol показує список протоколів для вибору
func (b *Bot) handleDeFiFilterProtocol(callback *tgbotapi.CallbackQuery) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, "Виберіть protocol"))

	text := "🏦 <b>Виберіть DeFi protocol</b>\n\n" +
		"Оберіть protocol для фільтрації можливостей:"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Uniswap", "defi_protocol_uniswap"),
			tgbotapi.NewInlineKeyboardButtonData("Aave", "defi_protocol_aave"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Curve", "defi_protocol_curve"),
			tgbotapi.NewInlineKeyboardButtonData("Compound", "defi_protocol_compound"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("PancakeSwap", "defi_protocol_pancakeswap"),
			tgbotapi.NewInlineKeyboardButtonData("Balancer", "defi_protocol_balancer"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "refresh_defi"),
		),
	)

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}

// handleDeFiByProtocol обробляє фільтрацію за конкретним protocol
func (b *Bot) handleDeFiByProtocol(callback *tgbotapi.CallbackQuery, protocol string) {
	// Answer callback
	b.sendMessage(tgbotapi.NewCallback(callback.ID, fmt.Sprintf("Фільтрую за protocol: %s", protocol)))

	user, _ := b.getUserAndPrefs(callback.From.ID)

	// Premium only
	if user == nil || !user.IsPremium() {
		b.sendDeFiPremiumRequired(callback.Message.Chat.ID)
		return
	}

	// Отримати DeFi opportunities за protocol
	opportunities, err := b.defiRepo.GetByProtocol(protocol, 10)
	if err != nil {
		b.sendError(callback.Message.Chat.ID)
		return
	}

	var text string

	if len(opportunities) == 0 {
		text = fmt.Sprintf("🌾 <b>DeFi Opportunities - %s</b>\n\n"+
			"Немає активних можливостей у цьому протоколі.\n\n"+
			"Спробуйте інший protocol або оновіть список.", strings.Title(protocol))
	} else {
		text = fmt.Sprintf("🌾 <b>DeFi Opportunities - %s</b>\n\n", strings.Title(protocol))

		for i, opp := range opportunities {
			text += formatDeFiOpportunity(opp, i+1)
			text += "\n"
		}

		text += "⏰ <i>Дані оновлюються кожні 30 хвилин</i>\n"
		text += "⚠️ <i>DeFi involves risks. DYOR before investing.</i>"
	}

	// Update message
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "HTML"
	keyboard := getDeFiKeyboard()
	edit.ReplyMarkup = &keyboard

	b.sendMessage(edit)
}
