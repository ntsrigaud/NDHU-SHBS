package listing

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"shbs-server/pkg/model"
)

// HandleListListings handles GET /listings.
//
// Public, paginated. Optional query params:
//
//	department, condition, status (default "active"),
//	search (ILIKE on title/author, exact on isbn),
//	price_min, price_max, page (default 1), limit (default 20, max 100).
func HandleListListings(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		conds := []string{}
		args := []any{}
		i := 1

		add := func(cond string, val any) {
			conds = append(conds, fmt.Sprintf(cond, i))
			args = append(args, val)
			i++
		}

		status := c.Query("status", model.ListingStatusActive)
		add("status = $%d", status)

		if dept := c.Query("department"); dept != "" {
			add("department = $%d", dept)
		}
		if cond := c.Query("condition"); cond != "" {
			add("condition = $%d", cond)
		}
		if q := c.Query("search"); q != "" {
			like := "%" + q + "%"
			conds = append(conds,
				fmt.Sprintf("(title ILIKE $%d OR author ILIKE $%d OR isbn = $%d)", i, i+1, i+2))
			args = append(args, like, like, q)
			i += 3
		}
		if v := c.Query("price_min"); v != "" {
			if d, err := decimal.NewFromString(v); err == nil {
				add("price >= $%d", d)
			}
		}
		if v := c.Query("price_max"); v != "" {
			if d, err := decimal.NewFromString(v); err == nil {
				add("price <= $%d", d)
			}
		}

		where := ""
		if len(conds) > 0 {
			where = " WHERE " + strings.Join(conds, " AND ")
		}

		// Total count (same WHERE, no LIMIT/OFFSET).
		var total int
		if err := db.QueryRow("SELECT COUNT(*) FROM book_listings"+where, args...).Scan(&total); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		page := c.QueryInt("page", 1)
		if page < 1 {
			page = 1
		}
		limit := c.QueryInt("limit", 20)
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit

		listArgs := append(args, limit, offset)
		query := "SELECT * FROM book_listings" + where +
			fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", i, i+1)

		var listings []model.BookListing
		if err := db.Select(&listings, query, listArgs...); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		return c.JSON(fiber.Map{
			"data":  listings,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

// HandleGetListing handles GET /listings/:id (public).
func HandleGetListing(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
		}

		var listing model.BookListing
		if err := db.Get(&listing, `SELECT * FROM book_listings WHERE id = $1`, id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "listing not found")
		}

		return c.JSON(fiber.Map{"listing": listing})
	}
}

// HandleCreateListing handles POST /listings (auth required).
func HandleCreateListing(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sellerID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var req CreateListingRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		normalizeCreateRequest(&req)

		if msg := validateCreateRequest(&req); msg != "" {
			return fiber.NewError(fiber.StatusUnprocessableEntity, msg)
		}

		// nil-ify empty optional strings so they are stored as SQL NULL.
		isbn := nilIfEmpty(req.ISBN)
		courseCode := nilIfEmpty(req.CourseCode)
		department := nilIfEmpty(req.Department)
		description := nilIfEmpty(req.Description)

		var listing model.BookListing
		err = db.QueryRowx(`
			INSERT INTO book_listings
				(id, seller_id, title, author, isbn, course_code, department,
				 price, condition, status, description)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10)
			RETURNING *`,
			uuid.New(), sellerID,
			req.Title, req.Author, isbn, courseCode, department,
			req.Price, req.Condition, description,
		).StructScan(&listing)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not create listing")
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"listing": listing})
	}
}

// HandleUpdateListing handles PUT /listings/:id (auth required, owner only).
func HandleUpdateListing(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
		}

		// Fetch and verify ownership.
		var listing model.BookListing
		if err := db.Get(&listing, `SELECT * FROM book_listings WHERE id = $1`, id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "listing not found")
		}
		if listing.SellerID != userID {
			return fiber.NewError(fiber.StatusForbidden, "you do not own this listing")
		}

		var req UpdateListingRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}

		if req.Condition != nil {
			*req.Condition = strings.ToLower(strings.TrimSpace(*req.Condition))
			if !validConditions[*req.Condition] {
				return fiber.NewError(fiber.StatusUnprocessableEntity,
					"condition must be one of: good, moderate, poor")
			}
		}
		if req.Price != nil && req.Price.IsNegative() {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "price must be >= 0")
		}
		if req.Status != nil && !sellerEditableStatuses[*req.Status] {
			return fiber.NewError(fiber.StatusUnprocessableEntity,
				"status must be one of: active, delisted")
		}
		if req.Title != nil {
			*req.Title = strings.TrimSpace(*req.Title)
			if *req.Title == "" {
				return fiber.NewError(fiber.StatusUnprocessableEntity, "title is required")
			}
		}
		if req.Author != nil {
			*req.Author = strings.TrimSpace(*req.Author)
			if *req.Author == "" {
				return fiber.NewError(fiber.StatusUnprocessableEntity, "author is required")
			}
		}

		setClauses := []string{"updated_at = NOW()"}
		args := []any{}
		n := 1

		setField := func(col string, val any) {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, n))
			args = append(args, val)
			n++
		}
		if req.Title != nil {
			setField("title", *req.Title)
		}
		if req.Author != nil {
			setField("author", *req.Author)
		}
		if req.ISBN != nil {
			setField("isbn", nilIfEmpty(*req.ISBN))
		}
		if req.CourseCode != nil {
			setField("course_code", nilIfEmpty(*req.CourseCode))
		}
		if req.Department != nil {
			setField("department", nilIfEmpty(*req.Department))
		}
		if req.Price != nil {
			setField("price", *req.Price)
		}
		if req.Condition != nil {
			setField("condition", *req.Condition)
		}
		if req.Status != nil {
			setField("status", *req.Status)
		}
		if req.Description != nil {
			setField("description", nilIfEmpty(*req.Description))
		}

		if len(setClauses) == 1 {
			return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
		}

		args = append(args, id)
		q := fmt.Sprintf("UPDATE book_listings SET %s WHERE id = $%d RETURNING *",
			strings.Join(setClauses, ", "), n)

		var updated model.BookListing
		if err := db.QueryRowx(q, args...).StructScan(&updated); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not update listing")
		}

		return c.JSON(fiber.Map{"listing": updated})
	}
}

// HandleDeleteListing handles DELETE /listings/:id (auth required, owner or admin).
// This is a soft-delete: it sets status = 'delisted'.
func HandleDeleteListing(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		isAdmin, _ := c.Locals("isAdmin").(bool)

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
		}

		var listing model.BookListing
		if err := db.Get(&listing, `SELECT * FROM book_listings WHERE id = $1`, id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "listing not found")
		}
		if listing.SellerID != userID && !isAdmin {
			return fiber.NewError(fiber.StatusForbidden, "you do not own this listing")
		}

		if _, err := db.Exec(
			`UPDATE book_listings SET status = 'delisted', updated_at = NOW() WHERE id = $1`, id,
		); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not delist listing")
		}

		return c.JSON(fiber.Map{"message": "listing delisted"})
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}

// nilIfEmpty converts an empty string to nil so it is stored as SQL NULL.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
