-- 000001_initial_schema.sql
-- Full SHBS database schema.
-- Applied once by the migrate tool; idempotent via CREATE TABLE IF NOT EXISTS.

-- ──────────────────────────────────────────────────────────────────────────────
-- Extension
-- ──────────────────────────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ──────────────────────────────────────────────────────────────────────────────
-- images  (S3/CloudFront asset registry)
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS images (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    s3_key     TEXT        NOT NULL UNIQUE,
    cdn_url    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ──────────────────────────────────────────────────────────────────────────────
-- users
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    email           TEXT        NOT NULL UNIQUE,
    password_hash   TEXT,                   -- NULL for CAS-only accounts
    is_admin        BOOLEAN     NOT NULL DEFAULT FALSE,
    avatar_image_id UUID        REFERENCES images(id) ON DELETE SET NULL,
    email_verified  BOOLEAN     NOT NULL DEFAULT FALSE,
    cas_id          TEXT        UNIQUE,     -- NDHU CAS principal name
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email   ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_cas_id  ON users (cas_id);

-- ──────────────────────────────────────────────────────────────────────────────
-- book_listings
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS book_listings (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT         NOT NULL,
    author        TEXT         NOT NULL,
    isbn          VARCHAR(13),
    course_code   TEXT,
    department    TEXT,
    price         NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    condition     VARCHAR(20)  NOT NULL CHECK (condition IN ('good','moderate','poor')),
    status        VARCHAR(20)  NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active','reserved','sold','delisted')),
    description   TEXT,
    ai_confidence NUMERIC(5,4),            -- 0.0000–1.0000 from AI microservice
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_listings_seller_id ON book_listings (seller_id);
CREATE INDEX IF NOT EXISTS idx_listings_status    ON book_listings (status);
CREATE INDEX IF NOT EXISTS idx_listings_isbn      ON book_listings (isbn);

-- ──────────────────────────────────────────────────────────────────────────────
-- listing_images  (multiple photos per listing)
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS listing_images (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id   UUID NOT NULL REFERENCES book_listings(id) ON DELETE CASCADE,
    image_id     UUID NOT NULL REFERENCES images(id)        ON DELETE CASCADE,
    display_order INT  NOT NULL DEFAULT 0,
    UNIQUE (listing_id, image_id)
);

CREATE INDEX IF NOT EXISTS idx_listing_images_listing_id ON listing_images (listing_id);

-- ──────────────────────────────────────────────────────────────────────────────
-- cart_items
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS cart_items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id     UUID        NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    listing_id   UUID        NOT NULL REFERENCES book_listings(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (buyer_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_buyer_id ON cart_items (buyer_id);

-- ──────────────────────────────────────────────────────────────────────────────
-- orders
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS orders (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id     UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending','confirmed','completed','cancelled')),
    total_amount NUMERIC(10,2) NOT NULL CHECK (total_amount >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_buyer_id ON orders (buyer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status   ON orders (status);

-- ──────────────────────────────────────────────────────────────────────────────
-- order_items  (one row per book per order)
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_items (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID          NOT NULL REFERENCES orders(id)       ON DELETE CASCADE,
    listing_id   UUID          NOT NULL REFERENCES book_listings(id) ON DELETE RESTRICT,
    price_at_purchase NUMERIC(10,2) NOT NULL CHECK (price_at_purchase >= 0),
    UNIQUE (order_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);

-- ──────────────────────────────────────────────────────────────────────────────
-- messages  (buyer ↔ seller thread per listing)
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS messages (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id   UUID        NOT NULL REFERENCES book_listings(id) ON DELETE CASCADE,
    sender_id    UUID        NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    receiver_id  UUID        NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    body         TEXT        NOT NULL CHECK (char_length(body) <= 2000),
    is_read      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_listing_id   ON messages (listing_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver_id  ON messages (receiver_id, is_read);

-- ──────────────────────────────────────────────────────────────────────────────
-- notifications
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notifications (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         TEXT        NOT NULL,
    payload      JSONB       NOT NULL DEFAULT '{}',
    is_read      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id, is_read);

-- ──────────────────────────────────────────────────────────────────────────────
-- token_blacklist  (invalidated JWTs — stored by SHA-256 hash)
-- ──────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS token_blacklist (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_token_blacklist_expires_at ON token_blacklist (expires_at);
