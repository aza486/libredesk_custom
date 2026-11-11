package ws

import (
	"encoding/json"
	"github.com/abhinavxd/libredesk/internal/ws/models"
	"testing"

	"github.com/zerodha/logf"
)

func TestRemoveClient(t *testing.T) {
	newHubWith := func(n int) (*Hub, []*Client) {
		lo := logf.New(logf.Opts{})
		h := NewHub(&lo, nil)
		clients := make([]*Client, n)
		for i := range clients {
			clients[i] = &Client{ID: 7, Hub: h}
			h.AddClient(clients[i])
		}
		return h, clients
	}

	for n := 1; n <= 4; n++ {
		for remove := 0; remove < n; remove++ {
			h, clients := newHubWith(n)
			h.RemoveClient(clients[remove])

			got := h.clients[7]
			if len(got) != n-1 {
				t.Fatalf("n=%d remove=%d: want len %d, got %d", n, remove, n-1, len(got))
			}
			if n-1 == 0 {
				if _, ok := h.clients[7]; ok {
					t.Fatalf("n=%d remove=%d: expected map entry deleted", n, remove)
				}
				continue
			}
			// Order preserved and the removed client is gone.
			want := append(append([]*Client{}, clients[:remove]...), clients[remove+1:]...)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("n=%d remove=%d: order changed at %d", n, remove, i)
				}
			}
			// The slot past the new length must not still reference the removed client.
			full := got[:len(got)+1]
			if full[len(got)] != nil {
				t.Fatalf("n=%d remove=%d: stale pointer left in hidden slot", n, remove)
			}
		}
	}
}

func TestRemoveClientUnknown(t *testing.T) {
	lo := logf.New(logf.Opts{})
	h := NewHub(&lo, nil)
	a := &Client{ID: 1, Hub: h}
	h.AddClient(a)
	h.RemoveClient(&Client{ID: 1, Hub: h})
	if len(h.clients[1]) != 1 || h.clients[1][0] != a {
		t.Fatalf("removing an unregistered client must not change the slice: %v", h.clients[1])
	}
}

type accessTestConversations struct {
	allowed bool
	typing  int
}

func (s *accessTestConversations) FilterAuthorizedListUUIDs(_ int, uuids []string) ([]string, error) {
	if s.allowed {
		return uuids, nil
	}
	return nil, nil
}
func (s *accessTestConversations) BroadcastTypingToWidgetClientsOnly(string, bool) { s.typing++ }
func TestCustomerAccessSubscriptionAndTypingRechecked(t *testing.T) {
	lo := logf.New(logf.Opts{})
	hub := NewHub(&lo, nil)
	store := &accessTestConversations{allowed: true}
	hub.SetConversationStore(store)
	client := &Client{ID: 1, Hub: hub, Send: make(chan models.WSMessage, 10)}
	client.processIncomingMessage([]byte(`{"type":"conversation_subscribe","data":{"conversation_uuid":"known"}}`))
	if len(hub.ListSubscribers("known")) != 1 {
		t.Fatal("authorized subscription rejected")
	}
	store.allowed = false
	client.processIncomingMessage([]byte(`{"type":"conversation_subscribe","data":{"conversation_uuid":"known"}}`))
	if len(hub.ListSubscribers("known")) != 0 {
		t.Fatal("denied resubscription retained previous subscription")
	}
	var event struct {
		Data map[string]any `json:"data"`
	}
	json.Unmarshal((<-client.Send).Data, &event)
	if event.Data["access_revoked"] != true || len(event.Data) != 2 {
		t.Fatalf("nonminimal denial %v", event.Data)
	}
	client.processIncomingMessage([]byte(`{"type":"typing","data":{"conversation_uuid":"known","is_typing":true}}`))
	if store.typing != 0 {
		t.Fatal("revoked user could send widget typing")
	}
	store.allowed = true
	client.processIncomingMessage([]byte(`{"type":"conversation_subscribe","data":{"conversation_uuid":"known"}}`))
	client.processIncomingMessage([]byte(`{"type":"typing","data":{"conversation_uuid":"known","is_typing":true}}`))
	if len(hub.ListSubscribers("known")) != 1 || store.typing != 1 {
		t.Fatal("access did not recover")
	}
	store.allowed = false
	hub.BroadcastTypingToAllConversationClients("known", []byte("protected frame"))
	if len(client.Send) > 0 {
		t.Fatal("raw subscriber delivery bypassed authorization")
	}
}
