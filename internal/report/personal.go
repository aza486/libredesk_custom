package report

import (
	"strings"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/user/models"
)

// Shadow the report sources with access-filtered CTEs so every aggregate,
// including SLA, CSAT and message volume, uses the same personal visibility.
func personalReportQuery(query string, viewer models.User) string {
	scope := `conversations AS (SELECT * FROM conversations c WHERE ` + authz.PersonalAccessSQL("c", viewer.ID, viewer.HasAdminRole()) + `),
 applied_slas AS (SELECT * FROM applied_slas WHERE conversation_id IN (SELECT id FROM conversations)),
 sla_events AS (SELECT * FROM sla_events WHERE applied_sla_id IN (SELECT id FROM applied_slas)),
 csat_responses AS (SELECT * FROM csat_responses WHERE conversation_id IN (SELECT id FROM conversations)),
 conversation_messages AS (SELECT * FROM conversation_messages WHERE conversation_id IN (SELECT id FROM conversations))`
	query = strings.TrimSpace(query)
	if strings.HasPrefix(query, "WITH ") {
		return "WITH " + scope + ", " + strings.TrimPrefix(query, "WITH ")
	}
	return "WITH " + scope + " " + query
}
