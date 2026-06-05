package cart

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/controllers/listing"
)

// HandleListCart handles GET /cart.
func HandleListCart(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		// We need to fetch the cart items and join with listing details.
		// Since we want the rich listing data, we reuse the listing logic.
		query := `
			SELECT 
				ci.id AS cart_item_id, ci.created_at AS cart_item_created_at,
				l.id, l.seller_id, l.title, l.author, l.isbn, l.course_code, l.department,
				l.price, l.condition, l.status, l.description, l.ai_confidence,
				l.created_at, l.updated_at,
				u.name AS seller_name, 
				img_avatar.cdn_url AS seller_avatar,
				COALESCE(array_agg(img.cdn_url ORDER BY li.display_order) FILTER (WHERE img.cdn_url IS NOT NULL), '{}') AS image_urls
			FROM cart_items ci
			JOIN book_listings l ON ci.listing_id = l.id
			JOIN users u ON l.seller_id = u.id
			LEFT JOIN images img_avatar ON u.avatar_image_id = img_avatar.id
			LEFT JOIN listing_images li ON l.id = li.listing_id
			LEFT JOIN images img ON li.image_id = img.id
			WHERE ci.buyer_id = $1
			GROUP BY ci.id, l.id, u.id, img_avatar.id
			ORDER BY ci.created_at DESC`

		rows, err := db.Queryx(query, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
		defer rows.Close()

		items := []CartItemResponse{}
		for rows.Next() {
			var item struct {
				CartItemID        uuid.UUID `db:"cart_item_id"`
				CartItemCreatedAt time.Time `db:"cart_item_created_at"`
				listing.ListingWithImages
			}
			if err := rows.StructScan(&item); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "database error")
			}
			items = append(items, CartItemResponse{
				ID:        item.CartItemID,
				Listing:   item.ListingWithImages,
				CreatedAt: item.CartItemCreatedAt,
			})
		}

		return c.JSON(fiber.Map{"items": items})
	}
}

// HandleAddToCart handles POST /cart.
func HandleAddToCart(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var req AddToCartRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}

		// Verify listing exists and is active
		var exists bool
		err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM book_listings WHERE id = $1 AND status = 'active')`, req.ListingID).Scan(&exists)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
		if !exists {
			return fiber.NewError(fiber.StatusNotFound, "active listing not found")
		}

		// Insert into cart_items (ON CONFLICT DO NOTHING to be idempotent)
		_, err = db.Exec(`
			INSERT INTO cart_items (id, buyer_id, listing_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (buyer_id, listing_id) DO NOTHING`,
			uuid.New(), userID, req.ListingID,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not add to cart")
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "added to cart"})
	}
}

// HandleRemoveFromCart handles DELETE /cart/:id.
func HandleRemoveFromCart(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		cartItemID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid cart item id")
		}

		res, err := db.Exec(`DELETE FROM cart_items WHERE id = $1 AND buyer_id = $2`, cartItemID, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fiber.NewError(fiber.StatusNotFound, "cart item not found")
		}

		return c.JSON(fiber.Map{"message": "removed from cart"})
	}
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
