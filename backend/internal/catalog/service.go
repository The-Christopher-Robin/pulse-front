package catalog

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceCents  int    `json:"price_cents"`
	ImageURL    string `json:"image_url"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) List(ctx context.Context, limit int) ([]Product, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, description, price_cents, image_url FROM products ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.PriceCents, &p.ImageURL); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, description, price_cents, image_url FROM products WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Title, &p.Description, &p.PriceCents, &p.ImageURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Product{}, ErrNotFound
		}
		return Product{}, err
	}
	return p, nil
}

var ErrNotFound = fmt.Errorf("product not found")
