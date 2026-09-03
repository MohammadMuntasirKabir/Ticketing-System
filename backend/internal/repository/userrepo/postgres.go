package userrepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/you/airline-ticketing/internal/domain/user"
)

// Postgres implements user.Repository using PostgreSQL.
type Postgres struct {
	db *sql.DB
}

// NewPostgres creates a new PostgreSQL-backed user repository.
func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (r *Postgres) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var u user.User
	var role, phone, status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, first_name, last_name, phone, role, status, created_at, updated_at
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &phone, &role, &status, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	u.Role = user.Role(role)
	u.Status = status
	if phone != "" {
		u.Phone = &phone
	}
	return &u, nil
}

func (r *Postgres) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	var role, phone, status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, first_name, last_name, phone, role, status, created_at, updated_at
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &phone, &role, &status, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	u.Role = user.Role(role)
	u.Status = status
	if phone != "" {
		u.Phone = &phone
	}
	return &u, nil
}

func (r *Postgres) Create(ctx context.Context, u *user.User) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, first_name, last_name, phone, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING id
	`, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Phone, string(u.Role), u.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

func (r *Postgres) UpdateProfile(ctx context.Context, id int64, upd user.ProfileUpdate) error {
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if upd.FirstName != nil {
		query += fmt.Sprintf(", first_name = $%d", argCount)
		args = append(args, *upd.FirstName)
		argCount++
	}
	if upd.LastName != nil {
		query += fmt.Sprintf(", last_name = $%d", argCount)
		args = append(args, *upd.LastName)
		argCount++
	}
	if upd.Phone != nil {
		query += fmt.Sprintf(", phone = $%d", argCount)
		args = append(args, *upd.Phone)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Postgres) UpdateRole(ctx context.Context, id int64, role user.Role) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, string(role), id)
	return err
}

func (r *Postgres) List(ctx context.Context, query string, limit, offset int) ([]user.User, error) {
	var rows *sql.Rows
	var err error
	if query != "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, email, password_hash, first_name, last_name, phone, role, status, created_at, updated_at
			WHERE email ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1
			ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`, "%"+query+"%", limit, offset)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, email, password_hash, first_name, last_name, phone, role, status, created_at, updated_at
			ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *Postgres) Count(ctx context.Context, query string) (int, error) {
	var count int
	if query != "" {
		err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM users WHERE email ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1
		`, "%"+query+"%").Scan(&count)
		return count, err
	}
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func scanUsers(rows *sql.Rows) ([]user.User, error) {
	var results []user.User
	for rows.Next() {
		var u user.User
		var role, phone, status string
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &phone, &role, &status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.Role = user.Role(role)
		u.Status = status
		if phone != "" {
			u.Phone = &phone
		}
		results = append(results, u)
	}
	return results, rows.Err()
}

var _ user.Repository = (*Postgres)(nil)
