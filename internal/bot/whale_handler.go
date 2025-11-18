package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleWhales shows recent whale transactions (Premium feature)
func (b *Bot) handleWhales(message *tgbotapi.Message) {
	user, err := b.userRepo.GetByTelegramID(message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "❌ Error loading your profile. Please use /start first.")
		return
	}

	// Check if user is Premium
	if !user.IsPremium() {
		text := `🐋 *Whale Watching - Premium Feature*

Track large cryptocurrency transactions in real-time!

*What you get:*
• 🐋 Whale alerts for transactions >$1M
• 📊 Direction analysis (Accumulation vs Distribution)
• ⛓️ Multi-chain support (Ethereum, BSC, Polygon)
• 📈 Historical outcome analysis
• ⚡ Real-time notifications

*Example alert:*
_"🐋 LARGE WHALE: 5,000 ETH ($12M) moved from Binance to unknown wallet - Potential accumulation signal!"_

💎 Upgrade to Premium to access Whale Watching!`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💎 Get Premium", "menu_premium"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("« Back", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
		}
		return
	}

	// Premium user - show whale transactions
	b.showWhaleTransactions(message.Chat.ID, user.ID, "all", 10)
}

// showWhaleTransactions displays whale transactions with filters
func (b *Bot) showWhaleTransactions(chatID int64, userID uint, filter string, limit int) {
	var whales interface{}
	var err error

	// Get whale transactions based on filter
	switch filter {
	case "ethereum":
		whales, err = b.whaleRepo.GetRecentByChain("ethereum", limit)
	case "bsc":
		whales, err = b.whaleRepo.GetRecentByChain("bsc", limit)
	case "accumulation":
		whales, err = b.whaleRepo.GetByDirection("exchange_to_wallet", limit)
	case "distribution":
		whales, err = b.whaleRepo.GetByDirection("wallet_to_exchange", limit)
	default:
		whales, err = b.whaleRepo.GetRecent(limit)
	}

	if err != nil {
		log.Printf("Error getting whale transactions: %v", err)
		b.sendMessage(chatID, "❌ Error loading whale transactions.")
		return
	}

	// Get count
	count24h, _ := b.whaleRepo.CountLast24h()

	// Format message
	text := fmt.Sprintf(`🐋 *Whale Watching*

📊 *Last 24 hours:* %d whale transactions

Recent whale movements:

`, count24h)

	whaleList := whales.([]*models.WhaleTransaction)
	if len(whaleList) == 0 {
		text += "No recent whale transactions found.\n\n_Whale monitoring is active. You'll be notified when large movements occur._"
	} else {
		for i, whale := range whaleList {
			if i >= 5 { // Show max 5 in list
				break
			}

			text += fmt.Sprintf("%s *%.0f %s* ($%.2fM)\n",
				whale.GetDirectionEmoji(),
				whale.Amount,
				whale.Token,
				whale.AmountUSD/1000000,
			)

			if whale.FromLabel != "" {
				text += fmt.Sprintf("   From: %s\n", whale.FromLabel)
			}
			if whale.ToLabel != "" {
				text += fmt.Sprintf("   To: %s\n", whale.ToLabel)
			}

			text += fmt.Sprintf("   %s • %s\n\n", whale.GetSignalInterpretation(), whale.GetTimeAgo())
		}
	}

	// Create filter keyboard
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⛓️ Ethereum", "whale_filter_ethereum"),
			tgbotapi.NewInlineKeyboardButtonData("🟡 BSC", "whale_filter_bsc"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 Accumulation", "whale_filter_accumulation"),
			tgbotapi.NewInlineKeyboardButtonData("📤 Distribution", "whale_filter_distribution"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "whale_refresh"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Stats", "whale_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	msg.DisableWebPagePreview = true

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending whale message: %v", err)
	}
}

// handleWhaleCallback handles whale-related callback queries
func (b *Bot) handleWhaleCallback(callback *tgbotapi.CallbackQuery, action string) {
	user, err := b.userRepo.GetByTelegramID(callback.From.ID)
	if err != nil || !user.IsPremium() {
		b.answerCallback(callback.ID, "❌ Premium feature only")
		return
	}

	switch action {
	case "whale_refresh":
		b.handleWhaleRefresh(callback)
	case "whale_stats":
		b.handleWhaleStats(callback)
	case "whale_filter_ethereum":
		b.handleWhaleFilter(callback, "ethereum")
	case "whale_filter_bsc":
		b.handleWhaleFilter(callback, "bsc")
	case "whale_filter_accumulation":
		b.handleWhaleFilter(callback, "accumulation")
	case "whale_filter_distribution":
		b.handleWhaleFilter(callback, "distribution")
	default:
		b.answerCallback(callback.ID, "Unknown action")
	}
}

// handleWhaleRefresh refreshes whale transaction list
func (b *Bot) handleWhaleRefresh(callback *tgbotapi.CallbackQuery) {
	user, _ := b.userRepo.GetByTelegramID(callback.From.ID)

	whales, err := b.whaleRepo.GetRecent(10)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Error loading whales")
		return
	}

	count24h, _ := b.whaleRepo.CountLast24h()

	text := fmt.Sprintf(`🐋 *Whale Watching*

📊 *Last 24 hours:* %d whale transactions

Recent whale movements:

`, count24h)

	if len(whales) == 0 {
		text += "No recent whale transactions.\n\n_Monitoring active..._"
	} else {
		for i, whale := range whales {
			if i >= 5 {
				break
			}

			text += fmt.Sprintf("%s *%.0f %s* ($%.2fM)\n   %s • %s\n\n",
				whale.GetDirectionEmoji(),
				whale.Amount,
				whale.Token,
				whale.AmountUSD/1000000,
				whale.GetSignalInterpretation(),
				whale.GetTimeAgo(),
			)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⛓️ Ethereum", "whale_filter_ethereum"),
			tgbotapi.NewInlineKeyboardButtonData("🟡 BSC", "whale_filter_bsc"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 Accumulation", "whale_filter_accumulation"),
			tgbotapi.NewInlineKeyboardButtonData("📤 Distribution", "whale_filter_distribution"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "whale_refresh"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Stats", "whale_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "main_menu"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	edit.DisableWebPagePreview = true

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Error editing message: %v", err)
	}

	b.answerCallback(callback.ID, "✅ Refreshed")
}

// handleWhaleStats shows whale statistics
func (b *Bot) handleWhaleStats(callback *tgbotapi.CallbackQuery) {
	stats, err := b.whaleRepo.GetStats24h("", "")
	if err != nil {
		b.answerCallback(callback.ID, "❌ Error loading stats")
		return
	}

	topTokens, _ := b.whaleRepo.GetTopTokens24h(5)

	text := fmt.Sprintf(`📊 *Whale Statistics (24h)*

📈 *Total Transactions:* %d
💰 *Total Volume:* $%.2fM
📊 *Average Size:* $%.2fM
🐋 *Largest Transaction:* $%.2fM

*Movement Analysis:*
📥 Accumulation: %d transactions
📤 Distribution: %d transactions
💵 Net Flow: $%.2fM

*Top Tokens:*
`,
		stats.Last24hCount,
		stats.Last24hVolume/1000000,
		stats.AverageTxSize/1000000,
		stats.LargestTx/1000000,
		stats.AccumulationCount,
		stats.DistributionCount,
		stats.NetFlow/1000000,
	)

	for i, token := range topTokens {
		text += fmt.Sprintf("%d. %s\n", i+1, token)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back to Whales", "whale_refresh"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Error editing message: %v", err)
	}

	b.answerCallback(callback.ID, "")
}

// handleWhaleFilter filters whale transactions
func (b *Bot) handleWhaleFilter(callback *tgbotapi.CallbackQuery, filter string) {
	user, _ := b.userRepo.GetByTelegramID(callback.From.ID)

	var whales []*models.WhaleTransaction
	var err error

	switch filter {
	case "ethereum":
		whales, err = b.whaleRepo.GetRecentByChain("ethereum", 10)
	case "bsc":
		whales, err = b.whaleRepo.GetRecentByChain("bsc", 10)
	case "accumulation":
		whales, err = b.whaleRepo.GetByDirection("exchange_to_wallet", 10)
	case "distribution":
		whales, err = b.whaleRepo.GetByDirection("wallet_to_exchange", 10)
	}

	if err != nil {
		b.answerCallback(callback.ID, "❌ Error loading whales")
		return
	}

	filterName := map[string]string{
		"ethereum":     "Ethereum",
		"bsc":          "BSC",
		"accumulation": "Accumulation",
		"distribution": "Distribution",
	}[filter]

	text := fmt.Sprintf(`🐋 *Whale Watching - %s*

Recent movements:

`, filterName)

	if len(whales) == 0 {
		text += fmt.Sprintf("No %s whale transactions in last 24h.", filterName)
	} else {
		for i, whale := range whales {
			if i >= 5 {
				break
			}

			text += fmt.Sprintf("%s *%.0f %s* ($%.2fM)\n   %s • %s\n\n",
				whale.GetDirectionEmoji(),
				whale.Amount,
				whale.Token,
				whale.AmountUSD/1000000,
				whale.GetSignalInterpretation(),
				whale.GetTimeAgo(),
			)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⛓️ Ethereum", "whale_filter_ethereum"),
			tgbotapi.NewInlineKeyboardButtonData("🟡 BSC", "whale_filter_bsc"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 Accumulation", "whale_filter_accumulation"),
			tgbotapi.NewInlineKeyboardButtonData("📤 Distribution", "whale_filter_distribution"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 All", "whale_refresh"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Stats", "whale_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "main_menu"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	edit.DisableWebPagePreview = true

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Error editing message: %v", err)
	}

	b.answerCallback(callback.ID, fmt.Sprintf("✅ Filtered by %s", filterName))
}
