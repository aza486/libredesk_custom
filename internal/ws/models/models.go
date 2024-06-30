package models

const (
	ActionConversationsSub                = "conversations_sub"
	ActionConversationSub                 = "conversation_sub"
	ActionConversationUnSub               = "conversation_unsub"
	MessageTypeNewMessage                 = "new_msg"
	MessageTypeMessagePropUpdate          = "msg_prop_update"
	MessageTypeNewConversation            = "new_conv"
	MessageTypeConversationPropertyUpdate = "conv_prop_update"
)

type IncomingReq struct {
	Action string `json:"a"`
}

type ConversationsSubscribe struct {
	Type             string `json:"t"`
	PreDefinedFilter string `json:"pf"`
}

type ConversationSubscribe struct {
	UUID string `json:"uuid"`
}
type ConversationUnsubscribe struct {
	UUID string `json:"uuid"`
}

type ConvSubUnsubReq struct {
	UUIDs []string `json:"v"`
}

type Message struct {
	Type string      `json:"typ"`
	Data interface{} `json:"d"`
}
