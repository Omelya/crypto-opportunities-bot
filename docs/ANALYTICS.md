# Analytics System Documentation

## Overview

The Analytics System provides comprehensive tracking and reporting of user behavior, platform metrics, and opportunity performance in the Crypto Opportunities Bot.

## Architecture

### Components

1. **Models** - Data structures for analytics
   - `UserAnalytics` - Aggregated user activity metrics
   - `DailyStats` - Platform-wide daily statistics
   - `OpportunityStats` - Per-opportunity performance metrics
   - `UserEngagement` - Daily user engagement tracking

2. **Repository** - Data access layer
   - `AnalyticsRepository` - CRUD operations for all analytics models

3. **Service** - Business logic layer
   - `AnalyticsService` - Tracking, calculation, and retrieval of analytics data

4. **Scheduler** - Automated tasks
   - Daily metrics calculation (runs at 3:00 AM UTC)

5. **API Handlers** - REST endpoints for analytics data
6. **Bot Handlers** - Telegram commands for viewing analytics

---

## Data Models

### UserAnalytics

Stores aggregated metrics for each user:

**Activity Metrics:**
- `ViewedOpportunities` - Total opportunities viewed
- `ClickedOpportunities` - Total clicks on opportunity links
- `ParticipatedOpportunities` - Total participations
- `IgnoredOpportunities` - Total ignored opportunities

**Engagement Metrics:**
- `TotalSessions` - Number of bot sessions
- `TotalTimeSpent` - Total time spent (seconds)
- `AverageSessionTime` - Average session duration
- `LastActivityAt` - Last activity timestamp
- `DaysSinceRegistration` - Days since user registered

**Conversion Metrics:**
- `ViewToClickRate` - Percentage of views that result in clicks
- `ClickToParticipateRate` - Percentage of clicks that result in participation
- `OverallConversionRate` - Overall conversion from view to participation

**Preferences:**
- `FavoriteExchanges` - Most used exchanges
- `FavoriteTypes` - Most interacted opportunity types

**Notifications:**
- `NotificationsReceived` - Total notifications received
- `NotificationsOpened` - Total notifications opened

**Revenue (Premium users):**
- `TotalRevenue` - Total revenue from user
- `LastPaymentAt` - Last payment date
- `SubscriptionDays` - Total subscription days

### DailyStats

Platform-wide daily statistics:

**User Metrics:**
- `ActiveUsers` - Users who performed any action
- `NewUsers` - New registrations
- `PremiumUsers` - Active premium subscribers
- `ReturnedUsers` - Users who returned after 7+ days

**Opportunity Metrics:**
- `TotalOpportunities` - Total active opportunities
- `NewOpportunities` - New opportunities added
- `ViewedOpportunities` - Total views
- `ClickedOpportunities` - Total clicks
- `ParticipatedOpportunities` - Total participations

**Notification Metrics:**
- `NotificationsSent` - Successfully sent
- `NotificationsFailed` - Failed to send
- `NotificationsOpened` - Opened by users

**Engagement:**
- `TotalSessions` - Total sessions
- `AverageSessionTime` - Average session duration

**Conversion & Retention:**
- `ConversionRate` - View to participation rate
- `RetentionRate` - 7-day retention rate

**Revenue:**
- `DailyRevenue` - Revenue generated
- `NewSubscriptions` - New premium subscriptions
- `ChurnedUsers` - Users who cancelled

**Top Performers:**
- `TopExchange` - Most popular exchange
- `TopOpportunity` - Most viewed opportunity
- `TopOpportunityType` - Most popular type

### OpportunityStats

Statistics for each opportunity:

**View Metrics:**
- `TotalViews` / `UniqueViews`

**Click Metrics:**
- `TotalClicks` / `UniqueClicks`

**Participation Metrics:**
- `TotalParticipations` / `UniqueParticipations`

**Conversion Metrics:**
- `ViewToClickRate`
- `ClickToParticipateRate`
- `OverallConversionRate`

**Time Metrics:**
- `AvgTimeToClick` - Average time from view to click
- `AvgTimeToParticipate` - Average time from view to participation

**Demographics:**
- `FreeUserViews` - Views from free users
- `PremiumUserViews` - Views from premium users

**Performance:**
- `PerformanceScore` (0-100) - Weighted score based on engagement

### UserEngagement

Daily engagement tracking per user:

**Session Metrics:**
- `SessionsCount` - Number of sessions
- `TimeSpent` - Total time spent (seconds)

**Activity:**
- `ActionsCount` - Total actions performed
- `OpportunitiesViewed/Clicked/Participated`
- `FirstActivityAt` / `LastActivityAt`

**Commands:**
- `CommandsUsed` - List of commands used

**Engagement Level:**
- `EngagementLevel` - "low", "medium", or "high"

---

## Service Methods

### Tracking

```go
// Track user action
TrackAction(userID uint, actionType string, opportunityID *uint, metadata map[string]interface{}) error

// Record user session
RecordSession(userID uint, duration int) error

// Record notification
RecordNotification(userID uint, success bool, opened bool) error

// Record payment
RecordPayment(userID uint, amount float64) error
```

### Retrieval

```go
// Get user analytics
GetUserAnalytics(userID uint) (*UserAnalytics, error)

// Get user engagement history
GetUserEngagementHistory(userID uint, days int) ([]*UserEngagement, error)

// Get daily stats range
GetDailyStatsRange(from, to time.Time) ([]*DailyStats, error)

// Get top opportunities
GetTopOpportunities(limit int) ([]*OpportunityStats, error)

// Get top users
GetTopUsers(limit int, orderBy string) ([]*UserAnalytics, error)

// Get platform summary
GetPlatformSummary() (*PlatformSummary, error)
```

### Calculation

```go
// Calculate daily metrics (runs via cron)
CalculateDailyMetrics() error
```

---

## API Endpoints

### User Analytics

```
GET /api/v1/analytics/users/:id
```

Returns analytics for a specific user.

**Response:**
```json
{
  "id": 123,
  "user_id": 456,
  "viewed_opportunities": 45,
  "clicked_opportunities": 30,
  "participated_opportunities": 12,
  "total_sessions": 25,
  "average_session_time": 180,
  "view_to_click_rate": 66.67,
  "overall_conversion_rate": 26.67,
  "favorite_exchanges": ["binance", "bybit"],
  "favorite_types": ["launchpool", "airdrop"]
}
```

---

### User Engagement

```
GET /api/v1/analytics/users/:id/engagement?days=7
```

Returns engagement history for a user.

**Parameters:**
- `days` (optional) - Number of days to retrieve (1-90, default: 7)

**Response:**
```json
{
  "user_id": 456,
  "days": 7,
  "engagements": [
    {
      "date": "2025-01-18",
      "sessions_count": 3,
      "time_spent": 540,
      "actions_count": 15,
      "engagement_level": "high"
    }
  ]
}
```

---

### Daily Statistics

```
GET /api/v1/analytics/daily?from=2025-01-01&to=2025-01-31
```

Returns daily platform statistics.

**Parameters:**
- `from` (optional) - Start date (YYYY-MM-DD)
- `to` (optional) - End date (YYYY-MM-DD)
- Default: last 30 days

**Response:**
```json
{
  "from": "2025-01-01",
  "to": "2025-01-31",
  "stats": [
    {
      "date": "2025-01-18",
      "active_users": 150,
      "new_users": 25,
      "premium_users": 30,
      "viewed_opportunities": 450,
      "conversion_rate": 15.5,
      "daily_revenue": 299.97
    }
  ]
}
```

---

### Platform Summary

```
GET /api/v1/analytics/summary
```

Returns overall platform statistics.

**Response:**
```json
{
  "today": {
    "active_users": 150,
    "new_users": 25,
    "viewed_opportunities": 450,
    "conversion_rate": 15.5
  },
  "weekly_active_users": 800,
  "weekly_new_users": 120,
  "weekly_revenue": 1499.85,
  "avg_daily_active_users": 114
}
```

---

### Top Opportunities

```
GET /api/v1/analytics/opportunities/top?limit=10
```

Returns top performing opportunities.

**Parameters:**
- `limit` (optional) - Number of results (1-100, default: 10)

**Response:**
```json
{
  "limit": 10,
  "opportunities": [
    {
      "opportunity_id": 123,
      "unique_views": 250,
      "unique_clicks": 180,
      "unique_participations": 90,
      "overall_conversion_rate": 36.0,
      "performance_score": 85.5
    }
  ]
}
```

---

### Top Users

```
GET /api/v1/analytics/users/top?order_by=participated&limit=10
```

Returns top users by different metrics.

**Parameters:**
- `order_by` (optional) - Sort metric: "viewed", "participated", "conversion", "engagement" (default: "participated")
- `limit` (optional) - Number of results (1-100, default: 10)

**Response:**
```json
{
  "order_by": "participated",
  "limit": 10,
  "users": [
    {
      "user_id": 456,
      "participated_opportunities": 45,
      "viewed_opportunities": 120,
      "overall_conversion_rate": 37.5,
      "total_sessions": 50
    }
  ]
}
```

---

## Telegram Commands

### User Commands

#### /stats
Shows detailed personal statistics for the user.

**Example Output:**
```
📊 Твоя статистика

Підписка: 💎 Premium
Реєстрація: 15.01.2025 (3 днів тому)

📈 Активність
• Переглянуто: 45 можливостей
• Кліків: 30
• Участі: 12
• Сесій: 25 (сер. 3 хв)

💹 Конверсія
• Перегляд → Клік: 66.7%
• Клік → Участь: 40.0%
• Загальна: 26.7%

⭐ Улюблені
• Типи: ["launchpool", "airdrop"]
• Біржі: ["binance", "bybit"]

📅 Активність за 7 днів
🟢 Jan 18: 15 дій, 9 хв
🟢 Jan 17: 12 дій, 7 хв
🟡 Jan 16: 8 дій, 5 хв
```

---

### Admin Commands

#### /analytics
Shows platform-wide analytics (admin only).

**Example Output:**
```
📊 Platform Summary

Today:
• Active Users: 150
• New Users: 25
• Opportunities Viewed: 450
• Conversion Rate: 15.50%
• Revenue: $299.97

This Week:
• Active Users: 800
• New Users: 120
• Notifications Sent: 5,420
• Revenue: $1,499.85
• Avg Daily Active: 114 users
```

**Interactive Buttons:**
- 🏆 Top Opportunities
- 👥 Top Users
- 📅 Daily Report
- 🔄 Refresh

---

## Automated Tasks

### Daily Metrics Calculation

**Schedule:** Every day at 3:00 AM UTC
**Task:** Calculate and update daily platform metrics

**What it does:**
- Counts active opportunities
- Calculates retention rates
- Updates conversion metrics
- Identifies top performers

---

## Event Tracking

The system automatically tracks the following events:

1. **Opportunity Views**
   - When user views opportunity details
   - Increments view counters
   - Updates user analytics

2. **Opportunity Clicks**
   - When user clicks on opportunity link
   - Tracks click-through rate
   - Records time to click

3. **Participations**
   - When user participates in opportunity
   - Calculates conversion rates
   - Updates performance scores

4. **Sessions**
   - Bot interaction sessions
   - Duration tracking
   - Engagement level calculation

5. **Commands**
   - Command usage tracking
   - Popular features identification

6. **Notifications**
   - Delivery status
   - Open rates
   - User preferences

7. **Payments**
   - Revenue tracking
   - Subscription metrics
   - Lifetime value

---

## Performance Metrics

### Opportunity Performance Score

Calculated using weighted formula:
- 30% - View to click rate
- 40% - Overall conversion rate
- 10% - Click to participate rate
- 20% - Total unique views (normalized)

Score ranges from 0-100.

### User Engagement Level

Determined by daily activity score:
- **High** (score ≥ 15): 3+ sessions, significant activity, participations
- **Medium** (score 7-14): Moderate activity
- **Low** (score < 7): Minimal activity

---

## Best Practices

### For Developers

1. **Always track user actions** - Call `analyticsService.TrackAction()` for all significant user interactions

2. **Use appropriate action types:**
   - `ActionTypeViewed` - User viewed opportunity
   - `ActionTypeClicked` - User clicked on link
   - `ActionTypeParticipated` - User participated
   - `ActionTypeIgnored` - User explicitly ignored

3. **Include metadata** - Add context to actions for better insights:
   ```go
   metadata := map[string]interface{}{
       "source": "notification",
       "device": "mobile",
   }
   ```

4. **Record sessions** - Track session duration for engagement metrics

5. **Handle errors gracefully** - Analytics failures shouldn't break main functionality

### For Data Analysis

1. **Use conversion funnels** - Analyze View → Click → Participate flow

2. **Segment users** - Compare free vs premium, active vs inactive

3. **Track trends** - Monitor daily/weekly changes in key metrics

4. **Identify drop-off points** - Find where users lose interest

5. **A/B testing** - Use metadata to track experiment results

---

## Troubleshooting

### Analytics not updating

1. Check if analytics service is initialized:
   ```bash
   # Look for this in logs
   ✅ Analytics service initialized
   ```

2. Verify database migration:
   ```bash
   # Check for analytics tables
   \dt user_analytics
   \dt daily_stats
   \dt opportunity_stats
   \dt user_engagement
   ```

3. Check scheduler is running:
   ```bash
   # Look for this in logs
   ✅ Analytics scheduler started
   ```

### Missing data

1. Ensure tracking calls are in place
2. Check for errors in logs
3. Verify user permissions
4. Run manual calculation:
   ```go
   analyticsScheduler.RunNow()
   ```

### Performance issues

1. Add database indexes if needed
2. Use pagination for large datasets
3. Cache frequently accessed data
4. Optimize date range queries

---

## Future Enhancements

- [ ] Real-time dashboard
- [ ] Custom report generation
- [ ] Data export (CSV, JSON)
- [ ] Advanced filtering
- [ ] Cohort analysis
- [ ] Predictive analytics
- [ ] A/B testing framework
- [ ] Revenue forecasting
- [ ] Churn prediction
- [ ] Recommendation engine

---

## References

- [CLAUDE.md](../CLAUDE.md) - Project architecture
- [API Documentation](./API.md) - REST API reference
- Models: `internal/models/user_analytics.go`, `daily_stats.go`, etc.
- Service: `internal/analytics/service.go`
- Repository: `internal/repository/analytics_repository.go`

---

**Last Updated:** 2025-11-18
**Version:** 1.0
