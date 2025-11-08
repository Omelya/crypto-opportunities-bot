package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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
			tgbotapi.NewInlineKeyboardButtonData("💰 Сьогодні", "menu_today"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Всі можливості", "menu_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Налаштування", "menu_settings"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Статистика", "menu_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💎 Premium", "menu_premium"),
		),
	)
}

func (b *Bot) buildPremiumKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Спробувати 7 днів", CallbackPremiumTry),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Підписатись ($9/міс)", CallbackPremiumBuy),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Залишитись на Free", CallbackStayFree),
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

func (b *Bot) buildPremiumOfferKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Спробувати Premium", CallbackPremiumTry),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Залишитись на Free", CallbackStayFree),
		),
	)
}
