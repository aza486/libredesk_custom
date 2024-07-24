package models

// CannedResponse represents a canned response with an ID, title, and content.
type CannedResponse struct {
	ID      string `db:"id" json:"id"`
	Title   string `db:"title" json:"title"`
	Content string `db:"content" json:"content"`
}
