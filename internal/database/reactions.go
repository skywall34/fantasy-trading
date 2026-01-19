package database

import (
	"time"
)

type Reaction struct {
	ID         int
	ActivityID string
	UserID     int
	Emoji      string
	CreatedAt  time.Time
}

// Toggle behavior: adds reaction if not present, removes if already present
func (db *DB) AddReaction(activityID string, userID int, emoji string) (bool, error) {
	existsQuery := `
		SELECT id FROM reactions
		WHERE activity_id = ? AND user_id = ? AND emoji = ?
	`

	var existingID int
	err := db.QueryRow(existsQuery, activityID, userID, emoji).Scan(&existingID)

	if err == nil {
		deleteQuery := `DELETE FROM reactions WHERE id = ?`
		_, err := db.Exec(deleteQuery, existingID)
		return false, err
	}

	insertQuery := `
		INSERT INTO reactions (activity_id, user_id, emoji)
		VALUES (?, ?, ?)
	`

	_, err = db.Exec(insertQuery, activityID, userID, emoji)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (db *DB) RemoveReaction(activityID string, userID int, emoji string) error {
	query := `
		DELETE FROM reactions
		WHERE activity_id = ? AND user_id = ? AND emoji = ?
	`

	_, err := db.Exec(query, activityID, userID, emoji)
	return err
}

func (db *DB) GetReactionsByActivity(activityID string) ([]Reaction, error) {
	query := `
		SELECT id, activity_id, user_id, emoji, created_at
		FROM reactions
		WHERE activity_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []Reaction
	for rows.Next() {
		var reaction Reaction
		err := rows.Scan(
			&reaction.ID,
			&reaction.ActivityID,
			&reaction.UserID,
			&reaction.Emoji,
			&reaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reactions = append(reactions, reaction)
	}

	return reactions, nil
}

func (db *DB) GetReactionCounts(activityID string) (map[string]int, error) {
	query := `
		SELECT emoji, COUNT(*) as count
		FROM reactions
		WHERE activity_id = ?
		GROUP BY emoji
	`

	rows, err := db.Query(query, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var emoji string
		var count int
		if err := rows.Scan(&emoji, &count); err != nil {
			return nil, err
		}
		counts[emoji] = count
	}

	return counts, nil
}

func (db *DB) GetReactionCountsForActivities(activityIDs []string) (map[string]map[string]int, error) {
	if len(activityIDs) == 0 {
		return make(map[string]map[string]int), nil
	}

	query := `
		SELECT activity_id, emoji, COUNT(*) as count
		FROM reactions
		WHERE activity_id IN (?` + generatePlaceholders(len(activityIDs)-1) + `)
		GROUP BY activity_id, emoji
	`

	args := make([]interface{}, len(activityIDs))
	for i, id := range activityIDs {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]int)
	for rows.Next() {
		var activityID, emoji string
		var count int
		if err := rows.Scan(&activityID, &emoji, &count); err != nil {
			return nil, err
		}

		if result[activityID] == nil {
			result[activityID] = make(map[string]int)
		}
		result[activityID][emoji] = count
	}

	return result, nil
}

func (db *DB) GetUserReactionsForActivity(activityID string, userID int) ([]string, error) {
	query := `
		SELECT emoji
		FROM reactions
		WHERE activity_id = ? AND user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, activityID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emojis []string
	for rows.Next() {
		var emoji string
		if err := rows.Scan(&emoji); err != nil {
			return nil, err
		}
		emojis = append(emojis, emoji)
	}

	return emojis, nil
}

func (db *DB) GetUserReactionsForActivities(activityIDs []string, userID int) (map[string][]string, error) {
	if len(activityIDs) == 0 {
		return make(map[string][]string), nil
	}

	query := `
		SELECT activity_id, emoji
		FROM reactions
		WHERE activity_id IN (?` + generatePlaceholders(len(activityIDs)-1) + `)
		AND user_id = ?
		ORDER BY created_at DESC
	`

	args := make([]interface{}, len(activityIDs)+1)
	for i, id := range activityIDs {
		args[i] = id
	}
	args[len(activityIDs)] = userID

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var activityID, emoji string
		if err := rows.Scan(&activityID, &emoji); err != nil {
			return nil, err
		}

		result[activityID] = append(result[activityID], emoji)
	}

	return result, nil
}
