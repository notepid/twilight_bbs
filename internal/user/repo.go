package user

import (
	"database/sql"
	"fmt"
	"time"
)

// Repo handles database operations for users.
type Repo struct {
	db *sql.DB
}

// NewRepo creates a new user repository.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// Create inserts a new user with a hashed password.
func (r *Repo) Create(username, password, realName, location, email string) (*User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(`
		INSERT INTO users (username, password_hash, real_name, location, email, security_level)
		VALUES (?, ?, ?, ?, ?, ?)
	`, username, hash, realName, location, email, LevelNew)
	if err != nil {
		return nil, fmt.Errorf("create user %s: %w", username, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get user id: %w", err)
	}

	return r.GetByID(int(id))
}

// Authenticate checks username/password and returns the user if valid.
func (r *Repo) Authenticate(username, password string) (*User, error) {
	u, err := r.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !CheckPassword(password, u.PasswordHash) {
		return nil, fmt.Errorf("invalid password")
	}

	// Update last call and total calls
	now := time.Now()
	r.db.Exec(`
		UPDATE users SET last_call_at = ?, total_calls = total_calls + 1, updated_at = ?
		WHERE id = ?
	`, now, now, u.ID)

	u.LastCallAt = &now
	u.TotalCalls++

	return u, nil
}

// AuthenticateForSSH validates credentials for SSH authentication.
// Returns true if credentials are valid, false otherwise.
// This method does not update last call time to avoid side effects during SSH handshake.
func (r *Repo) AuthenticateForSSH(username, password string) (bool, error) {
	u, err := r.GetByUsername(username)
	if err != nil {
		// User not found - return false without exposing error
		return false, nil
	}

	// Validate password
	if !CheckPassword(password, u.PasswordHash) {
		return false, nil
	}

	return true, nil
}

// GetByID retrieves a user by ID.
func (r *Repo) GetByID(id int) (*User, error) {
	u := &User{}
	var lastCall sql.NullTime
	var created, updated sql.NullTime

	err := r.db.QueryRow(`
		SELECT id, username, password_hash, real_name, location, email,
		       security_level, total_calls, last_call_at, ansi_enabled,
		       created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.RealName, &u.Location, &u.Email,
		&u.SecurityLevel, &u.TotalCalls, &lastCall, &u.ANSIEnabled,
		&created, &updated,
	)
	if err != nil {
		return nil, fmt.Errorf("get user %d: %w", id, err)
	}

	if lastCall.Valid {
		u.LastCallAt = &lastCall.Time
	}
	if created.Valid {
		u.CreatedAt = created.Time
	}
	if updated.Valid {
		u.UpdatedAt = updated.Time
	}

	return u, nil
}

// GetByUsername retrieves a user by username (case-insensitive).
func (r *Repo) GetByUsername(username string) (*User, error) {
	u := &User{}
	var lastCall sql.NullTime
	var created, updated sql.NullTime

	err := r.db.QueryRow(`
		SELECT id, username, password_hash, real_name, location, email,
		       security_level, total_calls, last_call_at, ansi_enabled,
		       created_at, updated_at
		FROM users WHERE username = ? COLLATE NOCASE
	`, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.RealName, &u.Location, &u.Email,
		&u.SecurityLevel, &u.TotalCalls, &lastCall, &u.ANSIEnabled,
		&created, &updated,
	)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", username, err)
	}

	if lastCall.Valid {
		u.LastCallAt = &lastCall.Time
	}
	if created.Valid {
		u.CreatedAt = created.Time
	}
	if updated.Valid {
		u.UpdatedAt = updated.Time
	}

	return u, nil
}

// Exists checks if a username is already taken.
func (r *Repo) Exists(username string) bool {
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? COLLATE NOCASE", username).Scan(&count)
	return count > 0
}

// UpdateProfile updates a user's profile fields.
func (r *Repo) UpdateProfile(id int, realName, location, email string) error {
	_, err := r.db.Exec(`
		UPDATE users SET real_name = ?, location = ?, email = ?, updated_at = ?
		WHERE id = ?
	`, realName, location, email, time.Now(), id)
	return err
}

// UpdateSecurityLevel changes a user's security level.
func (r *Repo) UpdateSecurityLevel(id int, level int) error {
	_, err := r.db.Exec(`
		UPDATE users SET security_level = ?, updated_at = ? WHERE id = ?
	`, level, time.Now(), id)
	return err
}

// UpdateANSI toggles a user's ANSI preference.
func (r *Repo) UpdateANSI(id int, enabled bool) error {
	_, err := r.db.Exec(`
		UPDATE users SET ansi_enabled = ?, updated_at = ? WHERE id = ?
	`, enabled, time.Now(), id)
	return err
}

// UpdatePassword changes a user's password.
func (r *Repo) UpdatePassword(id int, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?
	`, hash, time.Now(), id)
	return err
}

// List returns all users, ordered by username.
func (r *Repo) List() ([]*User, error) {
	rows, err := r.db.Query(`
		SELECT id, username, real_name, location, security_level, total_calls, last_call_at
		FROM users ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		var lastCall sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.RealName, &u.Location,
			&u.SecurityLevel, &u.TotalCalls, &lastCall); err != nil {
			return nil, err
		}
		if lastCall.Valid {
			u.LastCallAt = &lastCall.Time
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
