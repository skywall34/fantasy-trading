package database

import (
	"fmt"
	"time"
)

type Session struct {
	ID        string
	UserID    int
	APIKey    string
	APISecret string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateSession creates a new session with encrypted API keys
func (db *DB) CreateSession(sessionID string, userID int, apiKey, apiSecret string, expiresAt time.Time) (*Session, error) {
	// Encrypt API credentials
	encryptedKey, err := Encrypt(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	encryptedSecret, err := Encrypt(apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API secret: %w", err)
	}

	query := `
		INSERT INTO sessions (id, user_id, api_key, api_secret, expires_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, user_id, api_key, api_secret, expires_at, created_at
	`

	var session Session
	var encryptedKeyDB, encryptedSecretDB string
	err = db.QueryRow(query, sessionID, userID, encryptedKey, encryptedSecret, expiresAt).Scan(
		&session.ID,
		&session.UserID,
		&encryptedKeyDB,
		&encryptedSecretDB,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Decrypt for return (so caller gets plain text)
	session.APIKey = apiKey
	session.APISecret = apiSecret

	return &session, nil
}

// GetSessionByID retrieves a session by ID and decrypts API keys
func (db *DB) GetSessionByID(sessionID string) (*Session, error) {
	query := `
		SELECT id, user_id, api_key, api_secret, expires_at, created_at
		FROM sessions
		WHERE id = ?
	`

	var session Session
	var encryptedKey, encryptedSecret string
	err := db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&encryptedKey,
		&encryptedSecret,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Decrypt API credentials
	session.APIKey, err = Decrypt(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	session.APISecret, err = Decrypt(encryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API secret: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a session
func (db *DB) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := db.Exec(query, sessionID)
	return err
}

// IsExpired checks if a session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// GetAllActiveSessions retrieves all non-expired sessions
func (db *DB) GetAllActiveSessions() ([]Session, error) {
	query := `
		SELECT id, user_id, api_key, api_secret, expires_at, created_at
		FROM sessions
		WHERE expires_at > datetime('now')
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var encryptedKey, encryptedSecret string
		var session Session
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&encryptedKey,
			&encryptedSecret,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Decrypt API credentials
		session.APIKey, err = Decrypt(encryptedKey)
		if err != nil {
			continue // Skip sessions with decryption errors
		}

		session.APISecret, err = Decrypt(encryptedSecret)
		if err != nil {
			continue // Skip sessions with decryption errors
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// DeleteOldSessionsForUser deletes all sessions for a user except the most recent one
func (db *DB) DeleteOldSessionsForUser(userID int) error {
	query := `
		DELETE FROM sessions
		WHERE user_id = ?
		AND id NOT IN (
			SELECT id FROM sessions
			WHERE user_id = ?
			ORDER BY created_at DESC
			LIMIT 1
		)
	`
	_, err := db.Exec(query, userID, userID)
	return err
}
