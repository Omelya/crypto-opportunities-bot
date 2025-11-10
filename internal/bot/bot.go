package bot

import (
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/repository"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api               *tgbotapi.BotAPI
	userRepo          repository.UserRepository
	prefsRepo         repository.UserPreferencesRepository
	oppRepo           repository.OpportunityRepository
	actionRepo        repository.UserActionRepository
	config            *config.Config
	onboardingManager *OnboardingManager
}

func NewBot(
	cfg *config.Config,
	userRepo repository.UserRepository,
	prefsRepo repository.UserPreferencesRepository,
	oppRepo repository.OpportunityRepository,
	actionRepo repository.UserActionRepository,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = cfg.Telegram.Debug

	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api,
		userRepo,
		prefsRepo,
		oppRepo,
		actionRepo,
		cfg,
		NewOnboardingManager(),
	}, nil
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		go b.handleUpdate(update)
	}

	return nil
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		b.handleMessage(update.Message)
		return
	}

	if update.CallbackQuery != nil {
		b.handleCallback(update.CallbackQuery)
		return
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if !message.IsCommand() {
		return
	}

	switch message.Command() {
	case CommandStart:
		b.handleStart(message)
	case CommandHelp:
		b.handleHelp(message)
	case CommandToday:
		b.handleToday(message)
	case CommandSettings:
		b.handleSettings(message)
	case CommandStats:
		b.handleStats(message)
	case CommandPremium:
		b.handlePremium(message)
	case CommandSupport:
		b.handleSupport(message)
	default:
		b.handleUnknown(message)
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	_, err := b.api.Send(tgbotapi.NewCallback(callback.Data, ""))
	if err != nil {
		log.Println(err)
	}

	data := callback.Data

	// Menu callbacks
	if strings.HasPrefix(data, "menu_") {
		b.handleMenuCallback(callback)
		return
	}

	// Filter callbacks
	if strings.HasPrefix(data, "filter_") {
		b.handleFilterCallback(callback)
		return
	}

	// Pagination callbacks
	if strings.HasPrefix(data, "page_") {
		b.handlePaginationCallback(callback)
		return
	}

	// Onboarding callbacks
	switch callback.Data {
	// Language
	case CallbackLanguageUK:
		b.handleLanguageSelect(callback, "uk")
	case CallbackLanguageRU:
		b.handleLanguageSelect(callback, "ru")
	case CallbackLanguageEN:
		b.handleLanguageSelect(callback, "en")

	// Capital
	case CallbackCapital100_500:
		b.handleCapitalSelect(callback, "100-500")
	case CallbackCapital500_2000:
		b.handleCapitalSelect(callback, "500-2000")
	case CallbackCapital2000_5000:
		b.handleCapitalSelect(callback, "2000-5000")
	case CallbackCapital5000Plus:
		b.handleCapitalSelect(callback, "5000+")
	case CallbackSkipCapital:
		b.handleCapitalSelect(callback, "")

	// Risk
	case CallbackRiskLow:
		b.handleRiskSelect(callback, "low")
	case CallbackRiskMedium:
		b.handleRiskSelect(callback, "medium")
	case CallbackRiskHigh:
		b.handleRiskSelect(callback, "high")

	// Opportunities
	case CallbackOppLaunchpool:
		b.handleOpportunitiesToggle(callback, "launchpool")
	case CallbackOppAirdrop:
		b.handleOpportunitiesToggle(callback, "airdrop")
	case CallbackOppLearnEarn:
		b.handleOpportunitiesToggle(callback, "learn_earn")
	case CallbackOppComplete:
		b.handleOpportunitiesComplete(callback)

		// Premium
		//case CallbackPremiumTry, CallbackPremiumBuy:
		//	b.handlePremiumAction(callback)
		//case CallbackStayFree:
		//	b.handleStayFree(callback)
	}
}
