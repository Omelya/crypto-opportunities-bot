package bot

import (
	"crypto-opportunities-bot/internal/payment/monobank"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) buildPremiumPlansKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		// Річна підписка (найвигідніша)
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("👑 Річна - %d UAH (знижка 16%%)", monobank.PlanPrices[monobank.PlanPremiumYearly]/100),
				"premium:"+monobank.PlanPremiumYearly,
			),
		),
		// Місячна підписка (популярна)
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("💎 Місячна - %d UAH", monobank.PlanPrices[monobank.PlanPremiumMonthly]/100),
				"premium:"+monobank.PlanPremiumMonthly,
			),
		),
		// Тижнева підписка (для тестування)
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⚡ Тижнева - %d UAH", monobank.PlanPrices[monobank.PlanPremiumWeekly]/100),
				"premium:"+monobank.PlanPremiumWeekly,
			),
		),
		// Відміна
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Відміна", CallbackMenuAll),
		),
	)

}

func (b *Bot) buildPremiumOfferKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Спробувати Premium", CallbackPremiumTry),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Залишитись на Free", CallbackMenuAll),
		),
	)
}

func (b *Bot) buildSubscriptionManagementKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏸️ Скасувати підписку", "cancel_subscription"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Головне меню", CallbackMenuAll),
		),
	)
}
