package alpaca

import (
	"context"
)

type Activity struct {
	ID              string  `json:"id"`
	ActivityType    string  `json:"activity_type"`
	TransactionTime string  `json:"transaction_time"`
	Type            string  `json:"type"`
	Price           string  `json:"price"`
	Qty             string  `json:"qty"`
	Side            string  `json:"side"`
	Symbol          string  `json:"symbol"`
	LeavesQty       string  `json:"leaves_qty"`
	OrderID         string  `json:"order_id"`
	CumQty          string  `json:"cum_qty"`
	OrderStatus     string  `json:"order_status"`
}

// GetActivities retrieves account activities
func (c *Client) GetActivities(ctx context.Context) ([]Activity, error) {
	resp, err := c.doRequest(ctx, "GET", "/v2/account/activities", nil)
	if err != nil {
		return nil, err
	}

	var activities []Activity
	if err := c.decodeResponse(resp, &activities); err != nil {
		return nil, err
	}

	return activities, nil
}

// GetActivitiesByType retrieves account activities filtered by type
func (c *Client) GetActivitiesByType(ctx context.Context, activityType string) ([]Activity, error) {
	path := "/v2/account/activities/" + activityType

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var activities []Activity
	if err := c.decodeResponse(resp, &activities); err != nil {
		return nil, err
	}

	return activities, nil
}
