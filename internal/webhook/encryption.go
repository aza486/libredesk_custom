package webhook

import (
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/webhook/models"
)

// encryptSecret encrypts webhook secret if present.
func (m *Manager) encryptSecret(secret string) (string, error) {
	encrypted, err := crypto.Encrypt(secret, m.encryptionKey)
	if err != nil {
		m.lo.Error("error encrypting webhook secret", "error", err)
		return "", err
	}
	return encrypted, nil
}

// Decrypt failures clear the secret so the app stays usable across encryption_key rotation.
func (m *Manager) decryptWebhook(webhook *models.Webhook) {
	if webhook.Secret == "" {
		return
	}
	decrypted, err := crypto.Decrypt(webhook.Secret, m.encryptionKey)
	if err != nil {
		m.lo.Error("error decrypting webhook secret, clearing field", "webhook_id", webhook.ID, "error", err)
		webhook.Secret = ""
		return
	}
	webhook.Secret = decrypted
}

func (m *Manager) decryptWebhooks(webhooks []models.Webhook) {
	for i := range webhooks {
		m.decryptWebhook(&webhooks[i])
	}
}
