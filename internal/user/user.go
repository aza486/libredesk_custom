// Package user managers all users in libredesk - agents and contacts.
package user

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"log"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	rmodels "github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
)

var (
	//go:embed queries.sql
	efs embed.FS

	minPassword     = 10
	maxPassword     = 72
	maxListPageSize = 100

	// ErrPasswordTooLong is returned when the password passed to
	// GenerateFromPassword is too long (i.e. > 72 bytes).
	ErrPasswordTooLong = errors.New("password length exceeds 72 bytes")

	PasswordHint = fmt.Sprintf("Password must be %d-%d characters long should contain at least one uppercase letter, one lowercase letter, one number, and one special character.", minPassword, maxPassword)
)

// Manager handles user-related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
	db   *sqlx.DB
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetUser               *sqlx.Stmt `query:"get-user"`
	GetUsers              string     `query:"get-users"`
	GetAgentsCompact      *sqlx.Stmt `query:"get-agents-compact"`
	UpdateContact         *sqlx.Stmt `query:"update-contact"`
	UpdateAgent           *sqlx.Stmt `query:"update-agent"`
	UpdateAvatar          *sqlx.Stmt `query:"update-avatar"`
	UpdateAvailability    *sqlx.Stmt `query:"update-availability"`
	UpdateLastActiveAt    *sqlx.Stmt `query:"update-last-active-at"`
	UpdateInactiveOffline *sqlx.Stmt `query:"update-inactive-offline"`
	UpdateLastLoginAt     *sqlx.Stmt `query:"update-last-login-at"`
	SoftDeleteAgent       *sqlx.Stmt `query:"soft-delete-agent"`
	SetUserPassword       *sqlx.Stmt `query:"set-user-password"`
	SetResetPasswordToken *sqlx.Stmt `query:"set-reset-password-token"`
	ResetPassword         *sqlx.Stmt `query:"reset-password"`
	InsertAgent           *sqlx.Stmt `query:"insert-agent"`
	InsertContact         *sqlx.Stmt `query:"insert-contact"`
}

// New creates and returns a new instance of the Manager.
func New(i18n *i18n.I18n, opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: i18n,
		db:   opts.DB,
	}, nil
}

// VerifyPassword authenticates an user by email and password, returning the user if successful.
func (u *Manager) VerifyPassword(email string, password []byte) (models.User, error) {
	var user models.User
	if err := u.q.GetUser.Get(&user, 0, email, models.UserTypeAgent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, envelope.NewError(envelope.InputError, u.i18n.T("user.invalidEmailPassword"), nil)
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil)
	}
	if err := u.verifyPassword(password, user.Password); err != nil {
		return user, envelope.NewError(envelope.InputError, u.i18n.T("user.invalidEmailPassword"), nil)
	}
	return user, nil
}

// GetAllAgents returns a list of all agents.
func (u *Manager) GetAgents() ([]models.User, error) {
	// Some dirty hack.
	return u.GetAllUsers(1, 999999999, models.UserTypeAgent, "desc", "users.updated_at", "")
}

// GetAllContacts returns a list of all contacts.
func (u *Manager) GetContacts(page, pageSize int, order, orderBy string, filtersJSON string) ([]models.User, error) {
	if pageSize > maxListPageSize {
		return nil, envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.pageTooLarge", "max", fmt.Sprintf("%d", maxListPageSize)), nil)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return u.GetAllUsers(page, pageSize, models.UserTypeContact, order, orderBy, filtersJSON)
}

// GetAllUsers returns a list of all users.
func (u *Manager) GetAllUsers(page, pageSize int, userType, order, orderBy string, filtersJSON string) ([]models.User, error) {
	query, qArgs, err := u.makeUserListQuery(page, pageSize, userType, order, orderBy, filtersJSON)
	if err != nil {
		u.lo.Error("error creating user list query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil)
	}

	// Start a read-only txn.
	tx, err := u.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		u.lo.Error("error starting read-only transaction", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil)
	}
	defer tx.Rollback()

	// Execute query
	var users = make([]models.User, 0)
	if err := tx.Select(&users, query, qArgs...); err != nil {
		u.lo.Error("error fetching users", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil)
	}

	return users, nil
}

// GetAgentsCompact returns a compact list of users with limited fields.
func (u *Manager) GetAgentsCompact() ([]models.User, error) {
	var users = make([]models.User, 0)
	if err := u.q.GetAgentsCompact.Select(&users); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users, nil
		}
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", u.i18n.P("globals.terms.user")), nil)
	}
	return users, nil
}

// CreateAgent creates a new agent user.
func (u *Manager) CreateAgent(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil)
	}
	user.Email = null.NewString(strings.TrimSpace(strings.ToLower(user.Email.String)), user.Email.Valid)
	if err := u.q.InsertAgent.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, pq.Array(user.Roles)).Scan(&user.ID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.GeneralError, u.i18n.T("user.sameEmailAlreadyExists"), nil)
		}
		u.lo.Error("error creating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// GetAgent retrieves an agent by ID.
func (u *Manager) GetAgent(id int, email string) (models.User, error) {
	return u.Get(id, email, models.UserTypeAgent)
}

// GetContact retrieves a contact by ID.
func (u *Manager) GetContact(id int, email string) (models.User, error) {
	return u.Get(id, email, models.UserTypeContact)
}

// Get retrieves an user by ID or email.
func (u *Manager) Get(id int, email, type_ string) (models.User, error) {
	var user models.User
	if err := u.q.GetUser.Get(&user, id, email, type_); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			u.lo.Error("user not found", "id", id, "error", err)
			return user, envelope.NewError(envelope.NotFoundError, u.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.user}"), nil)
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil)
	}
	return user, nil
}

// GetSystemUser retrieves the system user.
func (u *Manager) GetSystemUser() (models.User, error) {
	return u.Get(0, models.SystemUserEmail, models.UserTypeAgent)
}

// UpdateAvatar updates the user avatar.
func (u *Manager) UpdateAvatar(id int, path string) error {
	if _, err := u.q.UpdateAvatar.Exec(id, null.NewString(path, path != "")); err != nil {
		u.lo.Error("error updating user avatar", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// UpdateAgent updates an agent in the database, including their password if provided.
func (u *Manager) UpdateAgent(id int, user models.User) error {
	var (
		hashedPassword any
		err            error
	)

	// Set password?
	if user.NewPassword != "" {
		if IsStrongPassword(user.NewPassword) {
			return envelope.NewError(envelope.InputError, PasswordHint, nil)
		}
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(user.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			u.lo.Error("error generating bcrypt password", "error", err)
			return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
		}
		u.lo.Debug("setting new password for user", "user_id", id)
	}

	// Update user in the database.
	if _, err := u.q.UpdateAgent.Exec(id, user.FirstName, user.LastName, user.Email, pq.Array(user.Roles), user.AvatarURL, hashedPassword, user.Enabled, user.AvailabilityStatus); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// UpdateContact updates a contact in the database.
func (u *Manager) UpdateContact(id int, user models.User) error {
	if _, err := u.q.UpdateContact.Exec(id, user.FirstName, user.LastName, user.Email, user.AvatarURL, user.PhoneNumber, user.PhoneNumberCallingCode, user.Enabled); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.contact}"), nil)
	}
	return nil
}

// UpdateLastLoginAt updates the last login timestamp of an user.
func (u *Manager) UpdateLastLoginAt(id int) error {
	if _, err := u.q.UpdateLastLoginAt.Exec(id); err != nil {
		u.lo.Error("error updating user last login at", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// SoftDeleteAgent soft deletes an agent by ID.
func (u *Manager) SoftDeleteAgent(id int) error {
	// Disallow if user is system user.
	systemUser, err := u.GetSystemUser()
	if err != nil {
		return err
	}
	if id == systemUser.ID {
		return envelope.NewError(envelope.InputError, u.i18n.T("user.cannotDeleteSystemUser"), nil)
	}
	if _, err := u.q.SoftDeleteAgent.Exec(id); err != nil {
		u.lo.Error("error deleting user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// SetResetPasswordToken sets a reset password token for an user and returns the token.
func (u *Manager) SetResetPasswordToken(id int) (string, error) {
	token, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		u.lo.Error("error generating reset password token", "error", err)
		return "", envelope.NewError(envelope.GeneralError, u.i18n.T("user.errorGeneratingPasswordToken"), nil)
	}
	if _, err := u.q.SetResetPasswordToken.Exec(id, token); err != nil {
		u.lo.Error("error setting reset password token", "error", err)
		return "", envelope.NewError(envelope.GeneralError, u.i18n.T("user.errorGeneratingPasswordToken"), nil)
	}
	return token, nil
}

// ResetPassword sets a new password for an user.
func (u *Manager) ResetPassword(token, password string) error {
	if !IsStrongPassword(password) {
		return envelope.NewError(envelope.InputError, "Password is not strong enough, "+PasswordHint, nil)
	}
	// Hash password.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		u.lo.Error("error generating bcrypt password", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.password}"), nil)
	}
	rows, err := u.q.ResetPassword.Exec(passwordHash, token)
	if err != nil {
		u.lo.Error("error setting new password", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.password}"), nil)
	}
	if count, _ := rows.RowsAffected(); count == 0 {
		return envelope.NewError(envelope.InputError, u.i18n.T("user.resetPasswordTokenExpired"), nil)
	}
	return nil
}

// UpdateAvailability updates the availability status of an user.
func (u *Manager) UpdateAvailability(id int, status string) error {
	if _, err := u.q.UpdateAvailability.Exec(id, status); err != nil {
		u.lo.Error("error updating user availability", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// UpdateLastActive updates the last active timestamp of an user.
func (u *Manager) UpdateLastActive(id int) error {
	if _, err := u.q.UpdateLastActiveAt.Exec(id); err != nil {
		u.lo.Error("error updating user last active at", "error", err)
		return fmt.Errorf("updating user last active at: %w", err)
	}
	return nil
}

// MonitorAgentAvailability continuously checks for user activity and sets them offline if inactive for more than 5 minutes.
func (u *Manager) MonitorAgentAvailability(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			u.markInactiveAgentsOffline()
		case <-ctx.Done():
			return
		}
	}
}

// makeUserListQuery generates a query to fetch users based on the provided filters.
func (u *Manager) makeUserListQuery(page, pageSize int, typ, order, orderBy, filtersJSON string) (string, []interface{}, error) {
	var (
		baseQuery = u.q.GetUsers
		qArgs     []any
	)
	// Set the type of user to fetch.
	qArgs = append(qArgs, typ)
	return dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
	}, filtersJSON, dbutil.AllowedFields{
		"users": {"email", "created_at", "updated_at"},
	})
}

// markInactiveAgentsOffline sets agents offline if they have been inactive for more than 5 minutes.
func (u *Manager) markInactiveAgentsOffline() {
	if res, err := u.q.UpdateInactiveOffline.Exec(); err != nil {
		u.lo.Error("error setting users offline", "error", err)
	} else {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			u.lo.Info("set inactive users offline", "count", rows)
		}
	}
}

// verifyPassword compares the provided password with the stored password hash.
func (u *Manager) verifyPassword(pwd []byte, pwdHash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(pwdHash), pwd); err != nil {
		u.lo.Error("error verifying password", "error", err)
		return fmt.Errorf("error verifying password: %w", err)
	}
	return nil
}

// generatePassword generates a random password and returns its bcrypt hash.
func (u *Manager) generatePassword() ([]byte, error) {
	password, _ := stringutil.RandomAlphanumeric(70)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		u.lo.Error("error generating bcrypt password", "error", err)
		return nil, fmt.Errorf("generating bcrypt password: %w", err)
	}
	return bytes, nil
}

// ChangeSystemUserPassword updates the system user's password with a newly prompted one.
func ChangeSystemUserPassword(ctx context.Context, db *sqlx.DB) error {
	// Prompt for password and get hashed password
	hashedPassword, err := promptAndHashPassword(ctx)
	if err != nil {
		return err
	}

	// Update system user's password in the database.
	if err := updateSystemUserPassword(db, hashedPassword); err != nil {
		return fmt.Errorf("error updating system user password: %v", err)
	}
	fmt.Println("password updated successfully. Login with email 'System' and the new password.")
	return nil
}

// CreateSystemUser creates a system user with the provided password or a random one.
func CreateSystemUser(ctx context.Context, password string, db *sqlx.DB) error {
	var err error

	// Set random password if not provided.
	if password == "" {
		password, err = stringutil.RandomAlphanumeric(32)
		if err != nil {
			return fmt.Errorf("failed to generate system used password: %v", err)
		}
	} else {
		log.Print("using provided password for system user")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash system user password: %v", err)
	}

	_, err = db.Exec(`
		WITH sys_user AS (
			INSERT INTO users (email, type, first_name, last_name, password)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		)
		INSERT INTO user_roles (user_id, role_id)
		SELECT sys_user.id, roles.id 
		FROM sys_user, roles 
		WHERE roles.name = $6`,
		models.SystemUserEmail, models.UserTypeAgent, "System", "", hashedPassword, rmodels.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create system user: %v", err)
	}
	log.Print("system user created successfully. Use command 'libredesk --set-system-user-password' to set the password and login with email 'System'.")
	return nil
}

// IsStrongPassword checks if the password meets the required strength for system user.
func IsStrongPassword(password string) bool {
	if len(password) < minPassword || len(password) > maxPassword {
		return false
	}
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLowercase := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	// Matches special characters
	hasSpecial := regexp.MustCompile(`[\W_]`).MatchString(password)
	return hasUppercase && hasLowercase && hasNumber && hasSpecial
}

// promptAndHashPassword handles password input and validation, and returns the hashed password.
func promptAndHashPassword(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			fmt.Printf("Please set System user password (%s): ", PasswordHint)
			buffer := make([]byte, 256)
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				return nil, fmt.Errorf("error reading input: %v", err)
			}
			password := strings.TrimSpace(string(buffer[:n]))
			if IsStrongPassword(password) {
				// Hash the password using bcrypt.
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return nil, fmt.Errorf("failed to hash password: %v", err)
				}
				return hashedPassword, nil
			}
			fmt.Println("Password does not meet the strength requirements.")
		}
	}
}

// updateSystemUserPassword updates the password of the system user in the database.
func updateSystemUserPassword(db *sqlx.DB, hashedPassword []byte) error {
	_, err := db.Exec(`UPDATE users SET password = $1 WHERE email = $2`, hashedPassword, models.SystemUserEmail)
	if err != nil {
		return fmt.Errorf("failed to update system user password: %v", err)
	}
	return nil
}
