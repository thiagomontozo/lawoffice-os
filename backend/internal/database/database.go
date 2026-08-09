package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	c, e := pgxpool.ParseConfig(url)
	if e != nil {
		return nil, e
	}
	c.MaxConns = 16
	c.MinConns = 1
	c.MaxConnLifetime = time.Hour
	p, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return p, nil
}
func Migrate(ctx context.Context, p *pgxpool.Pool, dir string) error {
	if _, e := p.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(name text PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now())`); e != nil {
		return e
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return e
	}
	names := []string{}
	for _, x := range entries {
		if !x.IsDir() && strings.HasSuffix(x.Name(), ".up.sql") {
			names = append(names, x.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var exists bool
		if e = p.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`, name).Scan(&exists); e != nil {
			return e
		}
		if exists {
			continue
		}
		body, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil {
			return e
		}
		tx, e := p.Begin(ctx)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, string(body), pgx.QueryExecModeSimpleProtocol); e == nil {
			_, e = tx.Exec(ctx, `INSERT INTO schema_migrations(name)VALUES($1)`, name)
		}
		if e != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, e)
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
	}
	return nil
}
