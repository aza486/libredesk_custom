package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
)

type SidebarFolder struct {
	ID        int             `db:"id" json:"id"`
	Name      string          `db:"name" json:"name"`
	ScopeType string          `db:"scope_type" json:"scope_type"`
	ScopeID   int             `db:"scope_id" json:"scope_id"`
	Filters   json.RawMessage `db:"filters" json:"filters"`
	SortOrder int             `db:"sort_order" json:"sort_order"`
	ShowCount bool            `db:"show_count" json:"show_count"`
	CreatedBy *int            `db:"created_by" json:"created_by"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

func folderInputError(message string) error {
	return envelope.NewError(envelope.InputError, message, nil)
}
func folderDenied() error {
	return envelope.NewError(envelope.PermissionError, "Kein Zugriff auf diesen Ordnerbereich", nil)
}

func (c *Manager) authorizeFolderScope(q sqlx.Queryer, userID int, scope string, id int) error {
	user, err := c.userStore.GetAgent(userID, "")
	if err != nil {
		return err
	}
	has := func(p string) bool { return user.HasAdminRole() || slices.Contains(user.Permissions, p) }
	allowed := false
	switch scope {
	case "my_inbox":
		// Assignment-based inboxes differ per employee. Even administrators
		// cannot manage another employee's logical inbox through this API.
		allowed = id > 0 && id == userID && has("conversations:read_assigned")
	case "customer", "redistributed", "service_mail":
		allowed = id == 0 && has("conversations:read") && (user.HasAdminRole() || slices.Contains(user.Roles, "Kundensupport"))
	case "team":
		if id <= 0 || !has("conversations:read_team_all") {
			return folderDenied()
		}
		err = sqlx.Get(q, &allowed, `SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id=$1 AND user_id=$2)`, id, userID)
	case "personal":
		if id <= 0 || !has("conversations:read_team_all") {
			return folderDenied()
		}
		err = sqlx.Get(q, &allowed, `SELECT EXISTS(SELECT 1 FROM inboxes i JOIN personal_inbox_owners p ON p.inbox_id=i.id WHERE i.id=$1 AND p.user_id=$2 AND i.access_mode='personal' AND i.deleted_at IS NULL)`, id, userID)
	default:
		return folderInputError("Ungültiger Ordnerbereich")
	}
	if err != nil {
		return err
	}
	if !allowed {
		return folderDenied()
	}
	return nil
}

func (c *Manager) ListSidebarFolders(userID int, scope string, id int) ([]SidebarFolder, error) {
	if err := c.authorizeFolderScope(c.db, userID, scope, id); err != nil {
		return nil, err
	}
	return loadSidebarFolders(c.db, scope, id)
}
func loadSidebarFolders(q sqlx.Queryer, scope string, id int) ([]SidebarFolder, error) {
	folders := make([]SidebarFolder, 0)
	err := sqlx.Select(q, &folders, `SELECT * FROM sidebar_folders WHERE scope_type=$1 AND scope_id=$2 ORDER BY sort_order,id`, scope, id)
	return folders, err
}

// All definition mutations take the same per-scope transaction lock, including
// insertion/deletion, so a reorder can never overwrite a concurrently changed set.
func (c *Manager) MutateSidebarFolder(userID int, scope string, scopeID, id int, action string, folder SidebarFolder, ids []int) ([]SidebarFolder, error) {
	if err := c.authorizeFolderScope(c.db, userID, scope, scopeID); err != nil {
		return nil, err
	}
	tx, err := c.db.BeginTxx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(hashtext('sidebar_folders:' || $1),$2::int)`, scope, scopeID); err != nil {
		return nil, err
	}
	if err = c.authorizeFolderScope(tx, userID, scope, scopeID); err != nil {
		return nil, err
	}
	current, err := loadSidebarFolders(tx, scope, scopeID)
	if err != nil {
		return nil, err
	}
	if action == "create" || action == "update" {
		folder.Name = strings.TrimSpace(folder.Name)
		if n := utf8.RuneCountInString(folder.Name); n < 2 || n > 140 {
			return nil, folderInputError("Ordnername: 2 bis 140 Zeichen erforderlich")
		}
		if folder.SortOrder < 0 {
			return nil, folderInputError("Ungültige Priorität")
		}
		if err := validateCompleteFolderFilters(folder.Filters); err != nil {
			return nil, err
		}
		clause, values, err := dbutil.BuildFilterClause(string(folder.Filters), nil, conversationListAllowedFields, conversationFilterRenderers, c.filterLocation())
		if err != nil || clause == "" {
			return nil, folderInputError("Ungültige oder leere Filter")
		}
		// Ask PostgreSQL to type-check without reading or changing any rows.
		// The common compiler binds every value and allowlists every field.
		if _, err = tx.Exec(`EXPLAIN SELECT conversations.id FROM conversations JOIN users ON users.id=conversations.contact_id JOIN inboxes ON inboxes.id=conversations.inbox_id LEFT JOIN conversation_statuses ON conversation_statuses.id=conversations.status_id WHERE `+clause, values...); err != nil {
			return nil, folderInputError("Filterwerte oder Operatoren passen nicht zum ausgewählten Feld")
		}
	}
	if action == "create" {
		if len(current) >= 100 {
			return nil, folderInputError("Maximal 100 Ordner pro Bereich")
		}
		err = tx.QueryRow(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters,sort_order,show_count,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, folder.Name, scope, scopeID, folder.Filters, len(current), folder.ShowCount, userID).Scan(&id)
		if err != nil {
			return nil, err
		}
		current = append(current, SidebarFolder{ID: id})
	} else if action == "update" || action == "delete" {
		if !slices.ContainsFunc(current, func(f SidebarFolder) bool { return f.ID == id }) {
			return nil, envelope.NewError(envelope.NotFoundError, "Ordner nicht gefunden", nil)
		}
		if action == "update" {
			_, err = tx.Exec(`UPDATE sidebar_folders SET name=$1,filters=$2,show_count=$3,updated_at=NOW() WHERE id=$4 AND scope_type=$5 AND scope_id=$6`, folder.Name, folder.Filters, folder.ShowCount, id, scope, scopeID)
		} else {
			_, err = tx.Exec(`DELETE FROM sidebar_folders WHERE id=$1 AND scope_type=$2 AND scope_id=$3`, id, scope, scopeID)
		}
		if err != nil {
			return nil, err
		}
	} else if action != "reorder" {
		return nil, folderInputError("Ungültige Aktion")
	}
	if action == "reorder" {
		seen := map[int]bool{}
		if len(ids) != len(current) {
			return nil, folderInputError("Ordner wurden geändert. Bitte neu laden.")
		}
		for _, fid := range ids {
			if seen[fid] || !slices.ContainsFunc(current, func(f SidebarFolder) bool { return f.ID == fid }) {
				return nil, folderInputError("Ungültige Ordnerreihenfolge")
			}
			seen[fid] = true
		}
	} else {
		ids = make([]int, 0, len(current))
		for _, f := range current {
			if f.ID != id {
				ids = append(ids, f.ID)
			}
		}
		if action != "delete" {
			pos := min(folder.SortOrder, len(ids))
			ids = append(ids, 0)
			copy(ids[pos+1:], ids[pos:])
			ids[pos] = id
		}
	}
	for pos, fid := range ids {
		if _, err = tx.Exec(`UPDATE sidebar_folders SET sort_order=$1,updated_at=NOW() WHERE id=$2 AND scope_type=$3 AND scope_id=$4`, pos, fid, scope, scopeID); err != nil {
			return nil, err
		}
	}
	result, err := loadSidebarFolders(tx, scope, scopeID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func sidebarListScope(types []string, teams, inboxes []int, userID int) (string, int) {
	if len(types) != 1 {
		return "", 0
	}
	switch types[0] {
	case models.AssignedConversations, models.AssignedHighPriorityConversations, "folder-my_inbox":
		return "my_inbox", userID
	case models.ServiceMailConversations, "folder-service_mail":
		return "service_mail", 0
	case models.CustomerRedistributedConversations, models.CustomerRedistributedUnfiledConversations, "folder-redistributed":
		return "redistributed", 0
	case models.CustomerConversations, models.CustomerHighPriorityConversations, "folder-customer":
		return "customer", 0
	case models.TeamAllConversations, models.TeamHighPriorityConversations, models.TeamUnassignedConversations, "folder-team":
		if len(teams) == 1 {
			return "team", teams[0]
		}
	case models.PersonalConversations, models.PersonalHighPriorityConversations, "folder-personal":
		if len(inboxes) == 1 {
			return "personal", inboxes[0]
		}
	}
	return "", 0
}

// IDs originate from validated integers, never from user-supplied SQL.
func sidebarScopeSQL(scope string, id int) string {
	public := `(COALESCE(inboxes.access_mode,'public')<>'personal' AND COALESCE(conversations.custom_attributes->>'access_mode','public')<>'personal')`
	switch scope {
	case "my_inbox":
		// The caller reuses AssignedConversations, including its complete
		// access/owner exclusions. Definitions are loaded for the subject user;
		// do not introduce a second, potentially divergent assignment predicate.
		return "TRUE"
	case "redistributed":
		return "(" + sidebarScopeSQL("customer", 0) + " AND " + activeRedistributedSQL + ")"
	case "service_mail":
		return `(` + public + ` AND COALESCE((conversations.custom_attributes->>'private')::boolean,false)=false AND EXISTS(SELECT 1 FROM conversation_tags ct WHERE ct.conversation_id=conversations.id AND ct.tag_id=14))`
	case "customer":
		return `(` + public + ` AND COALESCE((conversations.custom_attributes->>'private')::boolean,false)=false AND COALESCE((conversations.custom_attributes->>'customer_visibility')::boolean,false)=true AND NOT EXISTS(SELECT 1 FROM conversation_tags ct WHERE ct.conversation_id=conversations.id AND ct.tag_id=14))`
	case "team":
		return fmt.Sprintf(`(%s AND conversations.assigned_team_id=%d)`, public, id)
	case "personal":
		return fmt.Sprintf(`(inboxes.access_mode='personal' AND conversations.inbox_id=%d)`, id)
	}
	return "FALSE"
}

// SQL three-valued logic matters: NULL does not match a folder, and must not
// remove a conversation from subsequent folders or standard categories.
func sidebarAssignmentClause(folders []SidebarFolder, folderID int, existing []any, location string) (string, []any, error) {
	parts := []string{}
	args := []any{}
	found := folderID < 0
	for _, folder := range folders {
		clause, values, err := dbutil.BuildFilterClause(string(folder.Filters), append(append([]any{}, existing...), args...), conversationListAllowedFields, conversationFilterRenderers, location)
		if err != nil || clause == "" {
			return "", nil, fmt.Errorf("invalid stored folder filters: %d", folder.ID)
		}
		args = append(args, values...)
		match := "(" + sidebarScopeSQL(folder.ScopeType, folder.ScopeID) + " AND COALESCE((" + clause + "),FALSE))"
		if folder.ID == folderID {
			parts = append(parts, match)
			found = true
			break
		}
		parts = append(parts, "NOT "+match)
	}
	if !found {
		return "", nil, folderInputError("Ordner nicht gefunden")
	}
	return strings.Join(parts, " AND "), args, nil
}

func (c *Manager) GetSidebarFolderConversations(userID int, scope string, scopeID, folderID int, order, orderBy string, page, pageSize int) ([]models.ConversationListItem, error) {
	if folderID <= 0 && !((scope == "redistributed" || scope == "service_mail") && folderID == -1) {
		return nil, folderInputError("Ungültiger Ordner")
	}
	if err := c.authorizeFolderScope(c.db, userID, scope, scopeID); err != nil {
		return nil, err
	}
	var teams, inboxes []int
	if scope == "team" {
		teams = []int{scopeID}
	}
	if scope == "personal" {
		inboxes = []int{scopeID}
	}
	return c.getFolderAwareConversations(userID, userID, teams, []string{"folder-" + scope}, order, orderBy, "[]", page, pageSize, folderID, inboxes...)
}

// SidebarFolderCounts uses one database statement per scope. CASE selects the
// first matching folder once for each authorized conversation, before grouping.
func (c *Manager) SidebarFolderCounts(userID int, scope string, scopeID int) (map[int]int, error) {
	tx, err := c.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = c.authorizeFolderScope(tx, userID, scope, scopeID); err != nil {
		return nil, err
	}
	folders, err := loadSidebarFolders(tx, scope, scopeID)
	if err != nil {
		return nil, err
	}
	result := map[int]int{}
	if len(folders) == 0 && scope != "redistributed" && scope != "service_mail" {
		return result, nil
	}
	user, err := c.userStore.GetAgent(userID, "")
	if err != nil {
		return nil, err
	}
	var teams, inboxes []int
	if scope == "team" {
		teams = []int{scopeID}
	}
	if scope == "personal" {
		inboxes = []int{scopeID}
	}
	// Compile scope/access first to reserve its parameter numbers.
	base := `SELECT conversations.id FROM conversations JOIN users ON users.id=conversations.contact_id JOIN inboxes ON inboxes.id=conversations.inbox_id LEFT JOIN conversation_statuses ON conversation_statuses.id=conversations.status_id LEFT JOIN conversation_priorities ON conversation_priorities.id=conversations.priority_id WHERE $1::bigint>0 AND ($2::boolean OR NOT $2::boolean) %s`
	query, args, err := c.makeScopedConversationsListQuery(userID, userID, teams, []string{"folder-" + scope}, base, "", "", 1, 1, "[]", user.HasAdminRole(), slices.Contains(user.Roles, "Kundensupport"), 0, nil, false, inboxes...)
	if err != nil {
		return nil, err
	}
	cases := []string{"CASE"}
	for _, folder := range folders {
		clause, values, err := dbutil.BuildFilterClause(string(folder.Filters), args, conversationListAllowedFields, conversationFilterRenderers, c.filterLocation())
		if err != nil {
			return nil, err
		}
		args = append(args, values...)
		cases = append(cases, fmt.Sprintf("WHEN (%s) IS TRUE THEN %d", clause, folder.ID))
		result[folder.ID] = 0
	}
	if scope == "redistributed" || scope == "service_mail" {
		cases = append(cases, "ELSE -1 END AS folder_id")
		result[-1] = 0
	} else {
		cases = append(cases, "ELSE NULL END AS folder_id")
	}
	caseSQL := strings.Join(cases, " ")
	if len(folders) == 0 {
		caseSQL = "-1 AS folder_id"
	}
	query = strings.Replace(query, "SELECT conversations.id", "SELECT "+caseSQL, 1)
	rows := []struct {
		ID    int `db:"folder_id"`
		Count int `db:"count"`
	}{}
	if err = tx.Select(&rows, `SELECT folder_id,COUNT(*) AS count FROM (`+query+`) matched WHERE folder_id IS NOT NULL GROUP BY folder_id`, args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.Count
	}
	return result, nil
}

// Relative hours are resolved at query time, so saved filters do not grow stale.
func renderHoursSinceCreated(operator, value string, paramIndex int) (string, []any, error) {
	ops := map[string]string{"equals": "=", "not equals": "<>", "greater than": ">", "less than": "<"}
	op, ok := ops[operator]
	if !ok {
		return "", nil, fmt.Errorf("unsupported relative-age operator")
	}
	hours, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) || hours < 0 || hours > 876000 {
		return "", nil, fmt.Errorf("invalid relative hours")
	}
	return fmt.Sprintf("(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP-conversations.created_at))/3600.0) %s $%d::numeric", op, paramIndex), []any{hours}, nil
}

func validateCompleteFolderFilters(raw json.RawMessage) error {
	if len(raw) > 65536 {
		return folderInputError("Filter sind zu groß")
	}
	var node dbutil.FilterNode
	if trimmed := strings.TrimSpace(string(raw)); len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(raw, &node.Rules); err != nil {
			return folderInputError("Ungültige Filter")
		}
		node.Logic = "AND"
	} else if err := json.Unmarshal(raw, &node); err != nil {
		return folderInputError("Ungültige Filter")
	}
	var complete func(dbutil.FilterNode) bool
	complete = func(n dbutil.FilterNode) bool {
		if n.Logic != "" || len(n.Rules) > 0 {
			if len(n.Rules) == 0 {
				return false
			}
			for _, child := range n.Rules {
				if !complete(child) {
					return false
				}
			}
			return true
		}
		return n.Model != "" && n.Field != "" && n.Operator != "" && (n.Operator == "set" || n.Operator == "not set" || strings.TrimSpace(n.Value) != "")
	}
	if !complete(node) {
		return folderInputError("Unvollständige Filterbedingungen")
	}
	return nil
}
