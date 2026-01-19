package database

import (
	"database/sql"
	"time"
)

type Comment struct {
	ID         int
	ActivityID string
	UserID     int
	ParentID   sql.NullInt64
	Content    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CommentWithUser struct {
	Comment
	UserDisplayName string
	UserNickname    string
	UserAvatarURL   string
}

func (db *DB) CreateComment(activityID string, userID int, parentID *int, content string) (*Comment, error) {
	query := `
		INSERT INTO comments (activity_id, user_id, parent_id, content)
		VALUES (?, ?, ?, ?)
		RETURNING id, activity_id, user_id, parent_id, content, created_at, updated_at
	`

	var comment Comment
	var parentIDVal sql.NullInt64
	if parentID != nil {
		parentIDVal = sql.NullInt64{Int64: int64(*parentID), Valid: true}
	}

	err := db.QueryRow(query, activityID, userID, parentIDVal, content).Scan(
		&comment.ID,
		&comment.ActivityID,
		&comment.UserID,
		&comment.ParentID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (db *DB) GetCommentsByActivity(activityID string) ([]CommentWithUser, error) {
	query := `
		SELECT
			c.id, c.activity_id, c.user_id, c.parent_id, c.content, c.created_at, c.updated_at,
			u.display_name, u.nickname, u.avatar_url
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.activity_id = ?
		ORDER BY c.created_at ASC
	`

	rows, err := db.Query(query, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []CommentWithUser
	for rows.Next() {
		var comment CommentWithUser
		var displayName, nickname, avatarURL sql.NullString

		err := rows.Scan(
			&comment.ID,
			&comment.ActivityID,
			&comment.UserID,
			&comment.ParentID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&displayName,
			&nickname,
			&avatarURL,
		)
		if err != nil {
			return nil, err
		}

		comment.UserDisplayName = displayName.String
		comment.UserNickname = nickname.String
		comment.UserAvatarURL = avatarURL.String

		comments = append(comments, comment)
	}

	return comments, nil
}

func (db *DB) GetCommentByID(commentID int) (*Comment, error) {
	query := `
		SELECT id, activity_id, user_id, parent_id, content, created_at, updated_at
		FROM comments
		WHERE id = ?
	`

	var comment Comment
	err := db.QueryRow(query, commentID).Scan(
		&comment.ID,
		&comment.ActivityID,
		&comment.UserID,
		&comment.ParentID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (db *DB) UpdateComment(commentID int, content string) error {
	query := `
		UPDATE comments
		SET content = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.Exec(query, content, commentID)
	return err
}

func (db *DB) DeleteComment(commentID int) error {
	query := `DELETE FROM comments WHERE id = ?`
	_, err := db.Exec(query, commentID)
	return err
}

func (db *DB) GetCommentCount(activityID string) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE activity_id = ?`

	var count int
	err := db.QueryRow(query, activityID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (db *DB) GetCommentCountsForActivities(activityIDs []string) (map[string]int, error) {
	if len(activityIDs) == 0 {
		return make(map[string]int), nil
	}

	query := `
		SELECT activity_id, COUNT(*) as count
		FROM comments
		WHERE activity_id IN (?` + generatePlaceholders(len(activityIDs)-1) + `)
		GROUP BY activity_id
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

	counts := make(map[string]int)
	for rows.Next() {
		var activityID string
		var count int
		if err := rows.Scan(&activityID, &count); err != nil {
			return nil, err
		}
		counts[activityID] = count
	}

	return counts, nil
}

func generatePlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	placeholders := ""
	for i := 0; i < count; i++ {
		placeholders += ", ?"
	}
	return placeholders
}
