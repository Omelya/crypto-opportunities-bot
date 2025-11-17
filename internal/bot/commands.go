package bot

// Command константи
const (
	CommandStart        = "start"
	CommandHelp         = "help"
	CommandToday        = "today"
	CommandSettings     = "settings"
	CommandStats        = "stats"
	CommandPremium      = "premium"
	CommandBuyPremium   = "buy_premium"
	CommandSubscription = "subscription"
	CommandSupport      = "support"
	CommandArbitrage    = "arbitrage"
	CommandDeFi         = "defi"
)

// Callback data для inline buttons
const (
	CallbackLanguageUK = "lang_uk"
	CallbackLanguageRU = "lang_ru"
	CallbackLanguageEN = "lang_en"

	CallbackCapital100_500   = "capital_100_500"
	CallbackCapital500_2000  = "capital_500_2000"
	CallbackCapital2000_5000 = "capital_2000_5000"
	CallbackCapital5000Plus  = "capital_5000_plus"

	CallbackRiskLow    = "risk_low"
	CallbackRiskMedium = "risk_medium"
	CallbackRiskHigh   = "risk_high"

	CallbackPremiumTry     = "premium:premium_weekly"
	CallbackPremiumMonthly = "premium:premium_monthly"
	CallbackPremiumYearly  = "premium:premium_yearly"
	CallbackStayFree       = "stay_free"

	CallbackSkipCapital = "skip_capital"

	CallbackOppLaunchpool = "opp_launchpool"
	CallbackOppAirdrop    = "opp_airdrop"
	CallbackOppLearnEarn  = "opp_learn_earn"
	CallbackOppComplete   = "opp_complete"

	// Menu callbacks
	CallbackMenuToday    = "menu_today"
	CallbackMenuAll      = "menu_all"
	CallbackMenuSettings = "menu_settings"
	CallbackMenuStats    = "menu_stats"
	CallbackMenuPremium  = "menu_premium"

	// Opportunities filter callbacks
	CallbackFilterAll        = "filter_all"
	CallbackFilterLaunchpool = "filter_launchpool"
	CallbackFilterAirdrop    = "filter_airdrop"
	CallbackFilterLearnEarn  = "filter_learn_earn"
	CallbackFilterStaking    = "filter_staking"

	// Opportunity detail callbacks
	CallbackOppDetail = "opp_detail_"
	CallbackOppLink   = "opp_link_"
	CallbackOppIgnore = "opp_ignore_"

	// Pagination callbacks
	CallbackPageNext = "page_next_"
	CallbackPagePrev = "page_prev_"

	// Settings callbacks
	CallbackSettingsCapital   = "settings_capital"
	CallbackSettingsRisk      = "settings_risk"
	CallbackSettingsExchanges = "settings_exchanges"
	CallbackSettingsTypes     = "settings_types"
	CallbackSettingsLanguage  = "settings_language"
	CallbackSettingsDigest    = "settings_digest"
	CallbackSettingsBack      = "settings_back"

	// Settings - Capital selection
	CallbackSetCapital100_500   = "set_capital_100_500"
	CallbackSetCapital500_2000  = "set_capital_500_2000"
	CallbackSetCapital2000_5000 = "set_capital_2000_5000"
	CallbackSetCapital5000Plus  = "set_capital_5000_plus"

	// Settings - Risk selection
	CallbackSetRiskLow    = "set_risk_low"
	CallbackSetRiskMedium = "set_risk_medium"
	CallbackSetRiskHigh   = "set_risk_high"

	// Settings - Language selection
	CallbackSetLanguageUK = "set_lang_uk"
	CallbackSetLanguageEN = "set_lang_en"

	// Settings - Exchange toggles
	CallbackExchangeBinance = "exchange_binance"
	CallbackExchangeBybit   = "exchange_bybit"
	CallbackExchangeOKX     = "exchange_okx"
	CallbackExchangeGateIO  = "exchange_gateio"
	CallbackExchangeDone    = "exchange_done"

	// Settings - Type toggles
	CallbackTypeLaunchpool = "type_launchpool"
	CallbackTypeAirdrop    = "type_airdrop"
	CallbackTypeLearnEarn  = "type_learn_earn"
	CallbackTypeStaking    = "type_staking"
	CallbackTypeDone       = "type_done"

	// Settings - Digest
	CallbackDigestToggle = "digest_toggle"
	CallbackDigestDone   = "digest_done"
)
