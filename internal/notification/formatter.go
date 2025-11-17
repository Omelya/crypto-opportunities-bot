package notification

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Formatter struct {
	titleCaser cases.Caser
}

func NewFormatter() *Formatter {
	return &Formatter{
		titleCaser: cases.Title(language.English),
	}
}

// titleCase замінює deprecated strings.Title()
func (f *Formatter) titleCase(s string) string {
	return f.titleCaser.String(s)
}

func (f *Formatter) FormatOpportunity(opp *models.Opportunity) string {
	var builder strings.Builder

	emoji := f.getOpportunityEmoji(opp.Type)

	builder.WriteString(fmt.Sprintf("%s <b>%s</b>\n\n", emoji, opp.Title))

	builder.WriteString(fmt.Sprintf("🏦 Біржа: <b>%s</b>\n", f.titleCase(opp.Exchange)))

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
				f.titleCase(opp.Exchange),
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

// FormatArbitrage форматує арбітражну можливість з моделі
func (f *Formatter) FormatArbitrage(arb *models.ArbitrageOpportunity) string {
	var builder strings.Builder

	emoji := "💰"
	if arb.NetProfitPercent >= 1.0 {
		emoji = "🔥🔥"
	} else if arb.NetProfitPercent >= 0.5 {
		emoji = "🔥"
	}

	builder.WriteString(fmt.Sprintf("%s <b>АРБІТРАЖ!</b>\n\n", emoji))
	builder.WriteString(fmt.Sprintf("Пара: <b>%s</b>\n", arb.Pair))
	builder.WriteString(fmt.Sprintf("🟢 Купити: <b>%s</b> @ $%.4f\n", f.titleCase(arb.ExchangeBuy), arb.PriceBuy))
	builder.WriteString(fmt.Sprintf("🔴 Продати: <b>%s</b> @ $%.4f\n\n", f.titleCase(arb.ExchangeSell), arb.PriceSell))

	builder.WriteString(fmt.Sprintf("💵 Валовий profit: <b>%.2f%%</b>\n", arb.ProfitPercent))
	builder.WriteString(fmt.Sprintf("📊 На $1000: <b>$%.2f</b>\n", arb.ProfitUSD))
	builder.WriteString(fmt.Sprintf("💼 Рекомендовано: <b>$%.0f-%.0f</b>\n\n", arb.MinTradeAmount, arb.RecommendedAmount))

	builder.WriteString(fmt.Sprintf("⚠️ Trading fees: <b>-%.2f%%</b>\n", arb.TotalFeesPercent))
	builder.WriteString(fmt.Sprintf("📉 Slippage: <b>-%.2f%%</b>\n", arb.SlippageBuy+arb.SlippageSell))
	builder.WriteString(fmt.Sprintf("✅ Чистий profit: <b>%.2f%%</b> (<b>$%.2f</b> на $1000)\n\n",
		arb.NetProfitPercent, arb.NetProfitUSD))

	// Time left
	timeLeft := arb.TimeLeft()
	minutesLeft := int(timeLeft.Minutes())
	if minutesLeft < 0 {
		minutesLeft = 0
	}
	builder.WriteString(fmt.Sprintf("⏰ Залишилось: ~%d хв\n", minutesLeft))

	builder.WriteString("\n⚠️ <i>Це інформація, не гарантія прибутку. Ціни змінюються швидко.</i>")

	return builder.String()
}

// FormatArbitrageAlert форматує арбітражний алерт (Premium) - legacy
func (f *Formatter) FormatArbitrageAlert(exchangeBuy, exchangeSell, pair string,
	priceBuy, priceSell, profitPercent, netProfitPercent float64) string {

	var builder strings.Builder

	builder.WriteString("🔥 <b>АРБІТРАЖ!</b>\n\n")
	builder.WriteString(fmt.Sprintf("Пара: <b>%s</b>\n", pair))
	builder.WriteString(fmt.Sprintf("Купити: %s <b>$%.2f</b>\n", f.titleCase(exchangeBuy), priceBuy))
	builder.WriteString(fmt.Sprintf("Продати: %s <b>$%.2f</b>\n\n", f.titleCase(exchangeSell), priceSell))

	builder.WriteString(fmt.Sprintf("💰 Profit: <b>%.2f%%</b>\n", profitPercent))
	builder.WriteString(fmt.Sprintf("📊 На $1000: <b>$%.2f profit</b>\n", profitPercent*10))
	builder.WriteString("💵 Рекомендовано: $500-2000\n\n")

	builder.WriteString("⏰ Актуально: ~3-5 хвилин\n")
	builder.WriteString(fmt.Sprintf("⚠️ Fees включено: -%.2f%%\n", profitPercent-netProfitPercent))
	builder.WriteString(fmt.Sprintf("✅ Чистий profit: <b>%.2f%% ($%.2f на $1000)</b>\n",
		netProfitPercent, netProfitPercent*10))

	return builder.String()
}

// FormatDeFi форматує DeFi opportunity
func (f *Formatter) FormatDeFi(defi *models.DeFiOpportunity) string {
	var builder strings.Builder

	// Emoji based on APY
	emoji := "🌾"
	if defi.APY >= 50 {
		emoji = "🔥🌾"
	} else if defi.APY >= 30 {
		emoji = "⭐🌾"
	}

	builder.WriteString(fmt.Sprintf("%s <b>DeFi Opportunity</b>\n\n", emoji))

	// Protocol and Chain
	builder.WriteString(fmt.Sprintf("🏦 Protocol: <b>%s</b>\n", f.titleCase(defi.Protocol)))
	builder.WriteString(fmt.Sprintf("⛓️ Chain: <b>%s</b>\n", f.titleCase(defi.Chain)))
	builder.WriteString(fmt.Sprintf("💧 Pool: <b>%s</b>\n\n", defi.GetDisplayName()))

	// Profitability
	builder.WriteString(fmt.Sprintf("📈 APY: <b>%.2f%%</b>\n", defi.APY))
	if defi.APYBase > 0 && defi.APYReward > 0 {
		builder.WriteString(fmt.Sprintf("   ├ Base: %.2f%%\n", defi.APYBase))
		builder.WriteString(fmt.Sprintf("   └ Rewards: %.2f%%\n", defi.APYReward))
	}
	builder.WriteString(fmt.Sprintf("💰 Daily return: <b>%.3f%%</b>\n", defi.DailyReturn))
	builder.WriteString(fmt.Sprintf("💵 На $1000: <b>$%.2f/день</b> (<b>$%.2f/місяць</b>)\n\n",
		defi.DailyReturnUSD(1000), defi.MonthlyReturnUSD(1000)))

	// Liquidity and Volume
	builder.WriteString(fmt.Sprintf("📊 TVL: <b>$%.2fM</b>\n", defi.TVL/1_000_000))
	if defi.Volume24h > 0 {
		builder.WriteString(fmt.Sprintf("📈 Volume 24h: <b>$%.2fK</b>\n", defi.Volume24h/1000))
	}
	builder.WriteString("\n")

	// Risk Assessment
	riskEmoji := f.getRiskEmoji(defi.RiskLevel)
	builder.WriteString(fmt.Sprintf("%s Risk: <b>%s</b>\n", riskEmoji, f.getRiskName(defi.RiskLevel)))

	if defi.ILRisk > 0 {
		ilEmoji := "✅"
		if defi.ILRisk > 10 {
			ilEmoji = "⚠️"
		} else if defi.ILRisk > 5 {
			ilEmoji = "⚡"
		}
		builder.WriteString(fmt.Sprintf("%s IL Risk: <b>%.1f%%</b>\n", ilEmoji, defi.ILRisk))
	}

	if defi.IsAudited() {
		builder.WriteString("✅ Audited: <b>Yes</b>\n")
	}
	builder.WriteString("\n")

	// Requirements
	builder.WriteString(fmt.Sprintf("💼 Min Deposit: <b>$%.0f</b>\n", defi.MinDeposit))

	if defi.HasLockPeriod() {
		builder.WriteString(fmt.Sprintf("🔒 Lock Period: <b>%d days</b>\n", defi.LockPeriod))
	} else {
		builder.WriteString("🔓 No lock period\n")
	}

	// Rewards
	if len(defi.RewardTokens) > 0 {
		builder.WriteString(fmt.Sprintf("🎁 Rewards: <b>%s</b>\n", strings.Join(defi.RewardTokens, ", ")))
	}

	builder.WriteString("\n⚠️ <i>DeFi involves risks. DYOR before investing.</i>")

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

func (f *Formatter) getRiskEmoji(riskLevel string) string {
	switch riskLevel {
	case "low":
		return "✅"
	case "medium":
		return "⚡"
	case "high":
		return "⚠️"
	default:
		return "❓"
	}
}

func (f *Formatter) getRiskName(riskLevel string) string {
	switch riskLevel {
	case "low":
		return "Низький"
	case "medium":
		return "Середній"
	case "high":
		return "Високий"
	default:
		return "Невідомий"
	}
}
