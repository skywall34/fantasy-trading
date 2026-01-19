package handlers

import "github.com/skywall34/fantasy-trading/internal/database"

// getDisplayName returns the display name for a user in priority order:
// 1. Nickname (if set)
// 2. DisplayName (if set)
// 3. Email (if set)
// 4. "User" (fallback)
func getDisplayName(user *database.User) string {
	if user.Nickname.Valid {
		return user.Nickname.String
	}
	if user.DisplayName.Valid {
		return user.DisplayName.String
	}
	if user.Email.Valid {
		return user.Email.String
	}
	return "User"
}

// getInitials returns the initials for a user based on their display name
func getInitials(user *database.User) string {
	name := getDisplayName(user)
	if len(name) >= 2 {
		return name[:2]
	}
	return "U"
}
