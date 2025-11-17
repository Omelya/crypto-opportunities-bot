package bot

import (
	"crypto-opportunities-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) buildCapitalSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$100-500", CallbackSetCapital100_500),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$500-2000", CallbackSetCapital500_2000),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$2000-5000", CallbackSetCapital2000_5000),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("$5000+", CallbackSetCapital5000Plus),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackSettingsBack),
		),
	)
}

func (b *Bot) buildRiskSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 Низький", CallbackSetRiskLow),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟡 Середній", CallbackSetRiskMedium),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Високий", CallbackSetRiskHigh),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackSettingsBack),
		),
	)
}

func (b *Bot) buildLanguageSelectionKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇺🇦 Українська", CallbackSetLanguageUK),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇬🇧 English", CallbackSetLanguageEN),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackSettingsBack),
		),
	)
}

func (b *Bot) buildExchangeSelectionKeyboard(selected []string) tgbotapi.InlineKeyboardMarkup {
	isSelected := func(exchange string) bool {
		for _, s := range selected {
			if s == exchange {
				return true
			}
		}
		return false
	}

	mark := func(exchange string) string {
		if isSelected(exchange) {
			return "✅ "
		}
		return ""
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("binance")+"Binance", CallbackExchangeBinance),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("bybit")+"Bybit", CallbackExchangeBybit),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("okx")+"OKX", CallbackExchangeOKX),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("gateio")+"Gate.io", CallbackExchangeGateIO),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Готово", CallbackExchangeDone),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackSettingsBack),
		),
	)
}

func (b *Bot) buildTypeSelectionKeyboard(selected []string) tgbotapi.InlineKeyboardMarkup {
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
			tgbotapi.NewInlineKeyboardButtonData(mark("launchpool")+"🚀 Launchpool", CallbackTypeLaunchpool),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("airdrop")+"🎁 Airdrops", CallbackTypeAirdrop),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("learn_earn")+"📚 Learn & Earn", CallbackTypeLearnEarn),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mark("staking")+"💎 Staking", CallbackTypeStaking),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Готово", CallbackTypeDone),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackSettingsBack),
		),
	)
}

func (b *Bot) buildDigestSettingsKeyboard(prefs *models.UserPreferences) tgbotapi.InlineKeyboardMarkup {
	toggleText := "❌ Вимкнути"
	if !prefs.DailyDigestEnabled {
		toggleText = "✅ Ввімкнути"
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(toggleText, CallbackDigestToggle),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Готово", CallbackDigestDone),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", CallbackSettingsBack),
		),
	)
}
