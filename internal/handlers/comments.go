package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type CommentsHandler struct {
	db *database.DB
}

func NewCommentsHandler(db *database.DB) *CommentsHandler {
	return &CommentsHandler{db: db}
}

func (h *CommentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/activities/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	activityID := parts[0]
	endpoint := parts[1]

	// Check if this is a reaction request
	if endpoint == "react" {
		h.handleReaction(w, r, activityID)
		return
	}

	// Otherwise, handle as comment request
	switch r.Method {
	case http.MethodGet:
		h.getComments(w, r, activityID)
	case http.MethodPost:
		h.createComment(w, r, activityID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CommentsHandler) getComments(w http.ResponseWriter, r *http.Request, activityID string) {
	comments, err := h.db.GetCommentsByActivity(activityID)
	if err != nil {
		log.Printf("Error getting comments: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := templates.CommentsList(comments, activityID).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering comments: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *CommentsHandler) createComment(w http.ResponseWriter, r *http.Request, activityID string) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	if len(content) > 500 {
		http.Error(w, "Content too long (max 500 characters)", http.StatusBadRequest)
		return
	}

	var parentID *int
	if parentIDStr := r.FormValue("parent_id"); parentIDStr != "" {
		if pid, err := strconv.Atoi(parentIDStr); err == nil {
			parentID = &pid
		}
	}

	_, err := h.db.CreateComment(activityID, userID, parentID, content)
	if err != nil {
		log.Printf("Error creating comment: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.getComments(w, r, activityID)
}

func (h *CommentsHandler) handleReaction(w http.ResponseWriter, r *http.Request, activityID string) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data (HTMX sends hx-vals as form data by default)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	emoji := r.FormValue("emoji")
	if emoji == "" {
		http.Error(w, "Emoji is required", http.StatusBadRequest)
		return
	}

	validEmojis := map[string]bool{
		"🚀": true,
		"💎": true,
		"📈": true,
		"📉": true,
		"🔥": true,
		"👀": true,
		"🤔": true,
		"💰": true,
	}

	if !validEmojis[emoji] {
		http.Error(w, "Invalid emoji", http.StatusBadRequest)
		return
	}

	_, err := h.db.AddReaction(activityID, userID, emoji)
	if err != nil {
		log.Printf("Error toggling reaction: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	reactionCounts, err := h.db.GetReactionCounts(activityID)
	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	userReactions, err := h.db.GetUserReactionsForActivity(activityID, userID)
	if err != nil {
		log.Printf("Error getting user reactions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	activity := templates.ActivityFeedItem{
		ID:             activityID,
		ReactionCounts: reactionCounts,
		UserReactions:  userReactions,
	}

	if err := templates.ReactionButtons(activity).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering reactions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// CommentActionsHandler handles update and delete operations on individual comments
type CommentActionsHandler struct {
	db *database.DB
}

func NewCommentActionsHandler(db *database.DB) *CommentActionsHandler {
	return &CommentActionsHandler{db: db}
}

func (h *CommentActionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/comments/")
	commentID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	comment, err := h.db.GetCommentByID(commentID)
	if err != nil {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	if comment.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateComment(w, r, comment)
	case http.MethodDelete:
		h.deleteComment(w, r, comment)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CommentActionsHandler) updateComment(w http.ResponseWriter, r *http.Request, comment *database.Comment) {
	if time.Since(comment.CreatedAt) > 15*time.Minute {
		http.Error(w, "Edit window expired (15 minutes)", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	if len(content) > 500 {
		http.Error(w, "Content too long (max 500 characters)", http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateComment(comment.ID, content); err != nil {
		log.Printf("Error updating comment: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *CommentActionsHandler) deleteComment(w http.ResponseWriter, r *http.Request, comment *database.Comment) {
	if err := h.db.DeleteComment(comment.ID); err != nil {
		log.Printf("Error deleting comment: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
