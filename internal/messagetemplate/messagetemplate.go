// Package messagetemplate stores inert, sanitized message snippets. It never
// renders template expressions or interacts with sending, signatures or AI.
package messagetemplate

import (
	"database/sql"
	"html"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/envelope"
	uuidutil "github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/microcosm-cc/bluemonday"
	"github.com/volatiletech/null/v9"
)

const MaxContentBytes = 131072

// The protected technical Kundensupport role cannot be renamed/deleted through
// role management. Require its explicit capability as well as live membership.
const actorSQL = `EXISTS(SELECT 1 FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE u.id=$1 AND u.type='agent' AND u.enabled AND u.deleted_at IS NULL AND r.permissions @> ARRAY['messages:write']::text[])`
const supportSQL = `EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='Kundensupport' AND r.permissions @> ARRAY['message_templates:customer_support']::text[]) AND EXISTS(SELECT 1 FROM conversations c JOIN inboxes i ON i.id=c.inbox_id WHERE c.uuid=NULLIF($2,'')::uuid AND i.access_mode='public' AND i.channel='email' AND i.enabled AND i.deleted_at IS NULL)`
const visibleSQL = `(` + actorSQL + ` AND ((scope='personal' AND owner_id=$1) OR (scope='customer_support' AND ` + supportSQL + `)))`

// Access context is resolved from an authorized conversation by the API, never
// an inbox ID supplied by the browser. SQL rechecks grants and inbox state.
type Access struct {
	UserID           int
	ConversationUUID string
}
type Template struct {
	ID        int       `db:"id" json:"id"`
	UUID      string    `db:"uuid" json:"uuid"`
	Name      string    `db:"name" json:"name"`
	Content   string    `db:"content" json:"content,omitempty"`
	Scope     string    `db:"scope" json:"scope"`
	OwnerID   null.Int  `db:"owner_id" json:"owner_id"`
	CreatedBy null.Int  `db:"created_by" json:"created_by"`
	UpdatedBy null.Int  `db:"updated_by" json:"updated_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Revision  int64     `db:"revision" json:"revision"`
}
type Input struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Scope    string `json:"scope"`
	Revision int64  `json:"revision"`
}
type List struct {
	Templates  []Template `json:"templates"`
	CanSupport bool       `json:"can_customer_support"`
}
type Manager struct{ db *sqlx.DB }

func New(db *sqlx.DB) *Manager { return &Manager{db: db} }

var policy = func() *bluemonday.Policy {
	// Reuse the application's HTML sanitizer library with a text/link-only policy.
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "u", "ul", "ol", "li", "a")
	p.AllowAttrs("href", "title").OnElements("a")
	p.AllowAttrs("start").Matching(bluemonday.Integer).OnElements("ol")
	p.AllowURLSchemes("https", "http", "mailto")
	p.RequireParseableURLs(true)
	p.RequireNoFollowOnLinks(true)
	return p
}()
var plain = bluemonday.StrictPolicy()

func Validate(in Input) (Input, error) {
	in.Name = strings.TrimFunc(in.Name, func(r rune) bool { return unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) })
	if !utf8.ValidString(in.Name) || utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 160 || strings.IndexFunc(in.Name, unicode.IsControl) >= 0 {
		return in, envelope.NewError(envelope.InputError, "Der Vorlagenname muss 1–160 Zeichen enthalten.", nil)
	}
	if in.Scope != "personal" && in.Scope != "customer_support" {
		return in, envelope.NewError(envelope.InputError, "Ungültiger Geltungsbereich.", nil)
	}
	if len(in.Content) > MaxContentBytes || !utf8.ValidString(in.Content) || strings.ContainsRune(in.Content, 0) {
		return in, envelope.NewError(envelope.InputError, "Der Vorlageninhalt ist zu groß oder ungültig (maximal 128 KiB).", nil)
	}
	in.Content = policy.Sanitize(in.Content)
	if len(in.Content) > MaxContentBytes {
		return in, envelope.NewError(envelope.InputError, "Der bereinigte Vorlageninhalt ist zu groß (maximal 128 KiB).", nil)
	}
	text := html.UnescapeString(plain.Sanitize(in.Content))
	text = strings.TrimFunc(text, func(r rune) bool { return unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) })
	if text == "" {
		return in, envelope.NewError(envelope.InputError, "Die Vorlage benötigt einen Textinhalt.", nil)
	}
	return in, nil
}
func notFound() error {
	return envelope.NewError(envelope.NotFoundError, "Vorlage nicht gefunden oder nicht mehr verfügbar.", nil)
}
func (m *Manager) List(a Access) (List, error) {
	result := List{Templates: make([]Template, 0)}
	var active bool
	if err := m.db.Get(&active, `SELECT `+actorSQL, a.UserID); err != nil {
		return result, err
	}
	if !active {
		return result, envelope.NewError(envelope.PermissionError, "Keine Berechtigung für Nachrichtenvorlagen.", nil)
	}
	if err := m.db.Get(&result.CanSupport, `SELECT `+supportSQL, a.UserID, a.ConversationUUID); err != nil {
		return result, err
	}
	err := m.db.Select(&result.Templates, `SELECT id,uuid,name,scope,owner_id,created_by,updated_by,created_at,updated_at,revision FROM message_templates WHERE `+visibleSQL+` ORDER BY lower(name),id`, a.UserID, a.ConversationUUID)
	return result, err
}
func (m *Manager) Get(a Access, uuid string) (Template, error) {
	var t Template
	if _, err := uuidutil.Parse(uuid); err != nil {
		return t, notFound()
	}
	err := m.db.Get(&t, `SELECT * FROM message_templates WHERE uuid=$3::uuid AND `+visibleSQL, a.UserID, a.ConversationUUID, uuid)
	if err == sql.ErrNoRows {
		err = notFound()
	}
	return t, err
}
func (m *Manager) Create(a Access, in Input) (Template, error) {
	var t Template
	in, err := Validate(in)
	if err != nil {
		return t, err
	}
	err = m.db.Get(&t, `INSERT INTO message_templates(name,content,scope,owner_id,created_by,updated_by) SELECT $3,$4,$5,CASE WHEN $5='personal' THEN $1::bigint ELSE NULL END,$1,$1 WHERE `+actorSQL+` AND ($5='personal' OR (`+supportSQL+`)) RETURNING *`, a.UserID, a.ConversationUUID, in.Name, in.Content, in.Scope)
	if err == sql.ErrNoRows {
		err = envelope.NewError(envelope.PermissionError, "Keine Berechtigung für diesen Vorlagenbereich.", nil)
	}
	return t, err
}
func (m *Manager) Update(a Access, uuid string, in Input) (Template, error) {
	var t Template
	in, err := Validate(in)
	if err != nil {
		return t, err
	}
	old, err := m.Get(a, uuid)
	if err != nil {
		return t, err
	}
	if old.Scope != in.Scope {
		return t, envelope.NewError(envelope.InputError, "Der Geltungsbereich einer Vorlage kann nicht geändert werden.", nil)
	}
	err = m.db.Get(&t, `UPDATE message_templates SET name=$4,content=$5,updated_by=$1,updated_at=clock_timestamp(),revision=revision+1 WHERE uuid=$3::uuid AND revision=$6 AND `+visibleSQL+` RETURNING *`, a.UserID, a.ConversationUUID, uuid, in.Name, in.Content, in.Revision)
	if err == sql.ErrNoRows {
		err = m.conflict(a, uuid)
	}
	return t, err
}
func (m *Manager) Delete(a Access, uuid string, revision int64) error {
	if _, err := uuidutil.Parse(uuid); err != nil {
		return notFound()
	}
	res, err := m.db.Exec(`DELETE FROM message_templates WHERE uuid=$3::uuid AND revision=$4 AND `+visibleSQL, a.UserID, a.ConversationUUID, uuid, revision)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return m.conflict(a, uuid)
	}
	return nil
}
func (m *Manager) conflict(a Access, uuid string) error {
	if _, err := m.Get(a, uuid); err != nil {
		return err
	}
	return envelope.NewError(envelope.ConflictError, "Diese Vorlage wurde zwischenzeitlich geändert. Bitte neu laden und Änderungen vergleichen.", nil)
}
