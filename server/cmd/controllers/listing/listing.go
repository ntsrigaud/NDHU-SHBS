package listing

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"

	"shbs-server/pkg/model"
)

// validConditions contains the allowed values for Condition.
var validConditions = map[string]bool{
	"good": true, "moderate": true, "poor": true,
}

// sellerEditableStatuses are the status values a seller may set directly.
var sellerEditableStatuses = map[string]bool{
	"active": true, "delisted": true,
}

const listingWithImagesByIDQuery = `
	SELECT 
		l.id, l.seller_id, l.title, l.author, l.isbn, l.course_code, l.department,
		l.price, l.condition, l.status, l.description, l.ai_confidence,
		l.condition_score, l.ai_processed,
		l.created_at, l.updated_at,
		u.name AS seller_name,
		img_avatar.cdn_url AS seller_avatar,
		COALESCE(array_agg(img.cdn_url ORDER BY li.display_order) FILTER (WHERE img.cdn_url IS NOT NULL), '{}') AS image_urls
	FROM book_listings l
	JOIN users u ON l.seller_id = u.id
	LEFT JOIN images img_avatar ON u.avatar_image_id = img_avatar.id
	LEFT JOIN listing_images li ON l.id = li.listing_id
	LEFT JOIN images img ON li.image_id = img.id
	WHERE l.id = $1
	GROUP BY l.id, u.id, img_avatar.id`

func fetchListingWithImages(db *sqlx.DB, id uuid.UUID) (model.ListingWithImages, error) {
	var listing model.ListingWithImages
	err := db.Get(&listing, listingWithImagesByIDQuery, id)
	return listing, err
}

func normalizeCreateRequest(r *model.SwaggerCreateListingRequest) {
	r.Title = strings.TrimSpace(r.Title)
	r.Author = strings.TrimSpace(r.Author)
	r.ISBN = strings.TrimSpace(r.ISBN)
	r.CourseCode = strings.TrimSpace(r.CourseCode)
	r.Department = strings.TrimSpace(r.Department)
	r.Condition = strings.ToLower(strings.TrimSpace(r.Condition))
	r.Description = strings.TrimSpace(r.Description)
}

func validateCreateRequest(r *model.SwaggerCreateListingRequest) string {
	if r.Title == "" {
		return "title is required"
	}
	if r.Author == "" {
		return "author is required"
	}
	if r.Price.IsNegative() {
		return "price must be >= 0"
	}
	if !validConditions[r.Condition] {
		return "condition must be one of: good, moderate, poor"
	}
	return ""
}

// GetListings handles GET /listings.
//
// @Summary         List listings
// @Description     Paginated, filterable list of book listings. Returns the full page in the JSON body and total count in X-Total-Count header.
// @Tags            Listings
// @Produce         json
// @Param           status      query   string  false   "Filter by status (default: active)"
// @Param           department  query   string  false   "Filter by department"
// @Param           condition   query   string  false   "Filter by condition (good|moderate|poor)"
// @Param           search      query   string  false   "Full-text search on title, author, or ISBN"
// @Param           price_min   query   number  false   "Minimum price"
// @Param           price_max   query   number  false   "Maximum price"
// @Param           page        query   int     false   "Page number (default: 1)"
// @Param           limit       query   int     false   "Items per page (default: 20, max: 100)"
// @Success         200 {array} model.ListingWithImages
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getListings
// @Router          /listings [get]
func GetListings(db *sqlx.DB, c *fiber.Ctx) error {
	conds := []string{}
	args := []any{}
	i := 1

	add := func(cond string, val any) {
		conds = append(conds, fmt.Sprintf(cond, i))
		args = append(args, val)
		i++
	}

	status := c.Query("status", model.ListingStatusActive)
	if status != "all" {
		add("status = $%d", status)
	}

	if sellerID := c.Query("seller_id"); sellerID != "" {
		if id, err := uuid.Parse(sellerID); err == nil {
			add("seller_id = $%d", id)
		}
	}

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
		d, err := decimal.NewFromString(v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid price_min format")
		}
		add("price >= $%d", d)
	}
	if v := c.Query("price_max"); v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid price_max format")
		}
		add("price <= $%d", d)
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

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
	query := fmt.Sprintf(`
		SELECT 
			l.id, l.seller_id, l.title, l.author, l.isbn, l.course_code, l.department,
			l.price, l.condition, l.status, l.description, l.ai_confidence,
			l.condition_score, l.ai_processed,
			l.created_at, l.updated_at,
			u.name AS seller_name, 
			img_avatar.cdn_url AS seller_avatar,
			COALESCE(array_agg(img.cdn_url ORDER BY li.display_order) FILTER (WHERE img.cdn_url IS NOT NULL), '{}') AS image_urls
		FROM book_listings l
		JOIN users u ON l.seller_id = u.id
		LEFT JOIN images img_avatar ON u.avatar_image_id = img_avatar.id
		LEFT JOIN listing_images li ON l.id = li.listing_id
		LEFT JOIN images img ON li.image_id = img.id
		%s
		GROUP BY l.id, u.id, img_avatar.id
		ORDER BY l.created_at DESC 
		LIMIT $%d OFFSET $%d`, where, i, i+1)

	var listings []model.ListingWithImages
	if err := db.Select(&listings, query, listArgs...); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if listings == nil {
		listings = []model.ListingWithImages{}
	}
	c.Set("X-Total-Count", fmt.Sprintf("%d", total))
	return c.JSON(listings)
}

// GetListing handles GET /listings/:id.
//
// @Summary         Get listing by ID
// @Description     Returns a single listing with seller info and image URLs.
// @Tags            Listings
// @Produce         json
// @Param           id path string true "Listing ID"
// @Success         200 {object} model.ListingWithImages
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getListing
// @Router          /listings/{id} [get]
func GetListing(db *sqlx.DB, c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
	}

	listing, err := fetchListingWithImages(db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return fiber.NewError(fiber.StatusNotFound, "listing not found")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.JSON(listing)
}

// GetMyListings handles GET /listings/me (auth required).
//
// @Summary         My listings
// @Description     Returns all listings created by the authenticated user, newest first.
// @Tags            Listings
// @Produce         json
// @Success         200 {array} model.ListingWithImages
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getMyListings
// @Security        BearerAuth
// @Router          /listings/me [get]
func GetMyListings(db *sqlx.DB, c *fiber.Ctx) error {
	sellerID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var listings []model.ListingWithImages
	if err := db.Select(&listings, `
		SELECT
			l.id, l.seller_id, l.title, l.author, l.isbn, l.course_code, l.department,
			l.price, l.condition, l.status, l.description, l.ai_confidence,
			l.condition_score, l.ai_processed,
			l.created_at, l.updated_at,
			u.name AS seller_name,
			img_avatar.cdn_url AS seller_avatar,
			COALESCE(array_agg(img.cdn_url ORDER BY li.display_order) FILTER (WHERE img.cdn_url IS NOT NULL), '{}') AS image_urls
		FROM book_listings l
		JOIN users u ON l.seller_id = u.id
		LEFT JOIN images img_avatar ON u.avatar_image_id = img_avatar.id
		LEFT JOIN listing_images li ON l.id = li.listing_id
		LEFT JOIN images img ON li.image_id = img.id
		WHERE l.seller_id = $1
		GROUP BY l.id, u.id, img_avatar.id
		ORDER BY l.created_at DESC`, sellerID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if listings == nil {
		listings = []model.ListingWithImages{}
	}
	return c.JSON(listings)
}

// CreateListing handles POST /listings (auth required).
//
// @Summary         Create listing
// @Description     Publishes a new book listing for the authenticated seller.
// @Tags            Listings
// @Accept          json
// @Produce         json
// @Param           body body model.SwaggerCreateListingRequest true "Listing details"
// @Success         201 {object} model.ListingWithImages
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              createListing
// @Security        BearerAuth
// @Router          /listings [post]
func CreateListing(db *sqlx.DB, c *fiber.Ctx) error {
	sellerID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var req model.SwaggerCreateListingRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	normalizeCreateRequest(&req)

	if msg := validateCreateRequest(&req); msg != "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, msg)
	}

	tx, err := db.Beginx()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	defer tx.Rollback()

	isbn := nilIfEmpty(req.ISBN)
	courseCode := nilIfEmpty(req.CourseCode)
	department := nilIfEmpty(req.Department)
	description := nilIfEmpty(req.Description)

	var listing model.BookListing
	err = tx.QueryRowx(`
		INSERT INTO book_listings
			(id, seller_id, title, author, isbn, course_code, department,
			 price, condition, status, description, ai_processed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, FALSE)
		RETURNING *`,
		uuid.New(), sellerID,
		req.Title, req.Author, isbn, courseCode, department,
		req.Price, req.Condition, model.ListingStatusPending, description,
	).StructScan(&listing)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not create listing")
	}

	for i, imageID := range req.ImageIDs {
		_, err = tx.Exec(`
			INSERT INTO listing_images (listing_id, image_id, display_order)
			VALUES ($1, $2, $3)`,
			listing.ID, imageID, i,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not associate images")
		}
	}

	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	created, err := fetchListingWithImages(db, listing.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not fetch created listing")
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// UpdateListing handles PATCH /listings/:id (auth required, owner only).
//
// @Summary         Update listing
// @Description     Partially updates a listing. Only the seller may update their own listing.
// @Tags            Listings
// @Accept          json
// @Produce         json
// @Param           id   path string true "Listing ID"
// @Param           body body model.SwaggerUpdateListingRequest true "Fields to update"
// @Success         200 {object} model.ListingWithImages
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         403 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              updateListing
// @Security        BearerAuth
// @Router          /listings/{id} [patch]
func UpdateListing(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
	}

	var listing model.BookListing
	if err := db.Get(&listing, `SELECT * FROM book_listings WHERE id = $1`, id); errors.Is(err, sql.ErrNoRows) {
		return fiber.NewError(fiber.StatusNotFound, "listing not found")
	} else if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if listing.SellerID != userID {
		return fiber.NewError(fiber.StatusForbidden, "you do not own this listing")
	}

	var req model.SwaggerUpdateListingRequest
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

	tx, err := db.Beginx()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	defer tx.Rollback()

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

	if len(setClauses) > 1 {
		args = append(args, id)
		q := fmt.Sprintf("UPDATE book_listings SET %s WHERE id = $%d",
			strings.Join(setClauses, ", "), n)
		if _, err := tx.Exec(q, args...); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not update listing")
		}
	}

	if req.ImageIDs != nil {
		if _, err := tx.Exec(`DELETE FROM listing_images WHERE listing_id = $1`, id); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}
		for i, imageID := range *req.ImageIDs {
			_, err = tx.Exec(`
				INSERT INTO listing_images (listing_id, image_id, display_order)
				VALUES ($1, $2, $3)`,
				id, imageID, i,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "could not update images")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	updated, err := fetchListingWithImages(db, id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not fetch updated listing")
	}

	return c.JSON(updated)
}

// DeleteListing handles DELETE /listings/:id (auth required, owner or admin).
//
// @Summary         Delete listing
// @Description     Soft-deletes a listing by setting its status to 'delisted'. Only the seller or an admin may delete.
// @Tags            Listings
// @Produce         json
// @Param           id path string true "Listing ID"
// @Success         204
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         403 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              deleteListing
// @Security        BearerAuth
// @Router          /listings/{id} [delete]
func DeleteListing(db *sqlx.DB, c *fiber.Ctx) error {
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
	if err := db.Get(&listing, `SELECT * FROM book_listings WHERE id = $1`, id); errors.Is(err, sql.ErrNoRows) {
		return fiber.NewError(fiber.StatusNotFound, "listing not found")
	} else if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if listing.SellerID != userID && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "you do not own this listing")
	}

	if _, err := db.Exec(
		`UPDATE book_listings SET status = 'delisted', updated_at = NOW() WHERE id = $1`, id,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not delist listing")
	}

	return c.SendStatus(fiber.StatusNoContent)
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
