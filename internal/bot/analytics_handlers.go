package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// trackOpportunityView tracks when user views an opportunity
func (b *Bot) trackOpportunityView(userID uint, opportunityID uint) {
	if b.analyticsService == nil {
		return
	}

	if err := b.analyticsService.TrackAction(userID, models.ActionTypeViewed, &opportunityID, nil); err != nil {
		log.Printf("Error tracking opportunity view: %v", err)
	}
}

// trackOpportunityClick tracks when user clicks on opportunity link
func (b *Bot) trackOpportunityClick(userID uint, opportunityID uint) {
	if b.analyticsService == nil {
		return
	}

	if err := b.analyticsService.TrackAction(userID, models.ActionTypeClicked, &opportunityID, nil); err != nil {
		log.Printf("Error tracking opportunity click: %v", err)
	}
}

// trackCommand tracks command usage
func (b *Bot) trackCommand(userID uint, command string) {
	if b.analyticsService == nil {
		return
	}

	metadata := map[string]interface{}{
		"command": command,
	}

	if err := b.analyticsService.TrackAction(userID, "command_used", nil, metadata); err != nil {
		log.Printf("Error tracking command: %v", err)
	}
}

// handleMyStats shows detailed personal statistics
func (b *Bot) handleMyStats(message *tgbotapi.Message) {
	user, err := b.userRepo.GetByTelegramID(message.From.ID)
	if err != nil || user == nil {
		b.sendError(message.Chat.ID)
		return
	}

	// Track command usage
	b.trackCommand(user.ID, "mystats")

	if b.analyticsService == nil {
		b.handleStats(message) // Fallback to basic stats
		return
	}

	analytics, err := b.analyticsService.GetUserAnalytics(user.ID)
	if err != nil {
		log.Printf("Error getting user analytics: %v", err)
		b.sendError(message.Chat.ID)
		return
	}

	// Get engagement history
	engagements, err := b.analyticsService.GetUserEngagementHistory(user.ID, 7)
	if err != nil {
		log.Printf("Error getting engagement history: %v", err)
	}

	text := b.formatUserStats(user, analytics, engagements)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildStatsKeyboard(user)

	b.sendMessage(msg)
}

// handleAnalytics shows platform-wide analytics (admin only)
func (b *Bot) handleAnalytics(message *tgbotapi.Message) {
	user, err := b.userRepo.GetByTelegramID(message.From.ID)
	if err != nil || user == nil {
		b.sendError(message.Chat.ID)
		return
	}

	// Check if user is admin
	if !b.isAdmin(user) {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Ця команда доступна тільки адміністраторам")
		b.sendMessage(msg)
		return
	}

	if b.analyticsService == nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Analytics service not available")
		b.sendMessage(msg)
		return
	}

	summary, err := b.analyticsService.GetPlatformSummary()
	if err != nil {
		log.Printf("Error getting platform summary: %v", err)
		b.sendError(message.Chat.ID)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, summary.FormatSummary())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildAnalyticsKeyboard()

	b.sendMessage(msg)
}

// handleTopOpportunities shows top performing opportunities
func (b *Bot) handleTopOpportunities(callback *tgbotapi.CallbackQuery) {
	if b.analyticsService == nil {
		return
	}

	topOpps, err := b.analyticsService.GetTopOpportunities(10)
	if err != nil {
		log.Printf("Error getting top opportunities: %v", err)
		return
	}

	var text strings.Builder
	text.WriteString("🏆 <b>Top Performing Opportunities</b>\n\n")

	for i, stats := range topOpps {
		if stats.Opportunity.ID == 0 {
			continue
		}

		text.WriteString(fmt.Sprintf(
			"%d. <b>%s</b>\n"+
				"   📊 Score: %.1f/100\n"+
				"   👁 Views: %d | 🖱 Clicks: %d | ✅ Participations: %d\n"+
				"   📈 Conversion: %.2f%%\n\n",
			i+1,
			stats.Opportunity.Title,
			stats.PerformanceScore,
			stats.UniqueViews,
			stats.UniqueClicks,
			stats.UniqueParticipations,
			stats.OverallConversionRate,
		))
	}

	edit := tgbotapi.NewEditMessageText(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		text.String(),
	)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = b.buildAnalyticsKeyboard()

	b.sendMessage(edit)
	b.sendMessage(tgbotapi.NewCallback(callback.ID, ""))
}

// handleTopUsers shows top users by engagement
func (b *Bot) handleTopUsers(callback *tgbotapi.CallbackQuery) {
	if b.analyticsService == nil {
		return
	}

	topUsers, err := b.analyticsService.GetTopUsers(10, "participated")
	if err != nil {
		log.Printf("Error getting top users: %v", err)
		return
	}

	var text strings.Builder
	text.WriteString("👥 <b>Top Active Users</b>\n\n")

	for i, analytics := range topUsers {
		text.WriteString(fmt.Sprintf(
			"%d. User #%d\n"+
				"   🎯 Participations: %d\n"+
				"   👁 Views: %d | 🖱 Clicks: %d\n"+
				"   📈 Conversion: %.2f%%\n"+
				"   ⏱ Sessions: %d (avg: %ds)\n\n",
			i+1,
			analytics.UserID,
			analytics.ParticipatedOpportunities,
			analytics.ViewedOpportunities,
			analytics.ClickedOpportunities,
			analytics.OverallConversionRate,
			analytics.TotalSessions,
			analytics.AverageSessionTime,
		))
	}

	edit := tgbotapi.NewEditMessageText(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		text.String(),
	)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = b.buildAnalyticsKeyboard()

	b.sendMessage(edit)
	b.sendMessage(tgbotapi.NewCallback(callback.ID, ""))
}

// handleDailyReport shows daily analytics report
func (b *Bot) handleDailyReport(callback *tgbotapi.CallbackQuery) {
	if b.analyticsService == nil {
		return
	}

	// Get last 7 days
	to := time.Now()
	from := to.AddDate(0, 0, -7)

	stats, err := b.analyticsService.GetDailyStatsRange(from, to)
	if err != nil {
		log.Printf("Error getting daily stats: %v", err)
		return
	}

	var text strings.Builder
	text.WriteString("📅 <b>Daily Report (Last 7 Days)</b>\n\n")

	for _, stat := range stats {
		if stat == nil {
			continue
		}

		text.WriteString(fmt.Sprintf(
			"<b>%s</b>\n"+
				"• Users: %d active, %d new\n"+
				"• Opportunities: %d viewed, %d clicked\n"+
				"• Conversion: %.2f%% | Revenue: $%.2f\n\n",
			stat.Date.Format("Jan 02"),
			stat.ActiveUsers,
			stat.NewUsers,
			stat.ViewedOpportunities,
			stat.ClickedOpportunities,
			stat.ConversionRate,
			stat.DailyRevenue,
		))
	}

	edit := tgbotapi.NewEditMessageText(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		text.String(),
	)
	edit.ParseMode = "HTML"
	edit.ReplyMarkup = b.buildAnalyticsKeyboard()

	b.sendMessage(edit)
	b.sendMessage(tgbotapi.NewCallback(callback.ID, ""))
}

// formatUserStats formats user statistics for display
func (b *Bot) formatUserStats(user *models.User, analytics *models.UserAnalytics, engagements []*models.UserEngagement) string {
	var text strings.Builder

	tier := "🆓 Free"
	if user.IsPremium() {
		tier = "💎 Premium"
	}

	text.WriteString(fmt.Sprintf("📊 <b>Твоя статистика</b>\n\n"))
	text.WriteString(fmt.Sprintf("Підписка: %s\n", tier))
	text.WriteString(fmt.Sprintf("Реєстрація: %s (%d днів тому)\n\n",
		user.CreatedAt.Format("02.01.2006"),
		analytics.DaysSinceRegistration))

	if analytics != nil {
		text.WriteString("<b>📈 Активність</b>\n")
		text.WriteString(fmt.Sprintf("• Переглянуто: %d можливостей\n", analytics.ViewedOpportunities))
		text.WriteString(fmt.Sprintf("• Кліків: %d\n", analytics.ClickedOpportunities))
		text.WriteString(fmt.Sprintf("• Участі: %d\n", analytics.ParticipatedOpportunities))
		text.WriteString(fmt.Sprintf("• Сесій: %d (сер. %d хв)\n\n",
			analytics.TotalSessions,
			analytics.AverageSessionTime/60))

		if analytics.ViewedOpportunities > 0 {
			text.WriteString("<b>💹 Конверсія</b>\n")
			text.WriteString(fmt.Sprintf("• Перегляд → Клік: %.1f%%\n", analytics.ViewToClickRate))
			text.WriteString(fmt.Sprintf("• Клік → Участь: %.1f%%\n", analytics.ClickToParticipateRate))
			text.WriteString(fmt.Sprintf("• Загальна: %.1f%%\n\n", analytics.OverallConversionRate))
		}

		if len(analytics.FavoriteTypes) > 0 || len(analytics.FavoriteExchanges) > 0 {
			text.WriteString("<b>⭐ Улюблені</b>\n")
			if len(analytics.FavoriteTypes) > 0 {
				text.WriteString(fmt.Sprintf("• Типи: %v\n", analytics.FavoriteTypes))
			}
			if len(analytics.FavoriteExchanges) > 0 {
				text.WriteString(fmt.Sprintf("• Біржі: %v\n", analytics.FavoriteExchanges))
			}
			text.WriteString("\n")
		}

		if analytics.NotificationsReceived > 0 {
			openRate := float64(analytics.NotificationsOpened) / float64(analytics.NotificationsReceived) * 100
			text.WriteString(fmt.Sprintf("<b>🔔 Сповіщення</b>\n"))
			text.WriteString(fmt.Sprintf("• Отримано: %d\n", analytics.NotificationsReceived))
			text.WriteString(fmt.Sprintf("• Відкрито: %d (%.1f%%)\n\n", analytics.NotificationsOpened, openRate))
		}
	}

	// Show last 7 days engagement
	if len(engagements) > 0 {
		text.WriteString("<b>📅 Активність за 7 днів</b>\n")
		for _, eng := range engagements {
			if eng == nil {
				continue
			}

			level := "🟢"
			if eng.EngagementLevel == "medium" {
				level = "🟡"
			} else if eng.EngagementLevel == "low" {
				level = "🔴"
			}

			text.WriteString(fmt.Sprintf(
				"%s %s: %d дій, %d хв\n",
				level,
				eng.Date.Format("Jan 02"),
				eng.ActionsCount,
				eng.TimeSpent/60,
			))
		}
	}

	return text.String()
}

// buildStatsKeyboard creates keyboard for stats view
func (b *Bot) buildStatsKeyboard(user *models.User) *tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Оновити", "refresh_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Головне меню", CallbackMenuAll),
		),
	)

	return &keyboard
}

// buildAnalyticsKeyboard creates keyboard for analytics admin view
func (b *Bot) buildAnalyticsKeyboard() *tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏆 Top Opportunities", "analytics_top_opps"),
			tgbotapi.NewInlineKeyboardButtonData("👥 Top Users", "analytics_top_users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Daily Report", "analytics_daily"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "analytics_refresh"),
		),
	)

	return &keyboard
}

// isAdmin checks if user has admin privileges
func (b *Bot) isAdmin(user *models.User) bool {
	// You can implement admin check logic here
	// For now, check against config or database
	// Example: check if user is in admin list
	return user.TelegramID == 123456789 // Replace with actual admin check
}
