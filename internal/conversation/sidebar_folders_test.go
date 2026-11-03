package conversation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
)

const folderTagFilter = `{"logic":"AND","rules":[{"logic":"AND","rules":[{"model":"conversations","field":"tags","operator":"contains","value":"[42]"}]}]}`

func TestSidebarAssignmentSQL(t *testing.T) {
	folders := []SidebarFolder{
		{ID: 11, ScopeType: "customer", Filters: json.RawMessage(`{"logic":"OR","rules":[{"model":"conversations","field":"subject","operator":"contains","value":"Rekla"},{"model":"conversation_statuses","field":"name","operator":"equals","value":"Open"}]}`)},
		{ID: 12, ScopeType: "customer", Filters: json.RawMessage(folderTagFilter)},
	}
	for _, id := range []int{-1, 11, 12} {
		clause, args, err := sidebarAssignmentClause(folders, id, []any{999}, "UTC")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(clause, "Rekla") || !strings.Contains(clause, "COALESCE((") || !strings.Contains(clause, "private") || !strings.Contains(clause, "personal") {
			t.Fatalf("unsafe clause %s", clause)
		}
		if !strings.Contains(clause, "$2") || args[0] != "%Rekla%" {
			t.Fatalf("incorrect binding: %s %#v", clause, args)
		}
		if id == 12 && (!strings.Contains(clause, "NOT (") || !strings.Contains(clause, " AND ")) {
			t.Fatal("missing higher-priority exclusion")
		}
		if id == 11 && strings.Contains(clause, "tag_id") && len(args) > 2 {
			t.Fatal("lower-priority folder affected winner")
		}
		if id == -1 && len(args) != 3 {
			t.Fatalf("standard list did not exclude all folders: %#v", args)
		}
	}
	if _, _, err := sidebarAssignmentClause(folders, 99, nil, "UTC"); err == nil {
		t.Fatal("missing folder accepted")
	}
}

func TestSidebarScopeBoundaries(t *testing.T) {
	for scope, fragment := range map[string]string{"customer": "customer_visibility", "team": "assigned_team_id=7", "personal": "inbox_id=7"} {
		sql := sidebarScopeSQL(scope, 7)
		if !strings.Contains(sql, fragment) {
			t.Fatal(sql)
		}
		if scope != "personal" && !strings.Contains(sql, "<>'personal'") {
			t.Fatal("mailbox boundary missing")
		}
	}
	if sidebarScopeSQL("other", 7) != "FALSE" {
		t.Fatal("unknown scope was allowed")
	}
}

func TestSidebarFilterValidation(t *testing.T) {
	for _, filters := range []string{
		`[{"model":"conversations","field":"id; DROP TABLE users","operator":"equals","value":"1"}]`,
		`[{"model":"conversations","field":"subject","operator":"starts with","value":"1"}]`,
		`{"logic":"XOR","rules":[{"model":"conversations","field":"subject","operator":"equals","value":"a"}]}`,
		`[{"model":"conversations","field":"hours_since_created","operator":"greater than","value":"NaN"}]`,
		`[{"model":"conversations","field":"hours_since_created","operator":"less than","value":"-1"}]`,
	} {
		if err := dbutil.ValidateFilters(filters, conversationListAllowedFields, conversationFilterRenderers); err == nil {
			t.Fatalf("accepted invalid filter %s", filters)
		}
	}
	filters := `[{"model":"conversations","field":"hours_since_created","operator":"greater than","value":"24"}]`
	sql, args, err := dbutil.BuildFilterClause(filters, nil, conversationListAllowedFields, conversationFilterRenderers, "UTC")
	if err != nil || !strings.Contains(sql, "CURRENT_TIMESTAMP") || args[0] != float64(24) {
		t.Fatalf("relative date: %s %#v %v", sql, args, err)
	}
}

func TestSidebarCompleteFilters(t *testing.T) {
	for _, raw := range []string{"", "[]", "null", `{}`, `{"logic":"AND","rules":[]}`, `[{"model":"conversations","field":"subject","operator":"contains","value":""}]`, `{"logic":"AND","rules":[{}, {"model":"conversations","field":"subject","operator":"contains","value":"abc"}]}`} {
		if err := validateCompleteFolderFilters(json.RawMessage(raw)); err == nil {
			t.Fatalf("incomplete filter accepted: %s", raw)
		}
	}
	if err := validateCompleteFolderFilters(json.RawMessage(folderTagFilter)); err != nil {
		t.Fatal(err)
	}
}
