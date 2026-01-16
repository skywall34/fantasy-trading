package database

import (
	"database/sql"
	"time"
)

type Activity struct {
	ID              string
	UserID          int
	ActivityType    string
	AssetClass      sql.NullString
	Symbol          sql.NullString
	Side            sql.NullString
	Qty             sql.NullFloat64
	Price           sql.NullFloat64
	TransactionTime sql.NullTime
}

// UpsertActivity creates or updates an activity
func (db *DB) UpsertActivity(id string, userID int, activityType, assetClass, symbol, side string, qty, price float64, transactionTime time.Time) error {
	query := `
		INSERT INTO activities (id, user_id, activity_type, asset_class, symbol, side, qty, price, transaction_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			activity_type = excluded.activity_type,
			asset_class = excluded.asset_class,
			symbol = excluded.symbol,
			side = excluded.side,
			qty = excluded.qty,
			price = excluded.price,
			transaction_time = excluded.transaction_time
	`

	_, err := db.Exec(query, id, userID, activityType, assetClass, symbol, side, qty, price, transactionTime)
	return err
}

// GetActivitiesByUser retrieves activities for a specific user
func (db *DB) GetActivitiesByUser(userID int, limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, activity_type, asset_class, symbol, side, qty, price, transaction_time
		FROM activities
		WHERE user_id = ?
		ORDER BY transaction_time DESC
		LIMIT ?
	`

	rows, err := db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var act Activity
		err := rows.Scan(
			&act.ID,
			&act.UserID,
			&act.ActivityType,
			&act.AssetClass,
			&act.Symbol,
			&act.Side,
			&act.Qty,
			&act.Price,
			&act.TransactionTime,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, act)
	}

	return activities, nil
}

// GetUserRecentActivities retrieves recent activities for a specific user
func (db *DB) GetUserRecentActivities(userID int, limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, activity_type, asset_class, symbol, side, qty, price, transaction_time
		FROM activities
		WHERE user_id = ?
		ORDER BY transaction_time DESC
		LIMIT ?
	`

	rows, err := db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var act Activity
		err := rows.Scan(
			&act.ID,
			&act.UserID,
			&act.ActivityType,
			&act.AssetClass,
			&act.Symbol,
			&act.Side,
			&act.Qty,
			&act.Price,
			&act.TransactionTime,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, act)
	}

	return activities, nil
}

// GetRecentActivities retrieves recent activities across all users
func (db *DB) GetRecentActivities(limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, activity_type, asset_class, symbol, side, qty, price, transaction_time
		FROM activities
		ORDER BY transaction_time DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var act Activity
		err := rows.Scan(
			&act.ID,
			&act.UserID,
			&act.ActivityType,
			&act.AssetClass,
			&act.Symbol,
			&act.Side,
			&act.Qty,
			&act.Price,
			&act.TransactionTime,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, act)
	}

	return activities, nil
}
