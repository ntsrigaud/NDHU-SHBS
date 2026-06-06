package cart

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// GetCart handles GET /cart.
//
// @Summary         List cart
// @Description     Returns the authenticated user's cart items
// @Tags            Cart
// @Produce         json
// @Success         200 {array} model.CartItemResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getCart
// @Security        BearerAuth
// @Router          /cart [get]
func GetCart(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

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

	items := []model.CartItemResponse{}
	for rows.Next() {
		var item struct {
			CartItemID        uuid.UUID `db:"cart_item_id"`
			CartItemCreatedAt time.Time `db:"cart_item_created_at"`
			model.ListingWithImages
		}
		if err := rows.StructScan(&item); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
		items = append(items, model.CartItemResponse{
			ID:        item.CartItemID,
			Listing:   item.ListingWithImages,
			CreatedAt: item.CartItemCreatedAt,
		})
	}

	return c.JSON(fiber.Map{"items": items})
}

// AddToCart handles POST /cart.
//
// @Summary         Add to cart
// @Description     Adds a listing to the authenticated user's cart
// @Tags            Cart
// @Accept          json
// @Param           body body model.SwaggerAddToCartRequest true "Listing to add"
// @Produce         json
// @Success         201 {object} model.SwaggerMessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              addToCart
// @Security        BearerAuth
// @Router          /cart [post]
func AddToCart(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var req model.SwaggerAddToCartRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var exists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM book_listings WHERE id = $1 AND status = 'active')`, req.ListingID).Scan(&exists)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if !exists {
		return fiber.NewError(fiber.StatusNotFound, "active listing not found")
	}

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

// RemoveFromCart handles DELETE /cart/:id.
//
// @Summary         Remove from cart
// @Description     Removes a cart item from the authenticated user's cart
// @Tags            Cart
// @Produce         json
// @Param           id path string true "Cart item ID"
// @Success         200 {object} model.SwaggerMessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              removeFromCart
// @Security        BearerAuth
// @Router          /cart/{id} [delete]
func RemoveFromCart(db *sqlx.DB, c *fiber.Ctx) error {
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

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
