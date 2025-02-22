// Package user handles user login, logout and provides functions to fetch user details.
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

const (
	systemUserEmail       = "System"
	minSystemUserPassword = 8
	maxSystemUserPassword = 50
	UserTypeAgent         = "agent"
	UserTypeContact       = "contact"
)

var (
	//go:embed queries.sql
	efs embed.FS

	// ErrPasswordTooLong is returned when the password passed to
	// GenerateFromPassword is too long (i.e. > 72 bytes).
	ErrPasswordTooLong = errors.New("password length exceeds 72 bytes")

	 SystemUserPasswordHint = fmt.Sprintf("Password must be %d-%d characters long and contain at least one uppercase letter and one number", minSystemUserPassword, maxSystemUserPassword)
)

// Manager handles user-related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetUsers              *sqlx.Stmt `query:"get-users"`
	GetUserCompact        *sqlx.Stmt `query:"get-users-compact"`
	GetUser               *sqlx.Stmt `query:"get-user"`
	GetEmail              *sqlx.Stmt `query:"get-email"`
	GetPermissions        *sqlx.Stmt `query:"get-permissions"`
	GetUserByEmail        *sqlx.Stmt `query:"get-user-by-email"`
	UpdateUser            *sqlx.Stmt `query:"update-user"`
	UpdateAvatar          *sqlx.Stmt `query:"update-avatar"`
	SoftDeleteUser        *sqlx.Stmt `query:"soft-delete-user"`
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
	}, nil
}

// VerifyPassword authenticates a user by email and password.
func (u *Manager) VerifyPassword(email string, password []byte) (models.User, error) {
	var user models.User

	if err := u.q.GetUserByEmail.Get(&user, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, envelope.NewError(envelope.InputError, u.i18n.T("user.invalidEmailPassword"), nil)
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.user}"), nil)
	}

	if err := u.verifyPassword(password, user.Password); err != nil {
		return user, envelope.NewError(envelope.InputError, u.i18n.T("user.invalidEmailPassword"), nil)
	}

	return user, nil
}

// GetAll retrieves all users.
func (u *Manager) GetAll() ([]models.User, error) {
	var users = make([]models.User, 0)
	if err := u.q.GetUsers.Select(&users); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users, nil
		}
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, "Error fetching users", nil)
	}

	return users, nil
}

// GetAllCompact returns a compact list of users with limited fields.
func (u *Manager) GetAllCompact() ([]models.User, error) {
	var users = make([]models.User, 0)
	if err := u.q.GetUserCompact.Select(&users); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users, nil
		}
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, "Error fetching users", nil)
	}

	return users, nil
}

// CreateAgent creates a new agent user.
func (u *Manager) CreateAgent(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating user", nil)
	}
	user.Email = null.NewString(strings.TrimSpace(strings.ToLower(user.Email.String)), user.Email.Valid)
	if err := u.q.InsertAgent.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, pq.Array(user.Roles)).Scan(&user.ID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.GeneralError, "User with the same email already exists", nil)
		}
		u.lo.Error("error creating user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating user", nil)
	}
	return nil
}

// Get retrieves a user by ID.
func (u *Manager) Get(id int) (models.User, error) {
	var user models.User
	if err := u.q.GetUser.Get(&user, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			u.lo.Error("user not found", "id", id, "error", err)
			return user, envelope.NewError(envelope.GeneralError, "User not found", nil)
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, envelope.NewError(envelope.GeneralError, "Error fetching user", nil)
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (u *Manager) GetByEmail(email string) (models.User, error) {
	var user models.User
	if err := u.q.GetUserByEmail.Get(&user, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, envelope.NewError(envelope.GeneralError, "User not found", nil)
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, envelope.NewError(envelope.GeneralError, "Error fetching user", nil)
	}
	return user, nil
}

// GetSystemUser retrieves the system user.
func (u *Manager) GetSystemUser() (models.User, error) {
	return u.GetByEmail(systemUserEmail)
}

// UpdateAvatar updates the user avatar.
func (u *Manager) UpdateAvatar(id int, avatar string) error {
	if _, err := u.q.UpdateAvatar.Exec(id, null.NewString(avatar, avatar != "")); err != nil {
		u.lo.Error("error updating user avatar", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating avatar", nil)
	}
	return nil
}

// Update updates a user.
func (u *Manager) Update(id int, user models.User) error {
	var (
		hashedPassword interface{}
		err            error
	)

	if user.NewPassword != "" {
		if !u.isStrongPassword(user.NewPassword) {
			return envelope.NewError(envelope.InputError, SystemUserPasswordHint, nil)
		}
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(user.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			u.lo.Error("error generating bcrypt password", "error", err)
			return envelope.NewError(envelope.GeneralError, "Error updating user", nil)
		}
		u.lo.Debug("setting new password for user", "user_id", id)
	}

	if _, err := u.q.UpdateUser.Exec(id, user.FirstName, user.LastName, user.Email, pq.Array(user.Roles), user.AvatarURL, hashedPassword, user.Enabled); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating user", nil)
	}
	return nil
}

// SoftDelete soft deletes a user.
func (u *Manager) SoftDelete(id int) error {
	// Disallow if user is system user.
	systemUser, err := u.GetSystemUser()
	if err != nil {
		return err
	}
	if id == systemUser.ID {
		return envelope.NewError(envelope.InputError, "Cannot delete system user", nil)
	}

	if _, err := u.q.SoftDeleteUser.Exec(id); err != nil {
		u.lo.Error("error deleting user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting user", nil)
	}
	return nil
}

// GetEmail retrieves the email of a user by ID.
func (u *Manager) GetEmail(id int) (string, error) {
	var email string
	if err := u.q.GetEmail.Get(&email, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return email, fmt.Errorf("user not found: %v", err)
		}
		u.lo.Error("error fetching user email from db", "error", err)
		return email, fmt.Errorf("fetching user: %w", err)
	}
	return email, nil
}

// SetResetPasswordToken sets a reset password token for a user and returns the token.
func (u *Manager) SetResetPasswordToken(id int) (string, error) {
	token, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		u.lo.Error("error generating reset password token", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error generating reset password token", nil)
	}
	if _, err := u.q.SetResetPasswordToken.Exec(id, token); err != nil {
		u.lo.Error("error setting reset password token", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error setting reset password token", nil)
	}
	return token, nil
}

// ResetPassword sets a new password for a user.
func (u *Manager) ResetPassword(token, password string) error {
	if !u.isStrongPassword(password) {
		return envelope.NewError(envelope.InputError, "Password is not strong enough, " + SystemUserPasswordHint, nil)
	}
	// Hash password.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		u.lo.Error("error generating bcrypt password", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error setting new password", nil)
	}
	if _, err := u.q.ResetPassword.Exec(passwordHash, token); err != nil {
		u.lo.Error("error setting new password", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error setting new password", nil)
	}
	return nil
}

// GetPermissions retrieves the permissions of a user by ID.
func (u *Manager) GetPermissions(id int) ([]string, error) {
	var permissions []string
	if err := u.q.GetPermissions.Select(&permissions, id); err != nil {
		u.lo.Error("error fetching user permissions", "error", err)
		return permissions, envelope.NewError(envelope.GeneralError, "Error fetching user permissions", nil)
	}
	return permissions, nil
}

// verifyPassword compares the provided password with the stored password hash.
func (u *Manager) verifyPassword(pwd []byte, pwdHash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(pwdHash), pwd); err != nil {
		return fmt.Errorf("invalid username or password")
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

// isStrongPassword checks if the password meets the required strength.
func (u *Manager) isStrongPassword(password string) bool {
	if len(password) < minSystemUserPassword || len(password) > maxSystemUserPassword {
		return false
	}
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasUppercase && hasNumber
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
	fmt.Println("password updated successfully.")
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
		systemUserEmail, UserTypeAgent, "System", "", hashedPassword, rmodels.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create system user: %v", err)
	}
	log.Print("system user created successfully")
	return nil
}

// IsStrongSystemUserPassword checks if the password meets the required strength for system user.
func IsStrongSystemUserPassword(password string) bool {
	if len(password) < minSystemUserPassword || len(password) > maxSystemUserPassword {
		return false
	}
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasUppercase && hasNumber
}

// promptAndHashPassword handles password input and validation, and returns the hashed password.
func promptAndHashPassword(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			fmt.Printf("Please set System user password (%s): ", SystemUserPasswordHint)
			buffer := make([]byte, 256)
			n, err := os.Stdin.Read(buffer)
			if err != nil {
				return nil, fmt.Errorf("error reading input: %v", err)
			}
			password := strings.TrimSpace(string(buffer[:n]))
			if IsStrongSystemUserPassword(password) {
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
	_, err := db.Exec(`UPDATE users SET password = $1 WHERE email = $2`, hashedPassword, systemUserEmail)
	if err != nil {
		return fmt.Errorf("failed to update system user password: %v", err)
	}
	return nil
}
