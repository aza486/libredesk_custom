package notifier

// Notifier defines the interface for sending notifications.
type Notifier interface {
	SendMessage(userIDs []int, subject, content string) error
	SendAssignedConversationNotification(userIDs []int, convUUID string) error
}

// TemplateRenderer defines the interface for rendering templates.
type TemplateRenderer interface {
	RenderDefault(data interface{}) (subject, content string, err error)
}

// UserEmailFetcher defines the interface for fetching user email addresses.
type UserEmailFetcher interface {
	GetEmail(id int, uuid string) (string, error)
}

// UserStore defines the interface for the user store.
type UserStore interface {
	UserEmailFetcher
}
