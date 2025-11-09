package notification

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"strings"
	"time"
)

type Formatter struct{}

func NewFormatter() *Formatter {
	return &Formatter{}
}

func (f *Formatter) FormatOpportunity(opp *models.Opportunity) string {
	var builder strings.Builder

	emoji := f.getOpportunityEmoji(opp.Type)

	builder.WriteString(fmt.Sprintf("%s <b>%s</b>\n\n", emoji, opp.Title))

	builder.WriteString(fmt.Sprintf("🏦 Біржа: <b>%s</b>\n", strings.Title(opp.Exchange)))

	if opp.Reward != "" {
		builder.WriteString(fmt.Sprintf("💰 Винагорода: <b>%s</b>\n", opp.Reward))
	}

	if opp.EstimatedROI > 0 {
		builder.WriteString(fmt.Sprintf("📈 Очікуваний ROI: <b>%.2f%%</b>\n", opp.EstimatedROI))
	}

	if opp.MinInvestment > 0 {
		builder.WriteString(fmt.Sprintf("💵 Мін. інвестиція: <b>$%.2f</b>\n", opp.MinInvestment))
	}

	if opp.Duration != "" {
		builder.WriteString(fmt.Sprintf("⏱️ Тривалість: <b>%s</b>\n", opp.Duration))
	}

	if opp.EndDate != nil {
		daysLeft := opp.DaysLeft()
		if daysLeft >= 0 {
			builder.WriteString(fmt.Sprintf("⏰ Залишилось: <b>%d днів</b>\n", daysLeft))
		}
	}

	if opp.Requirements != "" {
		builder.WriteString(fmt.Sprintf("\n📋 Вимоги:\n%s\n", opp.Requirements))
	}

	if opp.Description != "" {
		desc := opp.Description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		builder.WriteString(fmt.Sprintf("\n💡 %s\n", desc))
	}

	return builder.String()
}

// FormatDailyDigest форматує щоденний дайджест
func (f *Formatter) FormatDailyDigest(opportunities []*models.Opportunity, user *models.User) string {
	var builder strings.Builder

	date := time.Now().Format("02.01.2006")
	greeting := f.getGreeting(user)

	builder.WriteString(fmt.Sprintf("📊 <b>%s</b>\n\n", greeting))
	builder.WriteString(fmt.Sprintf("Твій крипто-звіт за %s\n\n", date))

	if len(opportunities) == 0 {
		builder.WriteString("🔍 Сьогодні немає нових можливостей, які відповідають твоїм критеріям.\n\n")
		builder.WriteString("💡 Спробуй розширити фільтри у /settings")
		return builder.String()
	}

	byType := f.groupByType(opportunities)

	builder.WriteString(fmt.Sprintf("🆕 <b>Нових можливостей: %d</b>\n\n", len(opportunities)))

	for oppType, opps := range byType {
		emoji := f.getOpportunityEmoji(oppType)
		typeName := f.getTypeName(oppType)

		builder.WriteString(fmt.Sprintf("%s <b>%s (%d)</b>\n", emoji, typeName, len(opps)))

		for i, opp := range opps {
			if i >= 3 {
				builder.WriteString(fmt.Sprintf("   ... і ще %d\n", len(opps)-3))
				break
			}

			roi := ""
			if opp.EstimatedROI > 0 {
				roi = fmt.Sprintf(" • %.1f%% ROI", opp.EstimatedROI)
			}

			duration := ""
			if opp.Duration != "" {
				duration = fmt.Sprintf(" • %s", opp.Duration)
			}

			builder.WriteString(fmt.Sprintf("   • %s - %s%s%s\n",
				strings.Title(opp.Exchange),
				f.truncateTitle(opp.Title, 40),
				roi,
				duration,
			))
		}
		builder.WriteString("\n")
	}

	// Потенційна вигода
	minProfit, maxProfit := f.calculatePotentialProfit(opportunities, user)
	if minProfit > 0 {
		builder.WriteString(fmt.Sprintf("💵 <b>Твоя потенційна вигода: $%.0f-%.0f</b>\n\n", minProfit, maxProfit))
	}

	// Заклик до дії
	builder.WriteString("👉 /today - Переглянути всі можливості\n")
	builder.WriteString("⚙️ /settings - Налаштування фільтрів")

	return builder.String()
}

// FormatPremiumTeaser форматує тізер Premium для Free користувачів
func (f *Formatter) FormatPremiumTeaser(missedOpportunities int) string {
	var builder strings.Builder

	builder.WriteString("\n\n💎 <b>Premium користувачі також отримали:</b>\n")
	builder.WriteString(fmt.Sprintf("• %d арбітражних можливостей\n", missedOpportunities))
	builder.WriteString("• Real-time алерти (0-2 хв)\n")
	builder.WriteString("• DeFi пули з високим APR\n")
	builder.WriteString("• Китові транзакції\n\n")
	builder.WriteString("🚀 /premium - Дізнатись більше")

	return builder.String()
}

// FormatArbitrageAlert форматує арбітражний алерт (Premium)
func (f *Formatter) FormatArbitrageAlert(exchangeBuy, exchangeSell, pair string,
	priceBuy, priceSell, profitPercent, netProfitPercent float64) string {

	var builder strings.Builder

	builder.WriteString("🔥 <b>АРБІТРАЖ!</b>\n\n")
	builder.WriteString(fmt.Sprintf("Пара: <b>%s</b>\n", pair))
	builder.WriteString(fmt.Sprintf("Купити: %s <b>$%.2f</b>\n", strings.Title(exchangeBuy), priceBuy))
	builder.WriteString(fmt.Sprintf("Продати: %s <b>$%.2f</b>\n\n", strings.Title(exchangeSell), priceSell))

	builder.WriteString(fmt.Sprintf("💰 Profit: <b>%.2f%%</b>\n", profitPercent))
	builder.WriteString(fmt.Sprintf("📊 На $1000: <b>$%.2f profit</b>\n", profitPercent*10))
	builder.WriteString("💵 Рекомендовано: $500-2000\n\n")

	builder.WriteString("⏰ Актуально: ~3-5 хвилин\n")
	builder.WriteString(fmt.Sprintf("⚠️ Fees включено: -%.2f%%\n", profitPercent-netProfitPercent))
	builder.WriteString(fmt.Sprintf("✅ Чистий profit: <b>%.2f%% ($%.2f на $1000)</b>\n",
		netProfitPercent, netProfitPercent*10))

	return builder.String()
}

// Helper методи

func (f *Formatter) getOpportunityEmoji(oppType string) string {
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

func (f *Formatter) getTypeName(oppType string) string {
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

func (f *Formatter) getGreeting(user *models.User) string {
	hour := time.Now().Hour()

	var timeGreeting string
	switch {
	case hour < 6:
		timeGreeting = "Доброї ночі"
	case hour < 12:
		timeGreeting = "Доброго ранку"
	case hour < 18:
		timeGreeting = "Доброго дня"
	default:
		timeGreeting = "Доброго вечора"
	}

	if user.FirstName != "" {
		return fmt.Sprintf("%s, %s!", timeGreeting, user.FirstName)
	}

	return timeGreeting + "!"
}

func (f *Formatter) groupByType(opportunities []*models.Opportunity) map[string][]*models.Opportunity {
	result := make(map[string][]*models.Opportunity)

	for _, opp := range opportunities {
		result[opp.Type] = append(result[opp.Type], opp)
	}

	return result
}

func (f *Formatter) truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

func (f *Formatter) calculatePotentialProfit(opportunities []*models.Opportunity, user *models.User) (float64, float64) {
	// Простий розрахунок базуючись на capital range
	capitalMin, capitalMax := f.getCapitalRange(user.CapitalRange)

	var totalROI float64
	count := 0

	for _, opp := range opportunities {
		if opp.EstimatedROI > 0 {
			totalROI += opp.EstimatedROI
			count++
		}
	}

	if count == 0 {
		return 0, 0
	}

	avgROI := totalROI / float64(count)

	// Консервативна оцінка: користувач використає 20-50% можливостей
	minProfit := capitalMin * (avgROI / 100) * 0.2
	maxProfit := capitalMax * (avgROI / 100) * 0.5

	return minProfit, maxProfit
}

func (f *Formatter) getCapitalRange(capitalRange string) (float64, float64) {
	switch capitalRange {
	case "100-500":
		return 100, 500
	case "500-2000":
		return 500, 2000
	case "2000-5000":
		return 2000, 5000
	case "5000+":
		return 5000, 10000
	default:
		return 500, 1000
	}
}
