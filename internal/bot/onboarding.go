package bot

import (
	"crypto-opportunities-bot/internal/models"
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type OnboardingStep int

const (
	StepLanguage OnboardingStep = iota
	StepCapital
	StepRisk
	StepOpportunities
	StepComplete
)

type OnboardingState struct {
	Step            OnboardingStep
	SelectedCapital string
	SelectedRisk    string
	SelectedOpps    []string
}

// Зберігаємо стани в пам'яті (для production краще Redis)
type OnboardingManager struct {
	states map[int64]*OnboardingState
	mu     sync.RWMutex
}

func NewOnboardingManager() *OnboardingManager {
	return &OnboardingManager{
		states: make(map[int64]*OnboardingState),
	}
}

func (om *OnboardingManager) GetState(userID int64) *OnboardingState {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.states[userID]
}

func (om *OnboardingManager) SetState(userID int64, state *OnboardingState) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.states[userID] = state
}

func (om *OnboardingManager) DeleteState(userID int64) {
	om.mu.Lock()
	defer om.mu.Unlock()
	delete(om.states, userID)
}

func (b *Bot) startOnboarding(chatID int64, user *models.User) {
	state := &OnboardingState{
		Step: StepLanguage,
	}
	b.onboardingManager.SetState(user.TelegramID, state)

	text := fmt.Sprintf(
		"👋 Привіт, %s!\n\n"+
			"Я <b>Crypto Opportunities Assistant</b>.\n\n"+
			"Я допоможу тобі:\n"+
			"🎯 Знаходити прибуткові можливості на біржах\n"+
			"💰 Не пропускати аірдропи та лаунчпули\n"+
			"📈 Заробляти більше на крипто\n\n"+
			"Почнемо налаштування? Це займе 1 хвилину.\n\n"+
			"<b>Крок 1/4:</b> Обери мову 👇",
		user.FirstName,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildLanguageKeyboard()

	b.api.Send(msg)
}

func (b *Bot) handleLanguageSelect(callback *tgbotapi.CallbackQuery, lang string) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, _ := b.userRepo.GetByTelegramID(userID)
	if user == nil {
		return
	}

	user.LanguageCode = lang
	b.userRepo.Update(user)

	state := b.onboardingManager.GetState(userID)
	if state != nil {
		state.Step = StepCapital
		b.onboardingManager.SetState(userID, state)
	}

	text := "✅ Чудово!\n\n" +
		"<b>Крок 2/4:</b> Який у тебе капітал для інвестицій? 💰\n\n" +
		"Це допоможе показувати тільки підходящі можливості."

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildCapitalKeyboard()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	b.sendMessage(msg)
}

func (b *Bot) handleCapitalSelect(callback *tgbotapi.CallbackQuery, capital string) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, _ := b.userRepo.GetByTelegramID(userID)
	if user == nil {
		return
	}

	user.CapitalRange = capital
	b.userRepo.Update(user)

	// Оновлюємо стан
	state := b.onboardingManager.GetState(userID)
	if state != nil {
		state.SelectedCapital = capital
		state.Step = StepRisk
		b.onboardingManager.SetState(userID, state)
	}

	text := "💪 Відмінно!\n\n" +
		"<b>Крок 3/4:</b> Який твій ризик-профіль? ⚖️\n\n" +
		"🟢 <b>Низький</b> - Консервативні інвестиції\n" +
		"🟡 <b>Середній</b> - Баланс ризику та прибутку\n" +
		"🔴 <b>Високий</b> - Агресивні стратегії"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildRiskKeyboard()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	b.sendMessage(msg)
}

func (b *Bot) handleRiskSelect(callback *tgbotapi.CallbackQuery, risk string) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, _ := b.userRepo.GetByTelegramID(userID)
	if user == nil {
		return
	}

	user.RiskProfile = risk
	b.userRepo.Update(user)

	state := b.onboardingManager.GetState(userID)
	if state != nil {
		state.SelectedRisk = risk
		state.Step = StepOpportunities
		b.onboardingManager.SetState(userID, state)
	}

	text := "🎯 Супер!\n\n" +
		"<b>Крок 4/4:</b> Які можливості тебе цікавлять? 📊\n\n" +
		"Можеш вибрати кілька варіантів:"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildOpportunitiesKeyboard()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	b.sendMessage(msg)
}

func (b *Bot) handleOpportunitiesToggle(callback *tgbotapi.CallbackQuery, oppType string) {
	userID := callback.From.ID

	state := b.onboardingManager.GetState(userID)
	if state == nil {
		return
	}

	found := false
	newOpps := []string{}

	for _, opp := range state.SelectedOpps {
		if opp == oppType {
			found = true
		} else {
			newOpps = append(newOpps, opp)
		}
	}

	if !found {
		newOpps = append(newOpps, oppType)
	}

	state.SelectedOpps = newOpps
	b.onboardingManager.SetState(userID, state)

	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
		b.buildOpportunitiesKeyboard(state.SelectedOpps...),
	)
	b.sendMessage(editMsg)
}

func (b *Bot) handleOpportunitiesComplete(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	state := b.onboardingManager.GetState(userID)
	if state == nil {
		return
	}

	user, _ := b.userRepo.GetByTelegramID(userID)
	if user == nil {
		return
	}

	prefs := &models.UserPreferences{
		UserID:           user.ID,
		OpportunityTypes: state.SelectedOpps,
		Exchanges:        []string{"binance", "bybit"}, // Default
	}

	b.prefsRepo.Create(prefs)

	b.onboardingManager.DeleteState(userID)

	text := "🎉 <b>Готово!</b>\n\n" +
		"Ти отримуватимеш алерти про підходящі можливості.\n\n" +
		"💎 Хочеш отримувати на <b>80% більше</b> можливостей?\n" +
		"Спробуй Premium безкоштовно 7 днів!"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = b.buildPremiumOfferKeyboard()

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
	b.sendMessage(deleteMsg)

	b.sendMessage(msg)
}
