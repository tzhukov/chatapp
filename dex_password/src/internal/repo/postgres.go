package repo

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
)

type User struct {
	ID    int64
	Email string
	Hash  string
}

type Postgres struct{ DB *sql.DB }

func NewPostgres(dsn string) (*Postgres, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &Postgres{DB: db}, nil
}

func (p *Postgres) Close() error { return p.DB.Close() }

func (p *Postgres) Migrate(ctx context.Context) error {
	_, err := p.DB.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
	return err
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := p.DB.QueryRowContext(ctx, `SELECT id, email, password_hash FROM users WHERE email=$1`, email).Scan(&u.ID, &u.Email, &u.Hash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (p *Postgres) CreateUser(ctx context.Context, email, hash string) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING`, email, hash)
	return err
}
