package bot

// Command константи
const (
	CommandStart    = "start"
	CommandHelp     = "help"
	CommandToday    = "today"
	CommandSettings = "settings"
	CommandStats    = "stats"
	CommandPremium  = "premium"
	CommandSupport  = "support"
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

	CallbackPremiumTry = "premium_try"
	CallbackPremiumBuy = "premium_buy"
	CallbackStayFree   = "stay_free"

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
)
