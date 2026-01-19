package database

import (
	"database/sql"
	"time"
)

type User struct {
	ID              int
	AlpacaAccountID string
	Email           sql.NullString
	DisplayName     sql.NullString
	Nickname        sql.NullString
	AvatarURL       sql.NullString
	IsPublic        bool
	ShowAmounts     bool
	CreatedAt       time.Time
	LastSyncAt      sql.NullTime
}

// CreateUser creates a new user or returns existing user
func (db *DB) CreateUser(alpacaAccountID, email, displayName string) (*User, error) {
	query := `
		INSERT INTO users (alpaca_account_id, email, display_name)
		VALUES (?, ?, ?)
		ON CONFLICT(alpaca_account_id) DO UPDATE SET
			email = excluded.email,
			display_name = COALESCE(excluded.display_name, display_name)
		RETURNING id, alpaca_account_id, email, display_name, nickname, avatar_url, is_public, show_amounts, created_at, last_sync_at
	`

	var user User
	err := db.QueryRow(query, alpacaAccountID, email, displayName).Scan(
		&user.ID,
		&user.AlpacaAccountID,
		&user.Email,
		&user.DisplayName,
		&user.Nickname,
		&user.AvatarURL,
		&user.IsPublic,
		&user.ShowAmounts,
		&user.CreatedAt,
		&user.LastSyncAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAllPublicUsers retrieves all users with public profiles
func (db *DB) GetAllPublicUsers() ([]User, error) {
	query := `
		SELECT id, alpaca_account_id, email, display_name, nickname, avatar_url, is_public, show_amounts, created_at, last_sync_at
		FROM users
		WHERE is_public = 1
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.AlpacaAccountID,
			&user.Email,
			&user.DisplayName,
			&user.Nickname,
			&user.AvatarURL,
			&user.IsPublic,
			&user.ShowAmounts,
			&user.CreatedAt,
			&user.LastSyncAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(id int) (*User, error) {
	query := `
		SELECT id, alpaca_account_id, email, display_name, nickname, avatar_url, is_public, show_amounts, created_at, last_sync_at
		FROM users
		WHERE id = ?
	`

	var user User
	err := db.QueryRow(query, id).Scan(
		&user.ID,
		&user.AlpacaAccountID,
		&user.Email,
		&user.DisplayName,
		&user.Nickname,
		&user.AvatarURL,
		&user.IsPublic,
		&user.ShowAmounts,
		&user.CreatedAt,
		&user.LastSyncAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByAlpacaID retrieves a user by their Alpaca account ID
func (db *DB) GetUserByAlpacaID(alpacaAccountID string) (*User, error) {
	query := `
		SELECT id, alpaca_account_id, email, display_name, nickname, avatar_url, is_public, show_amounts, created_at, last_sync_at
		FROM users
		WHERE alpaca_account_id = ?
	`

	var user User
	err := db.QueryRow(query, alpacaAccountID).Scan(
		&user.ID,
		&user.AlpacaAccountID,
		&user.Email,
		&user.DisplayName,
		&user.Nickname,
		&user.AvatarURL,
		&user.IsPublic,
		&user.ShowAmounts,
		&user.CreatedAt,
		&user.LastSyncAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateLastSync updates the last sync timestamp for a user
func (db *DB) UpdateLastSync(userID int) error {
	query := `UPDATE users SET last_sync_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(query, userID)
	return err
}

// UpdateUserNickname updates the nickname for a user
func (db *DB) UpdateUserNickname(userID int, nickname sql.NullString) error {
	query := `UPDATE users SET nickname = ? WHERE id = ?`
	_, err := db.Exec(query, nickname, userID)
	return err
}

// SearchUsers searches for users by nickname, display name, or email
func (db *DB) SearchUsers(searchTerm string, limit int) ([]User, error) {
	query := `
		SELECT id, alpaca_account_id, email, display_name, nickname, avatar_url, is_public, show_amounts, created_at, last_sync_at
		FROM users
		WHERE is_public = 1
		AND (
			nickname LIKE '%' || ? || '%'
			OR display_name LIKE '%' || ? || '%'
			OR email LIKE '%' || ? || '%'
		)
		ORDER BY
			CASE
				WHEN nickname LIKE ? || '%' THEN 1
				WHEN display_name LIKE ? || '%' THEN 2
				WHEN email LIKE ? || '%' THEN 3
				ELSE 4
			END,
			created_at DESC
		LIMIT ?
	`

	rows, err := db.Query(query, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.AlpacaAccountID,
			&user.Email,
			&user.DisplayName,
			&user.Nickname,
			&user.AvatarURL,
			&user.IsPublic,
			&user.ShowAmounts,
			&user.CreatedAt,
			&user.LastSyncAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
