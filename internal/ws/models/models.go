package models

const (
	ActionConvSub           = "c_sub"
	ActionConvUnsub         = "c_unsub"
	ActionAssignedConvSub   = "a_c_sub"
	ActionAssignedConvUnSub = "a_c_unsub"

	EventNewMsg          = "new_msg"
	EventMsgStatusUpdate = "msg_status_update"
)

type IncomingReq struct {
	Action string `json:"a"`
}

type ConvSubUnsubReq struct {
	UUIDs []string `json:"v"`
}

type Event struct {
	Type string
	Data string
}
