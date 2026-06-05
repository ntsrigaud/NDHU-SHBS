package order

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"shbs-server/pkg/model"
)

// HandleCheckout handles POST /orders.
func HandleCheckout(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		tx, err := db.Beginx()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
		defer tx.Rollback()

		// 1. Get cart items
		var cartItems []struct {
			ListingID uuid.UUID       `db:"listing_id"`
			Price     decimal.Decimal `db:"price"`
			Status    string          `db:"status"`
		}
		err = tx.Select(&cartItems, `
			SELECT l.id as listing_id, l.price, l.status
			FROM cart_items ci
			JOIN book_listings l ON ci.listing_id = l.id
			WHERE ci.buyer_id = $1`, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		if len(cartItems) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "cart is empty")
		}

		// 2. Validate availability and calculate total
		var total decimal.Decimal
		for _, item := range cartItems {
			if item.Status != model.ListingStatusActive {
				return fiber.NewError(fiber.StatusUnprocessableEntity, "one or more items are no longer available")
			}
			total = total.Add(item.Price)
		}

		// 3. Create Order
		orderID := uuid.New()
		_, err = tx.Exec(`
			INSERT INTO orders (id, buyer_id, total_amount, status)
			VALUES ($1, $2, $3, 'confirmed')`,
			orderID, userID, total,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not create order")
		}

		// 4. Create Order Items & Update Listing Status
		for _, item := range cartItems {
			_, err = tx.Exec(`
				INSERT INTO order_items (id, order_id, listing_id, price_at_purchase)
				VALUES ($1, $2, $3, $4)`,
				uuid.New(), orderID, item.ListingID, item.Price,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "could not create order items")
			}

			_, err = tx.Exec(`
				UPDATE book_listings SET status = 'sold', updated_at = NOW() WHERE id = $1`,
				item.ListingID,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "could not update listing status")
			}
		}

		// 5. Clear Cart
		_, err = tx.Exec(`DELETE FROM cart_items WHERE buyer_id = $1`, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not clear cart")
		}

		if err := tx.Commit(); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"order_id": orderID,
			"message":  "order placed successfully",
		})
	}
}

// HandleListOrders handles GET /orders.
func HandleListOrders(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var orders []OrderResponse
		err = db.Select(&orders, `
			SELECT id, status, total_amount, created_at
			FROM orders
			WHERE buyer_id = $1
			ORDER BY created_at DESC`, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		for i := range orders {
			query := `
				SELECT 
					oi.id, oi.price_at_purchase,
					l.id AS "listing.id", l.seller_id AS "listing.seller_id", l.title AS "listing.title", 
					l.author AS "listing.author", l.isbn AS "listing.isbn", l.course_code AS "listing.course_code", 
					l.department AS "listing.department", l.price AS "listing.price", l.condition AS "listing.condition", 
					l.status AS "listing.status", l.description AS "listing.description", l.ai_confidence AS "listing.ai_confidence",
					l.created_at AS "listing.created_at", l.updated_at AS "listing.updated_at",
					u.name AS "listing.seller_name", 
					img_avatar.cdn_url AS "listing.seller_avatar",
					COALESCE(array_agg(img.cdn_url ORDER BY li.display_order) FILTER (WHERE img.cdn_url IS NOT NULL), '{}') AS "listing.image_urls"
				FROM order_items oi
				JOIN book_listings l ON oi.listing_id = l.id
				JOIN users u ON l.seller_id = u.id
				LEFT JOIN images img_avatar ON u.avatar_image_id = img_avatar.id
				LEFT JOIN listing_images li ON l.id = li.listing_id
				LEFT JOIN images img ON li.image_id = img.id
				WHERE oi.order_id = $1
				GROUP BY oi.id, l.id, u.id, img_avatar.id`

			var items []OrderItemResponse
			if err := db.Select(&items, query, orders[i].ID); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "database error fetching items")
			}
			orders[i].Items = items
		}

		return c.JSON(fiber.Map{"orders": orders})
	}
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
