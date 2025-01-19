package models

import "time"

type Conversation struct {
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UUID            string    `db:"uuid" json:"uuid"`
	ReferenceNumber string    `db:"reference_number" json:"reference_number"`
	Subject         string    `db:"subject" json:"subject"`
}

type Message struct {
	CreatedAt                   time.Time `db:"created_at" json:"created_at"`
	TextContent                 string    `db:"text_content" json:"text_content"`
	ConversationCreatedAt       time.Time `db:"conversation_created_at" json:"conversation_created_at"`
	ConversationUUID            string    `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber string    `db:"conversation_reference_number" json:"conversation_reference_number"`
}
