package models

// Action constants for WebSocket messages.
const (
	ActionConversationsSub                = "conversations_sub"
	ActionConversationSub                 = "conversation_sub"
	ActionConversationUnSub               = "conversation_unsub"
	MessageTypeMessagePropUpdate          = "message_prop_update"
	MessageTypeConversationPropertyUpdate = "conversation_prop_update"
	MessageTypeNewMessage                 = "new_message"
	MessageTypeNewConversation            = "new_conversation"
)

// IncomingReq represents an incoming WebSocket request.
type IncomingReq struct {
	Action string `json:"action"`
}

// ConversationsSubscribe represents a request to subscribe to conversations.
type ConversationsSubscribe struct {
	Type   string `json:"type"`
	Filter string `json:"filter"`
}

// ConversationSubscribe represents a request to subscribe to a single conversation.
type ConversationSubscribe struct {
	UUID string `json:"uuid"`
}

// ConversationUnsubscribe represents a request to unsubscribe from a single conversation.
type ConversationUnsubscribe struct {
	UUID string `json:"uuid"`
}

// ConvSubUnsubReq represents a request to subscribe or unsubscribe from multiple conversations.
type ConvSubUnsubReq struct {
	UUIDs []string `json:"value"`
}

// Message represents a WebSocket message.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
