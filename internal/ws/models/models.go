package models

// Action constants for WebSocket messages.
const (
	ActionConversationsSub                = "conversations_sub"
	ActionConversationSub                 = "conversation_sub"
	ActionConversationUnSub               = "conversation_unsub"
	MessageTypeNewMessage                 = "new_msg"
	MessageTypeMessagePropUpdate          = "msg_prop_update"
	MessageTypeNewConversation            = "new_conv"
	MessageTypeConversationPropertyUpdate = "conv_prop_update"
)

// IncomingReq represents an incoming WebSocket request.
type IncomingReq struct {
	Action string `json:"a"`
}

// ConversationsSubscribe represents a request to subscribe to conversations.
type ConversationsSubscribe struct {
	Type             string `json:"t"`
	PreDefinedFilter string `json:"pf"`
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
	UUIDs []string `json:"v"`
}

// Message represents a WebSocket message.
type Message struct {
	Type string      `json:"typ"`
	Data interface{} `json:"d"`
}
