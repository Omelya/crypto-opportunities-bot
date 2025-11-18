package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleReferral shows user's referral statistics and their referral code
func (b *Bot) handleReferral(message *tgbotapi.Message) {
	user, err := b.userRepo.GetByTelegramID(message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "❌ Error loading your profile. Please use /start first.")
		return
	}

	// Get or create referral code
	referralCode, err := b.referralService.GetUserReferralCode(user.ID)
	if err != nil {
		log.Printf("Error getting referral code: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error loading referral code. Please try again later.")
		return
	}

	// Get referral stats
	stats, err := b.referralService.GetUserReferralStats(user.ID)
	if err != nil {
		log.Printf("Error getting referral stats: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error loading referral statistics.")
		return
	}

	// Format message
	text := fmt.Sprintf(`🎁 *Referral Program*

Your referral code: *%s*

📊 *Your Statistics:*
• Total referrals: %d
• Active referrals: %d
• Rewards earned: %d
• Pending rewards: %d

💰 *How it works:*

*For you (Referrer):*
• Get 1 month Premium FREE for each friend who subscribes
• Unlimited referrals

*For your friend:*
• 20%% discount on first month subscription

*How to invite:*
1. Share your referral link (click button below)
2. Friend registers and subscribes to Premium
3. You both get rewards!

Share your link now 👇`,
		referralCode,
		stats.TotalReferrals,
		stats.ActiveReferrals,
		stats.TotalRewardsEarned,
		stats.PendingRewards,
	)

	// Create inline keyboard with share button
	referralURL := fmt.Sprintf("https://t.me/%s?start=ref_%s", b.botUsername, referralCode)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📤 Share Referral Link", referralURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("💬 Share in Chat", fmt.Sprintf("Join me on Crypto Opportunities Bot! Use my code %s to get 20%% discount: %s", referralCode, referralURL)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Stats", "referral_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending referral message: %v", err)
	}
}

// handleInvite shows quick share options for referral
func (b *Bot) handleInvite(message *tgbotapi.Message) {
	user, err := b.userRepo.GetByTelegramID(message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "❌ Error loading your profile. Please use /start first.")
		return
	}

	// Get or create referral code
	referralCode, err := b.referralService.GetUserReferralCode(user.ID)
	if err != nil {
		log.Printf("Error getting referral code: %v", err)
		b.sendMessage(message.Chat.ID, "❌ Error loading referral code.")
		return
	}

	referralURL := fmt.Sprintf("https://t.me/%s?start=ref_%s", b.botUsername, referralCode)

	text := fmt.Sprintf(`🎁 *Invite Friends & Earn Premium!*

Your referral link:
%s

Your code: *%s*

Share this link with friends and earn 1 month Premium for each friend who subscribes! 🚀

Your friend gets 20%% discount on their first month.`,
		referralURL,
		referralCode,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📤 Open Referral Link", referralURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("💬 Share in Chat", fmt.Sprintf("🎁 Join me on Crypto Opportunities Bot! Get 20%% off with code %s: %s", referralCode, referralURL)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 View Statistics", "referral_stats"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	msg.DisableWebPagePreview = true

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending invite message: %v", err)
	}
}

// handleReferralCallback handles referral-related callback queries
func (b *Bot) handleReferralCallback(callback *tgbotapi.CallbackQuery, action string) {
	switch action {
	case "referral_stats":
		b.handleReferralStatsCallback(callback)
	case "referral_info":
		b.handleReferralInfoCallback(callback)
	default:
		b.answerCallback(callback.ID, "Unknown action")
	}
}

// handleReferralStatsCallback refreshes referral statistics
func (b *Bot) handleReferralStatsCallback(callback *tgbotapi.CallbackQuery) {
	user, err := b.userRepo.GetByTelegramID(callback.From.ID)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Error loading profile")
		return
	}

	referralCode, err := b.referralService.GetUserReferralCode(user.ID)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Error loading referral code")
		return
	}

	stats, err := b.referralService.GetUserReferralStats(user.ID)
	if err != nil {
		b.answerCallback(callback.ID, "❌ Error loading statistics")
		return
	}

	text := fmt.Sprintf(`🎁 *Referral Program*

Your referral code: *%s*

📊 *Your Statistics:*
• Total referrals: %d
• Active referrals: %d
• Rewards earned: %d
• Pending rewards: %d

💰 *How it works:*

*For you (Referrer):*
• Get 1 month Premium FREE for each friend who subscribes
• Unlimited referrals

*For your friend:*
• 20%% discount on first month subscription

Share your link now 👇`,
		referralCode,
		stats.TotalReferrals,
		stats.ActiveReferrals,
		stats.TotalRewardsEarned,
		stats.PendingRewards,
	)

	referralURL := fmt.Sprintf("https://t.me/%s?start=ref_%s", b.botUsername, referralCode)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📤 Share Referral Link", referralURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("💬 Share in Chat", fmt.Sprintf("Join me on Crypto Opportunities Bot! Use my code %s: %s", referralCode, referralURL)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Stats", "referral_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "main_menu"),
		),
	)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Error editing message: %v", err)
	}

	b.answerCallback(callback.ID, "✅ Statistics refreshed")
}

// handleReferralInfoCallback shows referral program information
func (b *Bot) handleReferralInfoCallback(callback *tgbotapi.CallbackQuery) {
	text := `🎁 *Referral Program - How It Works*

*Step 1: Get Your Link*
Use /referral command to get your unique referral link and code.

*Step 2: Invite Friends*
Share your link with friends via social media, messaging apps, or in person.

*Step 3: Earn Rewards*
When your friend subscribes to Premium, you both get rewards:

*Your Reward:*
• 1 month Premium FREE
• Unlimited referrals = unlimited free months!

*Friend's Reward:*
• 20% discount on first month
• Full access to Premium features

*Terms:*
• Friend must be a new user
• Friend must subscribe to Premium
• Rewards are issued automatically
• No limit on number of referrals

Start earning now! 🚀`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 My Referrals", "referral_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("« Back", "premium_menu"),
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
