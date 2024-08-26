package notifier

import (
	"context"
	"fmt"
	"sync"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/zerodha/logf"
)

const (
	ProviderEmail = "email"
)

// NotificationMessage represents a message to be sent as a notification.
type NotificationMessage struct {
	// Recipients of the message
	UserIDs []int
	// Subject of the message
	Subject string
	// Body of the message
	Content string
	// Provider to send the message through
	Provider string
	// Attachments to be sent with the message
	Attachments []attachment.Attachment
	// Type of content ("plain" or "html")
	ContentType string
	// Alternative plain text version of the HTML content
	AltContent string
	// Additional email headers
	Headers map[string][]string
}

// TemplateRenderer defines the interface for rendering email templates.
type TemplateRenderer interface {
	RenderDefault(data map[string]string) (content string, err error)
}

// UserEmailFetcher defines the interface for fetching user email addresses.
type UserEmailFetcher interface {
	GetEmail(id int) (string, error)
}

// UserStore defines the interface for the user store.
type UserStore interface {
	UserEmailFetcher
}

// Notifier defines the interface for sending notifications through various providers.
type Notifier interface {
	// Sends the notification message using the specified provider
	Send(message NotificationMessage) error
	// Returns the name of the provider
	Name() string
}

// Service manages message providers and a worker pool.
type Service struct {
	providers      map[string]Notifier
	messageChannel chan NotificationMessage
	concurrency    int
	wg             sync.WaitGroup
	lo             *logf.Logger
}

// NewService initializes the Service with given concurrency, channel capacity, and logger.
func NewService(providers map[string]Notifier, concurrency, capacity int, logger *logf.Logger) *Service {
	return &Service{
		providers:      providers,
		messageChannel: make(chan NotificationMessage, capacity),
		concurrency:    concurrency,
		lo:             logger,
	}
}

// Send sends a message to the message channel.
func (s *Service) Send(message NotificationMessage) error {
	select {
	case s.messageChannel <- message:
		return nil
	default:
		s.lo.Error("message channel is full")
		return fmt.Errorf("message channel is full")
	}
}

// Run starts the worker pool to process messages with context for cancellation.
func (s *Service) Run(ctx context.Context) {
	for i := 0; i < s.concurrency; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}

	s.wg.Wait()
}

// worker processes messages from the channel until context is canceled.
func (s *Service) worker(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			s.lo.Info("worker exiting due to context cancellation")
			return
		case message := <-s.messageChannel:
			sender, exists := s.providers[message.Provider]
			if !exists {
				s.lo.Error("unsupported provider", "provider", message.Provider)
				continue
			}

			if err := sender.Send(message); err != nil {
				s.lo.Error("error sending message", "error", err)
			}
		}
	}
}

// Stop closes the message channel.
func (s *Service) Stop() {
	close(s.messageChannel)
}
