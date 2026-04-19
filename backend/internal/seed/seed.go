package seed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type variant struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type experimentSeed struct {
	Key        string
	Name       string
	Status     string
	Salt       string
	TrafficPct int
	Variants   []variant
}

type productSeed struct {
	ID          string
	Title       string
	Description string
	PriceCents  int
	ImageURL    string
}

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if err := seedProducts(ctx, pool); err != nil {
		return err
	}
	if err := seedExperiments(ctx, pool); err != nil {
		return err
	}
	return nil
}

func seedProducts(ctx context.Context, pool *pgxpool.Pool) error {
	products := []productSeed{
		{"p-runner-01", "City Runner 01", "Lightweight daily trainer built for paved-road mileage.", 11900, "https://picsum.photos/seed/runner01/600/400"},
		{"p-runner-02", "City Runner 02", "Same last, wider toe box, more foam under the forefoot.", 12900, "https://picsum.photos/seed/runner02/600/400"},
		{"p-trail-01", "Ridge Trail", "Aggressive lug pattern for mixed trail and gravel.", 14900, "https://picsum.photos/seed/trail01/600/400"},
		{"p-court-01", "Court Classic", "Low-profile court shoe with a gum outsole.", 9900, "https://picsum.photos/seed/court01/600/400"},
		{"p-hike-01", "Approach Mid", "Waterproof mid-cut for rocky approach trails.", 17900, "https://picsum.photos/seed/hike01/600/400"},
		{"p-lifestyle-01", "Daily Slip-On", "Pull-on silhouette in soft knit upper.", 8900, "https://picsum.photos/seed/slip01/600/400"},
		{"p-training-01", "Gym Trainer", "Flat stable platform for lifting and short runs.", 10900, "https://picsum.photos/seed/gym01/600/400"},
		{"p-kids-01", "Kids Runner", "Kid-sized version of City Runner 01.", 6900, "https://picsum.photos/seed/kids01/600/400"},
	}
	for _, p := range products {
		_, err := pool.Exec(ctx, `
			INSERT INTO products (id, title, description, price_cents, image_url)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				price_cents = EXCLUDED.price_cents,
				image_url = EXCLUDED.image_url
		`, p.ID, p.Title, p.Description, p.PriceCents, p.ImageURL)
		if err != nil {
			return fmt.Errorf("upsert product %s: %w", p.ID, err)
		}
	}
	return nil
}

func seedExperiments(ctx context.Context, pool *pgxpool.Pool) error {
	exps := buildExperimentSeeds()
	for _, e := range exps {
		blob, err := json.Marshal(e.Variants)
		if err != nil {
			return fmt.Errorf("marshal variants for %s: %w", e.Key, err)
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO experiments (key, name, status, salt, traffic_pct, variants)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (key) DO UPDATE SET
				name        = EXCLUDED.name,
				status      = EXCLUDED.status,
				salt        = EXCLUDED.salt,
				traffic_pct = EXCLUDED.traffic_pct,
				variants    = EXCLUDED.variants,
				updated_at  = now()
		`, e.Key, e.Name, e.Status, e.Salt, e.TrafficPct, blob)
		if err != nil {
			return fmt.Errorf("upsert experiment %s: %w", e.Key, err)
		}
	}
	return nil
}

// buildExperimentSeeds returns a catalog of 20+ realistic experiments covering
// landing, product, cart, checkout, and pricing flows. They all hash from
// independent salts so a user's assignment in one experiment does not
// correlate with another.
func buildExperimentSeeds() []experimentSeed {
	half := []variant{
		{Key: "control", Name: "Control", Weight: 50},
		{Key: "treatment", Name: "Treatment", Weight: 50},
	}
	threeWay := []variant{
		{Key: "control", Name: "Control", Weight: 34},
		{Key: "treatment_a", Name: "Treatment A", Weight: 33},
		{Key: "treatment_b", Name: "Treatment B", Weight: 33},
	}
	seventyThirty := []variant{
		{Key: "control", Name: "Control", Weight: 70},
		{Key: "treatment", Name: "Treatment", Weight: 30},
	}
	return []experimentSeed{
		{"landing_hero_copy", "Landing hero copy", "running", "s01", 100, half},
		{"landing_cta_color", "Landing CTA color", "running", "s02", 100, half},
		{"landing_hero_image", "Landing hero image", "running", "s03", 80, half},
		{"product_grid_layout", "Product grid layout", "running", "s04", 100, half},
		{"product_card_badge", "Product card badge", "running", "s05", 100, threeWay},
		{"product_title_emphasis", "Product title emphasis", "running", "s06", 60, half},
		{"product_price_format", "Product price format", "running", "s07", 100, half},
		{"product_image_aspect", "Product image aspect", "running", "s08", 100, threeWay},
		{"pdp_sticky_cta", "PDP sticky CTA", "running", "s09", 100, seventyThirty},
		{"pdp_recommendation_slot", "PDP recommendation slot", "running", "s10", 100, half},
		{"pdp_shipping_badge", "PDP shipping badge", "running", "s11", 100, half},
		{"cart_free_shipping_threshold", "Cart free-shipping threshold", "running", "s12", 100, threeWay},
		{"cart_upsell_strip", "Cart upsell strip", "running", "s13", 50, half},
		{"checkout_cta_copy", "Checkout CTA copy", "running", "s14", 100, half},
		{"checkout_progress_bar", "Checkout progress bar", "running", "s15", 100, half},
		{"checkout_guest_default", "Checkout guest default", "running", "s16", 100, half},
		{"payment_methods_order", "Payment methods order", "running", "s17", 100, half},
		{"post_purchase_survey", "Post-purchase survey", "running", "s18", 100, half},
		{"homepage_reco_algo", "Homepage reco algorithm", "running", "s19", 100, threeWay},
		{"search_rank_variant", "Search rank variant", "running", "s20", 100, threeWay},
		{"currency_display_mode", "Currency display mode", "running", "s21", 100, half},
		{"empty_cart_banner", "Empty cart banner", "paused", "s22", 100, half},
	}
}
