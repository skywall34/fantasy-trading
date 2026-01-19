package database

import (
	"database/sql"
	"time"
)

// Helper functions to create sql.Null types
func NewNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func NewNullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: true}
}

func NewNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// Activity represents an activity from Alpaca (not stored in DB, used for type safety)
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
