package database

import (
	"fmt"
)

// Follow represents a follow relationship between two users
type Follow struct {
	ID          int
	FollowerID  int
	FollowingID int
	CreatedAt   string
}

// FollowUser creates a follow relationship where followerID follows followingID
func (db *DB) FollowUser(followerID, followingID int) error {
	if followerID == followingID {
		return fmt.Errorf("users cannot follow themselves")
	}

	query := `INSERT INTO follows (follower_id, following_id) VALUES (?, ?)`
	_, err := db.Exec(query, followerID, followingID)
	if err != nil {
		return fmt.Errorf("failed to follow user: %w", err)
	}
	return nil
}

// UnfollowUser removes a follow relationship
func (db *DB) UnfollowUser(followerID, followingID int) error {
	query := `DELETE FROM follows WHERE follower_id = ? AND following_id = ?`
	result, err := db.Exec(query, followerID, followingID)
	if err != nil {
		return fmt.Errorf("failed to unfollow user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("follow relationship not found")
	}

	return nil
}

// IsFollowing checks if followerID is following followingID
func (db *DB) IsFollowing(followerID, followingID int) (bool, error) {
	query := `SELECT COUNT(*) FROM follows WHERE follower_id = ? AND following_id = ?`
	var count int
	err := db.QueryRow(query, followerID, followingID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check follow status: %w", err)
	}
	return count > 0, nil
}

// GetFollowers returns a list of user IDs who follow the given user
func (db *DB) GetFollowers(userID int) ([]int, error) {
	query := `SELECT follower_id FROM follows WHERE following_id = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get followers: %w", err)
	}
	defer rows.Close()

	var followers []int
	for rows.Next() {
		var followerID int
		if err := rows.Scan(&followerID); err != nil {
			return nil, fmt.Errorf("failed to scan follower: %w", err)
		}
		followers = append(followers, followerID)
	}

	return followers, nil
}

// GetFollowing returns a list of user IDs that the given user is following
func (db *DB) GetFollowing(userID int) ([]int, error) {
	query := `SELECT following_id FROM follows WHERE follower_id = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get following: %w", err)
	}
	defer rows.Close()

	var following []int
	for rows.Next() {
		var followingID int
		if err := rows.Scan(&followingID); err != nil {
			return nil, fmt.Errorf("failed to scan following: %w", err)
		}
		following = append(following, followingID)
	}

	return following, nil
}

// GetFollowerCount returns the number of followers for a user
func (db *DB) GetFollowerCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM follows WHERE following_id = ?`
	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get follower count: %w", err)
	}
	return count, nil
}

// GetFollowingCount returns the number of users that a user is following
func (db *DB) GetFollowingCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM follows WHERE follower_id = ?`
	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get following count: %w", err)
	}
	return count, nil
}
