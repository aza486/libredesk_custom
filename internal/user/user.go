// Package user handles user login, logout and provides functions to fetch user details.
package user

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/stringutil"
	"github.com/abhinavxd/artemis/internal/user/models"
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

	// ErrPasswordTooLong is returned when the password passed to
	// GenerateFromPassword is too long (i.e. > 72 bytes).
	ErrPasswordTooLong = errors.New("password length exceeds 72 bytes")
)

const (
	SystemUserEmail          = "System"
	MinSystemUserPasswordLen = 8
	MaxSystemUserPasswordLen = 50
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
	CreateUser      *sqlx.Stmt `query:"create-user"`
	GetUsers        *sqlx.Stmt `query:"get-users"`
	GetUserCompact  *sqlx.Stmt `query:"get-users-compact"`
	GetUser         *sqlx.Stmt `query:"get-user"`
	GetEmail        *sqlx.Stmt `query:"get-email"`
	GetPermissions  *sqlx.Stmt `query:"get-permissions"`
	GetUserByEmail  *sqlx.Stmt `query:"get-user-by-email"`
	UpdateUser      *sqlx.Stmt `query:"update-user"`
	UpdateAvatar    *sqlx.Stmt `query:"update-avatar"`
	SetUserPassword *sqlx.Stmt `query:"set-user-password"`
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

// Login authenticates a user by email and password.
func (u *Manager) Login(email string, password []byte) (models.User, error) {
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
	var users []models.User
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
	var users []models.User
	if err := u.q.GetUserCompact.Select(&users); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users, nil
		}
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, "Error fetching users", nil)
	}

	return users, nil
}

// Create creates a new user.
func (u *Manager) Create(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating user", nil)
	}

	user.Email = strings.ToLower(user.Email)
	if err := u.q.CreateUser.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, pq.Array(user.Roles)).Scan(&user.ID); err != nil {
		u.lo.Error("error creating user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating user", nil)
	}
	return nil
}

// Get retrieves a user by ID or UUID.
func (u *Manager) Get(id int) (models.User, error) {
	var user models.User
	if err := u.q.GetUser.Get(&user, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, fmt.Errorf("fetching user: %w", err)
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (u *Manager) GetByEmail(email string) (models.User, error) {
	var user models.User
	if err := u.q.GetUserByEmail.Get(&user, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, err
	}
	return user, nil
}

// GetSystemUser retrieves the system user.
func (u *Manager) GetSystemUser() (models.User, error) {
	return u.GetByEmail(SystemUserEmail)
}

// UpdateAvatar updates the user avatar.
func (u *Manager) UpdateAvatar(id int, avatar string) error {
	if _, err := u.q.UpdateAvatar.Exec(id, null.NewString(avatar, avatar != "")); err != nil {
		u.lo.Error("error updating user avatar", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating avatar", nil)
	}
	return nil
}

// UpdateUser updates a user.
func (u *Manager) UpdateUser(id int, user models.User) error {
	var (
		hashedPassword interface{}
		err            error
	)

	if user.NewPassword != "" {
		if !u.isStrongPassword(user.NewPassword) {
			return envelope.NewError(envelope.InputError, "Entered password is not strong please make sure the password has min 8, max 50 characters, at least 1 uppercase letter, 1 number", nil)
		}
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(user.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			u.lo.Error("error generating bcrypt password", "error", err)
			return envelope.NewError(envelope.GeneralError, "Error updating user", nil)
		}
		u.lo.Debug("setting new password for user", "user_id", id)
	}

	if _, err := u.q.UpdateUser.Exec(id, user.FirstName, user.LastName, user.Email, pq.Array(user.Roles), user.AvatarURL, hashedPassword); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating user", nil)
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
	password, _ := stringutil.RandomAlNumString(16)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		u.lo.Error("error generating bcrypt password", "error", err)
		return nil, fmt.Errorf("error generating bcrypt password: %w", err)
	}
	return bytes, nil
}

func (u *Manager) isStrongPassword(password string) bool {
	if len(password) < MinSystemUserPasswordLen || len(password) > MaxSystemUserPasswordLen {
		return false
	}
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasUppercase && hasNumber
}

// CreateSystemUser inserts a default system user into the users table with the prompted password.
func CreateSystemUser(db *sqlx.DB) error {
	// Prompt for password and get hashed password
	hashedPassword, err := promptAndHashPassword()
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (email, first_name, last_name, password, roles) 
		VALUES ($1, $2, $3, $4, $5)`,
		SystemUserEmail, "System", "", hashedPassword, pq.StringArray{"Admin"})
	if err != nil {
		return fmt.Errorf("failed to create system user: %v", err)
	}
	fmt.Println("System user created successfully")
	return nil
}

// ChangeSystemUserPassword updates the system user's password with a newly prompted one.
func ChangeSystemUserPassword(db *sqlx.DB) error {
	// Prompt for password and get hashed password
	hashedPassword, err := promptAndHashPassword()
	if err != nil {
		return err
	}

	// Update system user's password in the database.
	if err := updateSystemUserPassword(db, hashedPassword); err != nil {
		return fmt.Errorf("error updating system user password: %v", err)
	}
	fmt.Println("System user password updated successfully.")
	return nil
}

// promptAndHashPassword handles password input and validation, and returns the hashed password.
func promptAndHashPassword() ([]byte, error) {
	var password string
	for {
		fmt.Print("Please set System admin password (min 8, max 50 characters, at least 1 uppercase letter, 1 number): ")
		fmt.Scanf("%s", &password)
		if isStrongSystemUserPassword(password) {
			break
		}
		fmt.Println("Password does not meet the strength requirements.")
	}

	// Hash the password using bcrypt.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}
	return hashedPassword, nil
}

// updateSystemUserPassword updates the password of the system user in the database.
func updateSystemUserPassword(db *sqlx.DB, hashedPassword []byte) error {
	_, err := db.Exec(`UPDATE users SET password = $1 WHERE email = $2`, hashedPassword, SystemUserEmail)
	if err != nil {
		return fmt.Errorf("failed to update system user password: %v", err)
	}
	return nil
}

// isStrongSystemUserPassword checks if the password meets the required strength for system user.
func isStrongSystemUserPassword(password string) bool {
	if len(password) < MinSystemUserPasswordLen || len(password) > MaxSystemUserPasswordLen {
		return false
	}
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasUppercase && hasNumber
}
