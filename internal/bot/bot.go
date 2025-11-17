package bot

import (
	"crypto-opportunities-bot/internal/config"
	"crypto-opportunities-bot/internal/models"
	"crypto-opportunities-bot/internal/payment"
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
	subsRepo          repository.SubscriptionRepository
	arbRepo           repository.ArbitrageRepository
	defiRepo          repository.DeFiRepository
	paymentService    *payment.Service
	config            *config.Config
	onboardingManager *OnboardingManager
}

func NewBot(
	cfg *config.Config,
	userRepo repository.UserRepository,
	prefsRepo repository.UserPreferencesRepository,
	oppRepo repository.OpportunityRepository,
	actionRepo repository.UserActionRepository,
	subsRepo repository.SubscriptionRepository,
	arbRepo repository.ArbitrageRepository,
	defiRepo repository.DeFiRepository,
	paymentService *payment.Service,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return nil, err
	}

	api.Debug = cfg.Telegram.Debug

	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		api:               api,
		userRepo:          userRepo,
		prefsRepo:         prefsRepo,
		oppRepo:           oppRepo,
		actionRepo:        actionRepo,
		subsRepo:          subsRepo,
		arbRepo:           arbRepo,
		defiRepo:          defiRepo,
		paymentService:    paymentService,
		config:            cfg,
		onboardingManager: NewOnboardingManager(),
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
	case CommandBuyPremium:
		b.handleBuyPremium(message)
	case CommandSubscription:
		b.handleSubscription(message)
	case CommandArbitrage:
		b.handleArbitrage(message)
	case CommandDeFi:
		b.handleDeFi(message)
	case CommandSupport:
		b.handleSupport(message)
	case "client":
		b.handleClient(message)
	case "clientstats":
		b.handleClientStats(message)
	default:
		b.handleUnknown(message)
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Premium callbacks
	if strings.HasPrefix(data, "premium:") {
		b.handlePremiumCallback(callback)
		return
	}

	if data == "cancel_subscription" {
		b.handleCancelSubscription(callback)
		return
	}

	if data == "cancel_payment" {
		b.sendMessage(tgbotapi.NewCallback(callback.ID, "Оплату скасовано"))
		deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)
		return
	}

	if data == CallbackRefreshArbitrage {
		b.handleArbitrageRefresh(callback)
		return
	}

	// DeFi callbacks
	if data == "refresh_defi" {
		b.handleDeFiRefresh(callback)
		return
	}

	if data == "defi_filter_apy" {
		b.handleDeFiRefresh(callback) // Same as refresh - top by APY
		return
	}

	if data == "defi_filter_tvl" {
		b.handleDeFiFilterByTVL(callback)
		return
	}

	if data == "defi_filter_low" {
		b.handleDeFiFilterByRisk(callback, "low")
		return
	}

	if data == "defi_filter_med" {
		b.handleDeFiFilterByRisk(callback, "medium")
		return
	}

	if data == "defi_filter_chain" {
		b.handleDeFiFilterChain(callback)
		return
	}

	if data == "defi_filter_protocol" {
		b.handleDeFiFilterProtocol(callback)
		return
	}

	// DeFi chain filters
	if strings.HasPrefix(data, "defi_chain_") {
		chain := strings.TrimPrefix(data, "defi_chain_")
		b.handleDeFiByChain(callback, chain)
		return
	}

	// DeFi protocol filters
	if strings.HasPrefix(data, "defi_protocol_") {
		protocol := strings.TrimPrefix(data, "defi_protocol_")
		b.handleDeFiByProtocol(callback, protocol)
		return
	}

	_, err := b.api.Send(tgbotapi.NewCallback(callback.ID, ""))
	if err != nil {
		log.Println(err)
	}

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

	// Settings callbacks
	if strings.HasPrefix(data, "settings_") && data != CallbackSettingsBack {
		b.handleSettingsCallback(callback)
		return
	}

	// Exchange toggles
	if strings.HasPrefix(data, "exchange_") && data != CallbackExchangeDone {
		switch data {
		case CallbackExchangeBinance:
			b.handleExchangeToggle(callback, "binance")
		case CallbackExchangeBybit:
			b.handleExchangeToggle(callback, "bybit")
		case CallbackExchangeOKX:
			b.handleExchangeToggle(callback, "okx")
		case CallbackExchangeGateIO:
			b.handleExchangeToggle(callback, "gateio")
		}
		return
	}

	// Type toggles
	if strings.HasPrefix(data, "type_") && data != CallbackTypeDone {
		switch data {
		case CallbackTypeLaunchpool:
			b.handleTypeToggle(callback, "launchpool")
		case CallbackTypeAirdrop:
			b.handleTypeToggle(callback, "airdrop")
		case CallbackTypeLearnEarn:
			b.handleTypeToggle(callback, "learn_earn")
		case CallbackTypeStaking:
			b.handleTypeToggle(callback, "staking")
		}
		return
	}

	// Digest settings
	if data == CallbackDigestToggle {
		b.handleDigestToggle(callback)
		return
	}

	if _, ok := map[string]struct{}{
		CallbackDigestDone:   {},
		CallbackSettingsBack: {},
		CallbackTypeDone:     {},
		CallbackExchangeDone: {},
	}[data]; ok {
		chatID := callback.Message.Chat.ID
		userID := callback.From.ID

		user, _ := b.userRepo.GetByTelegramID(userID)
		prefs, _ := b.prefsRepo.GetByUserID(user.ID)

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.showSettingsMenu(chatID, user, prefs)
		return
	}

	// Capital/Risk/Language settings
	if strings.HasPrefix(data, "set_") {
		b.handleSettingsUpdate(callback)
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
	}
}

func (b *Bot) handleSettingsUpdate(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	user, err := b.userRepo.GetByTelegramID(userID)
	if err != nil || user == nil {
		b.sendError(chatID)
		return
	}

	prefs, err := b.prefsRepo.GetByUserID(user.ID)
	if err != nil || prefs == nil {
		b.sendError(chatID)
		return
	}

	updated := false

	switch callback.Data {
	// Capital
	case CallbackSetCapital100_500:
		user.CapitalRange = "100-500"
		updated = true
	case CallbackSetCapital500_2000:
		user.CapitalRange = "500-2000"
		updated = true
	case CallbackSetCapital2000_5000:
		user.CapitalRange = "2000-5000"
		updated = true
	case CallbackSetCapital5000Plus:
		user.CapitalRange = "5000+"
		updated = true

	// Risk
	case CallbackSetRiskLow:
		user.RiskProfile = "low"
		updated = true
	case CallbackSetRiskMedium:
		user.RiskProfile = "medium"
		updated = true
	case CallbackSetRiskHigh:
		user.RiskProfile = "high"
		updated = true

	// Language
	case CallbackSetLanguageUK:
		user.LanguageCode = "uk"
		updated = true
	case CallbackSetLanguageEN:
		user.LanguageCode = "en"
		updated = true
	}

	if updated {
		if err := b.userRepo.Update(user); err != nil {
			log.Printf("Error updating user: %v", err)
			b.sendError(chatID)
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.sendMessage(deleteMsg)

		b.showSettingsMenu(chatID, user, prefs)
	}
}

func (b *Bot) getUserAndPrefs(userID int64) (*models.User, *models.UserPreferences) {
	user, _ := b.userRepo.GetByTelegramID(userID)
	prefs, _ := b.prefsRepo.GetByUserID(user.ID)

	return user, prefs
}
