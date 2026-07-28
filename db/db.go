package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:pass@localhost:5434/product_db"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return err
	}

	Pool = pool
	return nil
}

func InitSchema() error {
	_, err := Pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			price NUMERIC(10,2) NOT NULL,
			category VARCHAR(100),
			stock_quantity INTEGER NOT NULL DEFAULT 0,
			image_url TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_by INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_products_name ON products (name);
		CREATE INDEX IF NOT EXISTS idx_products_category ON products (category);
		CREATE INDEX IF NOT EXISTS idx_products_price ON products (price);
		CREATE INDEX IF NOT EXISTS idx_products_active ON products (is_active);
	`)
	return err
}