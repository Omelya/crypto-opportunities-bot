package notification

import (
	"crypto-opportunities-bot/internal/models"
	"time"
)

type Filter struct{}

func NewFilter() *Filter {
	return &Filter{}
}

func (f *Filter) ShouldNotify(user *models.User, prefs *models.UserPreferences, opp *models.Opportunity) bool {
	if !user.IsActive || user.IsBlocked {
		return false
	}

	if !opp.IsActive || opp.IsExpired() {
		return false
	}

	if f.isPremiumOpportunity(opp.Type) && !user.IsPremium() {
		return false
	}

	if !f.isTypeEnabled(opp.Type, prefs) {
		return false
	}

	if !f.isExchangeEnabled(opp.Exchange, prefs) {
		return false
	}

	if opp.EstimatedROI > 0 && opp.EstimatedROI < prefs.MinROI {
		return false
	}

	if prefs.MaxInvestment > 0 && opp.MinInvestment > float64(prefs.MaxInvestment) {
		return false
	}

	if !f.matchesCapitalRange(user.CapitalRange, opp.MinInvestment) {
		return false
	}

	if !f.matchesRiskProfile(user.RiskProfile, opp) {
		return false
	}

	return true
}

func (f *Filter) GetNotificationPriority(user *models.User, opp *models.Opportunity) string {
	if user.IsPremium() {
		return models.NotificationPriorityHigh
	}

	if opp.IsHighROI() {
		return models.NotificationPriorityHigh
	}

	if opp.DaysLeft() >= 0 && opp.DaysLeft() <= 2 {
		return models.NotificationPriorityHigh
	}

	return models.NotificationPriorityNormal
}

func (f *Filter) ShouldSendDailyDigest(user *models.User, prefs *models.UserPreferences) bool {
	if !user.IsActive || user.IsBlocked {
		return false
	}

	if !prefs.DailyDigestEnabled {
		return false
	}

	return f.isDigestTime(prefs.DailyDigestTime, user.Timezone)
}

func (f *Filter) CalculateDelay(user *models.User) time.Duration {
	if user.IsPremium() {
		return 0
	}

	return 20 * time.Minute
}

func (f *Filter) GetDailyAlertLimit(user *models.User) int {
	if user.IsPremium() {
		return 0
	}

	return 5
}

func (f *Filter) isPremiumOpportunity(oppType string) bool {
	premiumTypes := []string{
		models.OpportunityTypeArbitrage,
		models.OpportunityTypeDeFi,
	}

	for _, pt := range premiumTypes {
		if oppType == pt {
			return true
		}
	}

	return false
}

func (f *Filter) isTypeEnabled(oppType string, prefs *models.UserPreferences) bool {
	if len(prefs.OpportunityTypes) == 0 {
		return true
	}

	for _, enabledType := range prefs.OpportunityTypes {
		if enabledType == oppType {
			return true
		}
	}

	return false
}

func (f *Filter) isExchangeEnabled(exchange string, prefs *models.UserPreferences) bool {
	if len(prefs.Exchanges) == 0 {
		return true
	}

	for _, enabledExchange := range prefs.Exchanges {
		if enabledExchange == exchange {
			return true
		}
	}

	return false
}

func (f *Filter) matchesCapitalRange(capitalRange string, minInvestment float64) bool {
	if capitalRange == "" {
		return true
	}

	var maxCapital float64

	switch capitalRange {
	case "100-500":
		maxCapital = 500
	case "500-2000":
		maxCapital = 2000
	case "2000-5000":
		maxCapital = 5000
	case "5000+":
		maxCapital = 1000000
	default:
		return true
	}

	return minInvestment <= maxCapital*0.5
}

func (f *Filter) matchesRiskProfile(riskProfile string, opp *models.Opportunity) bool {
	if riskProfile == "" {
		return true
	}

	switch riskProfile {
	case "low":
		safeTypes := []string{
			models.OpportunityTypeLaunchpool,
			models.OpportunityTypeLearnEarn,
		}
		for _, st := range safeTypes {
			if opp.Type == st {
				return true
			}
		}
		return false

	case "high":
		return true

	default: // medium
		return opp.Type != models.OpportunityTypeDeFi
	}
}

func (f *Filter) isDigestTime(digestTime, timezone string) bool {
	// TODO: Врахувати timezone користувача
	// Зараз просто перевіряємо чи поточний час близький до вказаного

	now := time.Now()
	currentTime := now.Format("15:04")

	return currentTime >= digestTime && currentTime < addMinutes(digestTime, 30)
}

func addMinutes(timeStr string, minutes int) string {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return timeStr
	}

	t = t.Add(time.Duration(minutes) * time.Minute)
	return t.Format("15:04")
}
