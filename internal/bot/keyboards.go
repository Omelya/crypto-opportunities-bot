package bot

import (
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/payment/monobank"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) buildLanguageKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇺🇦 Українська", CallbackLanguageUK),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇬🇧 English", CallbackLanguageEN),
		),
	)
}

func (b *Bot) buildCapitalKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$100-500", CallbackCapital100_500),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$500-2000", CallbackCapital500_2000),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$2000-5000", CallbackCapital2000_5000),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$5000+", CallbackCapital5000Plus),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏭️ Пропустити", CallbackSkipCapital),
		),
	)
}

func (b *Bot) buildMainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Сьогодні", CallbackMenuToday),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Всі можливості", CallbackMenuToday),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Налаштування", CallbackMenuSettings),
			tgbotapi.NewInlineKeyboardButtonData("📈 Статистика", CallbackMenuStats),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💎 Premium", CallbackMenuPremium),
		),
	)
}

func (b *Bot) buildPremiumKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Спробувати 7 днів", CallbackPremiumTry),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("💎 Місячна - %d UAH", monobank.PlanPrices[monobank.PlanPremiumMonthly]/100), CallbackPremiumMonthly),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("👑 Річна - %d UAH (знижка 16%%)", monobank.PlanPrices[monobank.PlanPremiumYearly]/100), CallbackPremiumYearly),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackMenuAll),
		),
	)
}

func (b *Bot) buildRiskKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 Низький", CallbackRiskLow),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟡 Середній", CallbackRiskMedium),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Високий", CallbackRiskHigh),
		),
	)
}

func (b *Bot) buildOpportunitiesKeyboard(selected ...string) tgbotapi.InlineKeyboardMarkup {
	isSelected := func(oppType string) bool {
		for _, s := range selected {
			if s == oppType {
				return true
			}
		}
		return false
	}

	mark := func(oppType string) string {
		if isSelected(oppType) {
			return "✅ "
		}
		return ""
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				mark("launchpool")+"Launchpool",
				CallbackOppLaunchpool,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				mark("airdrop")+"Airdrops",
				CallbackOppAirdrop,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				mark("learn_earn")+"Learn & Earn",
				CallbackOppLearnEarn,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"➡️ Продовжити",
				CallbackOppComplete,
			),
		),
	)
}

func (b *Bot) buildSettingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Змінити капітал", CallbackSettingsCapital),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚖️ Змінити ризик-профіль", CallbackSettingsRisk),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏦 Обрати біржі", CallbackSettingsExchanges),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Типи можливостей", CallbackSettingsTypes),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌐 Змінити мову", CallbackSettingsLanguage),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📬 Дайджест", CallbackSettingsDigest),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Головне меню", CallbackMenuAll),
		),
	)
}

func (b *Bot) buildOpportunitiesFilterKeyboard(currentFilter string, hasPagination bool, page int) tgbotapi.InlineKeyboardMarkup {
	mark := func(filter string) string {
		if currentFilter == filter {
			return "✅ "
		}
		return ""
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("all")+"🌐 Всі", CallbackFilterAll),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("launchpool")+"🚀 Launchpool", CallbackFilterLaunchpool),
			tgbotapi.NewInlineKeyboardButtonData(mark("airdrop")+"🎁 Airdrops", CallbackFilterAirdrop),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("learn_earn")+"📚 Learn & Earn", CallbackFilterLearnEarn),
			tgbotapi.NewInlineKeyboardButtonData(mark("staking")+"💎 Staking", CallbackFilterStaking),
		),
	}

	if hasPagination {
		paginationRow := []tgbotapi.InlineKeyboardButton{}
		if page > 0 {
			paginationRow = append(paginationRow,
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("%s%d", CallbackPagePrev, page-1)),
			)
		}
		paginationRow = append(paginationRow,
			tgbotapi.NewInlineKeyboardButtonData("➡️ Далі", fmt.Sprintf("%s%d", CallbackPageNext, page+1)),
		)
		rows = append(rows, paginationRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Головне меню", CallbackMenuAll),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) buildOpportunityDetailKeyboard(opp *models.Opportunity) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	if opp.URL != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔗 Перейти на біржу", opp.URL),
		))
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад до списку", CallbackFilterAll),
		),
	)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
