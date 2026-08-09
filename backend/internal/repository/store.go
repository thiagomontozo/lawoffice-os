package repository

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid")
)

type Store struct{ Pool *pgxpool.Pool }

func New(p *pgxpool.Pool) *Store { return &Store{Pool: p} }
func mapError(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "23505", "23503":
			return ErrConflict
		case "23514", "22P02":
			return ErrInvalid
		}
	}
	return e
}
