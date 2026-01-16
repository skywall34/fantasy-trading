package alpaca

import (
	"context"
)

type Position struct {
	AssetID            string  `json:"asset_id"`
	Symbol             string  `json:"symbol"`
	Exchange           string  `json:"exchange"`
	AssetClass         string  `json:"asset_class"`
	AssetMarginable    bool    `json:"asset_marginable"`
	Qty                string  `json:"qty"`
	AvgEntryPrice      string  `json:"avg_entry_price"`
	Side               string  `json:"side"`
	MarketValue        string  `json:"market_value"`
	CostBasis          string  `json:"cost_basis"`
	UnrealizedPL       string  `json:"unrealized_pl"`
	UnrealizedPLPC     string  `json:"unrealized_plpc"`
	UnrealizedIntradayPL string `json:"unrealized_intraday_pl"`
	UnrealizedIntradayPLPC string `json:"unrealized_intraday_plpc"`
	CurrentPrice       string  `json:"current_price"`
	LastdayPrice       string  `json:"lastday_price"`
	ChangeToday        string  `json:"change_today"`
	QtyAvailable       string  `json:"qty_available"`
}

// GetPositions retrieves all open positions
func (c *Client) GetPositions(ctx context.Context) ([]Position, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/positions", nil)
	if err != nil {
		return nil, err
	}

	var positions []Position
	if err := c.decodeResponse(resp, &positions); err != nil {
		return nil, err
	}

	return positions, nil
}

// GetPosition retrieves a specific position by symbol
func (c *Client) GetPosition(ctx context.Context, symbol string) (*Position, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/positions/"+symbol, nil)
	if err != nil {
		return nil, err
	}

	var position Position
	if err := c.decodeResponse(resp, &position); err != nil {
		return nil, err
	}

	return &position, nil
}
