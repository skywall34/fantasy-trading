package alpaca

import (
	"context"
)

type Account struct {
	ID                    string  `json:"id"`
	AccountNumber         string  `json:"account_number"`
	Status                string  `json:"status"`
	CryptoStatus          string  `json:"crypto_status"`
	Currency              string  `json:"currency"`
	BuyingPower           string  `json:"buying_power"`
	Cash                  string  `json:"cash"`
	PortfolioValue        string  `json:"portfolio_value"`
	PatternDayTrader      bool    `json:"pattern_day_trader"`
	TradingBlocked        bool    `json:"trading_blocked"`
	TransfersBlocked      bool    `json:"transfers_blocked"`
	AccountBlocked        bool    `json:"account_blocked"`
	CreatedAt             string  `json:"created_at"`
	TradeSuspendedByUser  bool    `json:"trade_suspended_by_user"`
	Multiplier            string  `json:"multiplier"`
	ShortingEnabled       bool    `json:"shorting_enabled"`
	Equity                string  `json:"equity"`
	LastEquity            string  `json:"last_equity"`
	LongMarketValue       string  `json:"long_market_value"`
	ShortMarketValue      string  `json:"short_market_value"`
	InitialMargin         string  `json:"initial_margin"`
	MaintenanceMargin     string  `json:"maintenance_margin"`
	LastMaintenanceMargin string  `json:"last_maintenance_margin"`
	DaytradeCount         int     `json:"daytrade_count"`
	BalanceAsOf           string  `json:"balance_asof"`
}

// GetAccount retrieves the account information
func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/account", nil)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := c.decodeResponse(resp, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// PortfolioHistory represents portfolio history data
type PortfolioHistory struct {
	Timestamp      []int64   `json:"timestamp"`
	Equity         []float64 `json:"equity"`
	ProfitLoss     []float64 `json:"profit_loss"`
	ProfitLossPct  []float64 `json:"profit_loss_pct"`
	BaseValue      float64   `json:"base_value"`
	Timeframe      string    `json:"timeframe"`
}

// GetPortfolioHistory retrieves portfolio history
func (c *Client) GetPortfolioHistory(ctx context.Context, period, timeframe string) (*PortfolioHistory, error) {
	path := "/v2/account/portfolio/history"
	if period != "" && timeframe != "" {
		path += "?period=" + period + "&timeframe=" + timeframe
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var history PortfolioHistory
	if err := c.decodeResponse(resp, &history); err != nil {
		return nil, err
	}

	return &history, nil
}
