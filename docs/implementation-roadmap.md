# NDHU Second-Hand Book Store — Revised Implementation Roadmap

**Team:** Neil Taison Rigaud, Jn Neil Alexander, Sley Hortes, Jaime Medina.  
**Methodology:** Agile / Scrum — six-week timeline, four sprints.  
**Reference Implementation:** Sharlice-Workshop (API generation pipeline, containerization strategy, middleware stack, migration tooling).

---

## Architectural Decisions (binding)

| Concern                        | Decision                                                                                                                                     |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Frontend                       | Next.js 14 (App Router), TypeScript, TailwindCSS, Redux Toolkit, TanStack Query, React Hook Form + Zod                                       |
| Backend                        | Go 1.22 + Fiber v2 — sole application backend                                                                                                |
| Database (relational)          | PostgreSQL 16 — system of record for all structured data                                                                                     |
| Database (unstructured / chat) | PostgreSQL JSONB columns for messages and notifications (eliminates MongoDB operational overhead at prototype scale; reassess at production) |
| File Storage                   | AWS S3 + CloudFront (same pattern as reference)                                                                                              |
| Auth                           | Custom JWT flow with NDHU SSO as primary identity provider; local email/password fallback                                                    |
| API Contract                   | OpenAPI 3.0 generated from Swag annotations → `swagger-typescript-api` → typed `client/generated/api.ts`                                     |
| Reverse Proxy                  | Traefik v3 for request routing, TLS termination, and service discovery                                                                       |
| Containerization               | Multi-stage Docker builds (builder → production, non-root user) + Docker Compose                                                             |
| AI/ML                          | Dedicated Python FastAPI microservice (`ai-service/`) consuming Roboflow-trained models; Go backend calls it internally                      |
| CI/CD                          | GitHub Actions: lint + typecheck + build (client), go test + coverage (server), image build (on merge to main)                               |

---

## Phase 0 — Repository Standards & Infrastructure (Sprint 0, Week 1)

**Goal:** Every developer can clone, run, and contribute with a single command. All tooling is deterministic.

### 0.1 Repository Structure

```
NDHU-SHBS/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml          # lint, test, build on every PR
│   │   └── deploy.yml      # build + push images on merge to main
│   └── PULL_REQUEST_TEMPLATE.md
├── client/                 # Next.js frontend
├── server/                 # Go Fiber backend
├── ai-service/             # Python FastAPI AI microservice
├── docker-compose.yml      # full local stack (Traefik + client + server + ai + postgres)
├── Makefile                # unified developer commands
└── docs/
    ├── adr-0001-system-boundary.md
    ├── project-proposal.md
    └── implementation-roadmap.md
```

### 0.2 Makefile Targets

| Target                | Action                                           |
| --------------------- | ------------------------------------------------ |
| `make install`        | Install all client and server dependencies       |
| `make format`         | Run Prettier (client) and gofmt (server)         |
| `make check`          | Run lint, typecheck, go vet, and tests           |
| `make build`          | Build client and server production artifacts     |
| `make stack-up`       | Start full Docker Compose stack                  |
| `make stack-down`     | Stop and remove containers                       |
| `make server-migrate` | Apply pending SQL migrations                     |
| `make server-test`    | Run Go tests with coverage report                |
| `make generate-api`   | Regenerate typed client from server OpenAPI spec |
| `make seed`           | Seed staging database with demo fixtures         |

### 0.3 Environment Files

- `server/.env.example` — `DATABASE_URL`, `JWT_SECRET`, `JWT_EXPIRY_HOURS`, `AWS_BUCKET`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AI_SERVICE_URL`, `PORT`, `NDHU_SSO_BASE_URL`
- `client/.env.example` — `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_CLOUDFRONT_URL`
- `ai-service/.env.example` — `ROBOFLOW_API_KEY`, `MODEL_ENDPOINT`, `PORT`

### 0.4 Docker Compose (local dev)

Services: `traefik`, `postgres`, `api`, `ai`, `client`.

- `postgres` — `postgres:16-alpine`, healthcheck (`pg_isready`), persistent named volume.
- `api` — multi-stage Go image; runs `./migrate && ./main` on startup; depends on `postgres` healthy.
- `ai` — Python 3.12-slim image; uvicorn on port 8000.
- `client` — Next.js standalone output on port 3000.
- `traefik` — routes `/api/*` and `/ai/*` to respective services; routes `/` to client.

### 0.5 CI Pipeline (`.github/workflows/ci.yml`)

```
on: [push, pull_request]

jobs:
  client:   npm ci → lint → tsc --noEmit → next build
  server:   go mod download → go vet → go test -race -coverprofile=coverage.out ./...
  ai:       pip install → pytest
```

Quality gates: PRs must pass all jobs before merge. Branch protection enforces this on `main`.

---

## Phase 1 — Database Schema & Backend Foundation (Sprint 0–1, Weeks 1–2)

**Goal:** The complete relational schema is defined upfront as versioned SQL migrations. The Go server boots, serves health, and exposes typed Swagger docs.

### 1.1 PostgreSQL Schema (Migration Files)

**`000001_initial_schema.sql`**

```sql
CREATE TABLE IF NOT EXISTS images (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key         TEXT        NOT NULL,
    url         TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id    VARCHAR(20) UNIQUE,                     -- NDHU student ID (nullable for local-auth fallback)
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255),                           -- NULL when SSO-only user
    avatar_url    TEXT        DEFAULT '',
    is_verified   BOOLEAN     NOT NULL DEFAULT FALSE,
    is_admin      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS book_listings (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(500) NOT NULL,
    author          VARCHAR(255),
    isbn            VARCHAR(20),
    department      VARCHAR(255),
    course_code     VARCHAR(50),
    price           INT          NOT NULL CHECK (price >= 0),
    condition       VARCHAR(20)  NOT NULL CHECK (condition IN ('good','moderate','poor')),
    condition_score NUMERIC(4,3),                         -- AI confidence score 0.000–1.000
    description     TEXT,
    status          VARCHAR(20)  NOT NULL DEFAULT 'active' CHECK (status IN ('active','reserved','sold','delisted')),
    ai_processed    BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS listing_images (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID        NOT NULL REFERENCES book_listings(id) ON DELETE CASCADE,
    image_id    UUID        NOT NULL REFERENCES images(id),
    position    INT         NOT NULL DEFAULT 0,           -- display order
    is_cover    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cart_items (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    listing_id  UUID        NOT NULL REFERENCES book_listings(id) ON DELETE CASCADE,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (buyer_id, listing_id)
);

CREATE TABLE IF NOT EXISTS orders (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id        UUID        NOT NULL REFERENCES users(id),
    total_amount    INT         NOT NULL CHECK (total_amount >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','cancelled')),
    invoice_key     TEXT,                                 -- S3 key of generated invoice PDF
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS order_items (
    id          UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID  NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    listing_id  UUID  NOT NULL REFERENCES book_listings(id),
    price_paid  INT   NOT NULL CHECK (price_paid >= 0)
);

CREATE TABLE IF NOT EXISTS messages (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID        NOT NULL REFERENCES book_listings(id) ON DELETE CASCADE,
    sender_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body        TEXT        NOT NULL CHECK (char_length(body) <= 2000),
    is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        VARCHAR(50) NOT NULL,                     -- 'new_message' | 'order_confirmed' | 'listing_sold'
    payload     JSONB       NOT NULL DEFAULT '{}',
    is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS token_blacklist (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_listings_seller     ON book_listings(seller_id);
CREATE INDEX idx_listings_status     ON book_listings(status);
CREATE INDEX idx_listings_department ON book_listings(department);
CREATE INDEX idx_listings_condition  ON book_listings(condition);
CREATE INDEX idx_listing_images_lid  ON listing_images(listing_id);
CREATE INDEX idx_cart_buyer          ON cart_items(buyer_id);
CREATE INDEX idx_messages_listing    ON messages(listing_id);
CREATE INDEX idx_messages_recipient  ON messages(recipient_id, is_read);
CREATE INDEX idx_notifications_user  ON notifications(user_id, is_read);
CREATE INDEX idx_token_blacklist_exp ON token_blacklist(expires_at);
```

**`000002_verification_tokens.sql`**

```sql
CREATE TABLE IF NOT EXISTS verification_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vtokens_user    ON verification_tokens(user_id);
CREATE INDEX idx_vtokens_expires ON verification_tokens(expires_at);
```

### 1.2 Go Server Foundation

**Middleware stack** (same order as reference):

1. `HealthCheck(db)` — `/health` → checks DB ping, returns `{"status":"ok"}`.
2. `CorsHandler()` — allows configured origins only (no wildcard in production).
3. `Logger()` — structured request logging.
4. `ErrorHandler()` — centralised error-to-JSON mapping.
5. `Compressor()` — gzip/brotli response compression.
6. `RateLimiter()` — 100 req/min per IP (configurable via env).

**Package layout:**

```
server/
├── cmd/
│   ├── controllers/
│   │   ├── auth/        # register, login, logout, verify, refresh
│   │   ├── user/        # profile CRUD
│   │   ├── listing/     # book listing CRUD + filter
│   │   ├── image/       # S3 upload
│   │   ├── cart/        # cart management
│   │   ├── order/       # checkout, invoice
│   │   ├── message/     # conversation CRUD
│   │   └── notification/# notification list + mark-read
│   ├── middleware/      # auth, admin, cors, etc.
│   └── migrate/         # standalone migration binary
├── pkg/
│   ├── config/          # env loading + validation
│   ├── model/           # struct definitions (no ORM)
│   ├── services/        # database, scheduler, S3, AI client
│   └── util/            # JWT, password hash, pagination, validation
├── migrations/          # versioned SQL files
├── docs/                # Swag-generated OpenAPI artifacts
└── main.go
```

**API client generation pipeline** (identical to reference):

```
Swag annotations → swagger.json
→ swagger2openapi → openapi.json
→ swagger-typescript-api → client/generated/api.ts
```

Run via `make generate-api`. The generated file is committed; it changes only when the server API changes.

---

## Phase 2 — Authentication (Sprint 1, Week 2)

**Goal:** Only verified NDHU students can access the system. Auth state is managed exclusively by the Go backend via signed JWTs.

### 2.1 NDHU SSO (Primary Path)

NDHU uses a CAS-compatible SSO. The backend implements:

1. `GET /api/auth/sso/login` — redirects to NDHU CAS login URL.
2. `GET /api/auth/sso/callback?ticket=...` — validates ticket against CAS server, creates or fetches local user row, issues JWT.

**Risk mitigation:** If NDHU IT does not expose a publicly accessible CAS endpoint during development, all SSO routes are mocked behind a `NDHU_SSO_MOCK=true` environment flag that skips external validation and verifies any ticket matching `dev-ticket-*`.

### 2.2 Local Auth Fallback (Development & Fallback)

| Method | Path                         | Auth Required | Description                             |
| ------ | ---------------------------- | ------------- | --------------------------------------- |
| POST   | `/api/auth/register`         | No            | Create account, send verification email |
| POST   | `/api/auth/login`            | No            | Validate credentials, return JWT        |
| POST   | `/api/auth/logout`           | Yes           | Add token to blacklist                  |
| GET    | `/api/auth/verify?token=...` | No            | Mark email as verified                  |
| POST   | `/api/auth/refresh`          | Yes           | Issue new JWT from valid existing token |

**Security practices:**

- Passwords stored as bcrypt hashes (`cost=12`).
- JWT signed with HS256; secret loaded from env, minimum 32 bytes.
- Token blacklist cleaned by scheduled job (same cron pattern as reference).
- Registration rate-limited to 5 requests/min per IP.
- Email verification tokens expire in 24 hours.

### 2.3 Frontend Auth Flow

- `AuthProvider` wraps the app in `client/components/auth/`.
- On successful login, JWT stored in `httpOnly` cookie set by the server (not `localStorage`).
- TanStack Query `useQuery` for `/api/users/me` drives authenticated state.
- Unauthenticated routes redirect to `/login`.

---

## Phase 3 — Marketplace: Book Listings (Sprint 1, Weeks 2–3)

**Goal:** Sellers can list books; buyers can browse, search, and filter.

### 3.1 Backend API — Listings

| Method | Path                      | Auth              | Description                             |
| ------ | ------------------------- | ----------------- | --------------------------------------- |
| GET    | `/api/listings`           | No                | Paginated listing wall with filters     |
| GET    | `/api/listings/:id`       | No                | Single listing detail                   |
| POST   | `/api/listings`           | Yes               | Create listing (triggers AI processing) |
| PATCH  | `/api/listings/:id`       | Yes (owner)       | Update listing fields                   |
| DELETE | `/api/listings/:id`       | Yes (owner/admin) | Delist a book                           |
| GET    | `/api/users/:id/listings` | No                | All listings by a specific seller       |

**Filter query parameters for `GET /api/listings`:**

`?department=CS&condition=good&price_min=50&price_max=500&status=active&sort=price_asc&page=1&limit=20`

Pagination uses `LIMIT`/`OFFSET` with a `total_count` header. Default page size: 20.

### 3.2 Backend API — Images (S3)

| Method | Path              | Auth              | Description                          |
| ------ | ----------------- | ----------------- | ------------------------------------ |
| POST   | `/api/images`     | Yes               | Upload file → S3, return `{id, url}` |
| DELETE | `/api/images/:id` | Yes (owner/admin) | Delete from S3 and DB                |

Images are validated server-side: JPEG/PNG/WebP only, max 5 MB per file, max 6 images per listing.

**S3 key format:** `listings/{listing_id}/{uuid}.{ext}` (same naming convention as reference).

### 3.3 Frontend Pages & Components

| Route                 | Component           | Description                                   |
| --------------------- | ------------------- | --------------------------------------------- |
| `/`                   | `HomePage`          | Hero, featured listings, CTA                  |
| `/books`              | `BookWallPage`      | Filterable grid of active listings            |
| `/books/:id`          | `ListingDetailPage` | Full detail, contact seller, add to cart      |
| `/sell`               | `CreateListingPage` | Multi-step form: upload → AI review → publish |
| `/dashboard/listings` | `MyListingsPage`    | Seller's own listings with status management  |

**`BookWallPage` components:**

- `FilterSidebar` — department, price range, condition checkboxes.
- `BookCard` — cover image, title, price, condition badge, seller.
- `SortDropdown` — price asc/desc, newest.
- `PaginationControls`.

---

## Phase 4 — AI Integration (Sprint 2, Weeks 3–4)

**Goal:** Sellers upload cover images; the system automatically extracts metadata and classifies condition. Sellers review and confirm before the listing goes live.

### 4.1 AI Microservice (`ai-service/`)

**Stack:** Python 3.12, FastAPI, Uvicorn.

```
ai-service/
├── main.py
├── routers/
│   ├── metadata.py    # ISBN/title/author extraction
│   └── condition.py   # condition classification
├── models/
│   └── loader.py      # Roboflow model loading
├── schemas.py
├── requirements.txt
└── Dockerfile
```

**Endpoints:**

| Method | Path                 | Input                 | Output                                                  |
| ------ | -------------------- | --------------------- | ------------------------------------------------------- |
| POST   | `/analyze/metadata`  | `{image_url: str}`    | `{title, author, isbn, confidence}`                     |
| POST   | `/analyze/condition` | `{image_urls: [str]}` | `{condition: "good"\|"moderate"\|"poor", score: float}` |
| GET    | `/health`            | —                     | `{"status":"ok"}`                                       |

The Go backend calls the AI service **asynchronously after listing creation** using a goroutine + gocron worker. The listing is created with `status=active, ai_processed=false`; the AI results are patched in once ready.

### 4.2 Go → AI Integration

```go
// services/ai_client.go
type AIClient struct { BaseURL string }

func (c *AIClient) AnalyzeMetadata(imageURL string) (*MetadataResult, error)
func (c *AIClient) AnalyzeCondition(imageURLs []string) (*ConditionResult, error)
```

HTTP client with 30-second timeout and structured error handling. If the AI service is unreachable, the listing remains `ai_processed=false` and the seller is prompted to fill metadata manually.

### 4.3 Seller Review Flow

1. Seller uploads images → listing created (`ai_processed=false`).
2. Background job calls AI service.
3. AI results are written back to `book_listings` row.
4. Seller receives in-app notification: "Your listing metadata is ready to review."
5. Seller visits `/dashboard/listings/:id/review`, confirms or edits title/author/ISBN/condition.
6. Listing becomes publicly visible after confirmation.

---

## Phase 5 — Messaging (Sprint 2, Week 4)

**Goal:** Buyers and sellers communicate within the context of a specific listing. No global chat.

### 5.1 Backend API

| Method | Path                         | Auth            | Description                                   |
| ------ | ---------------------------- | --------------- | --------------------------------------------- |
| GET    | `/api/listings/:id/messages` | Yes             | Conversation thread for a listing (paginated) |
| POST   | `/api/listings/:id/messages` | Yes             | Send a message                                |
| PATCH  | `/api/messages/:id/read`     | Yes (recipient) | Mark message as read                          |
| GET    | `/api/messages/unread-count` | Yes             | Total unread count badge                      |

Conversation participants are the listing `seller_id` plus any user who has sent a message on that listing. Access control: only participants can read/write a thread.

### 5.2 Polling Strategy

The frontend polls `GET /api/listings/:id/messages` every 5 seconds when the conversation view is active (via TanStack Query `refetchInterval`). WebSocket is deferred to production; polling is sufficient for prototype scale and eliminates connection state complexity.

### 5.3 Frontend

| Route                 | Description                                             |
| --------------------- | ------------------------------------------------------- |
| `/books/:id`          | "Message Seller" button opens inline conversation panel |
| `/dashboard/messages` | All conversations grouped by listing                    |

---

## Phase 6 — Commerce (Sprint 3, Week 5)

**Goal:** Buyer can add books to a persistent cart, go through a simulated checkout, receive an invoice, and the listing is automatically de-listed.

### 6.1 Backend API — Cart

| Method | Path                    | Auth | Description                         |
| ------ | ----------------------- | ---- | ----------------------------------- |
| GET    | `/api/cart`             | Yes  | Get authenticated user's cart items |
| POST   | `/api/cart`             | Yes  | Add listing to cart                 |
| DELETE | `/api/cart/:listing_id` | Yes  | Remove item from cart               |
| DELETE | `/api/cart`             | Yes  | Clear cart                          |

Cart validation: a user cannot add their own listing; sold/delisted listings are rejected with `409 Conflict`.

### 6.2 Backend API — Orders

| Method | Path                      | Auth                  | Description                                  |
| ------ | ------------------------- | --------------------- | -------------------------------------------- |
| POST   | `/api/orders`             | Yes                   | Create order from current cart items         |
| GET    | `/api/orders`             | Yes                   | List authenticated user's orders             |
| GET    | `/api/orders/:id`         | Yes (owner/admin)     | Order detail + invoice URL                   |
| POST   | `/api/orders/:id/confirm` | Yes (admin or seller) | Confirm order → trigger de-listing + invoice |

**Order confirmation flow (transactional):**

```sql
BEGIN;
  UPDATE book_listings SET status = 'sold' WHERE id = ANY($listing_ids);
  UPDATE orders SET status = 'confirmed', confirmed_at = NOW() WHERE id = $order_id;
  INSERT INTO notifications (user_id, type, payload) VALUES
    ($buyer_id, 'order_confirmed', $payload),
    ($seller_id, 'listing_sold', $payload);
COMMIT;
```

All state changes within a single PostgreSQL transaction — no partial updates.

### 6.3 Invoice Generation

Invoices are generated server-side as PDF (using a Go PDF library such as `go-pdf/fpdf`) and uploaded to S3. The S3 key is stored in `orders.invoice_key`. The buyer receives a pre-signed S3 URL valid for 1 hour.

### 6.4 Frontend

| Route       | Description                                  |
| ----------- | -------------------------------------------- |
| `/cart`     | Cart page: item list, total, checkout button |
| `/checkout` | Order summary + "Confirm Purchase"           |
| `/orders`   | Order history with invoice download links    |

---

## Phase 7 — Notifications (Sprint 3, Week 5)

**Goal:** Users receive in-app alerts for all relevant events.

### 7.1 Notification Types

| Type               | Trigger                   | Recipient            |
| ------------------ | ------------------------- | -------------------- |
| `new_message`      | Message sent on a listing | Recipient of message |
| `listing_ai_ready` | AI processing complete    | Seller               |
| `order_confirmed`  | Order confirmed           | Buyer                |
| `listing_sold`     | Order confirmed           | Seller               |

### 7.2 Backend API

| Method | Path                              | Auth | Description                      |
| ------ | --------------------------------- | ---- | -------------------------------- |
| GET    | `/api/notifications`              | Yes  | Paginated notifications for user |
| PATCH  | `/api/notifications/:id/read`     | Yes  | Mark one as read                 |
| PATCH  | `/api/notifications/read-all`     | Yes  | Mark all as read                 |
| GET    | `/api/notifications/unread-count` | Yes  | Count badge                      |

### 7.3 Frontend

- `NotificationBell` in the navbar with unread count badge.
- Polls `GET /api/notifications/unread-count` every 10 seconds (TanStack Query).
- Dropdown panel lists recent notifications with timestamp and link to relevant page.

---

## Phase 8 — Testing & Hardening (Sprint 4, Week 6)

**Goal:** Minimum 80% backend coverage; all P1/P2 defects resolved; security hardened to OWASP Top 10.

### 8.1 Backend Testing (Go)

**Unit tests** — pure function logic (JWT utilities, password hashing, input sanitizers, AI response parsers).

**Integration tests with Testcontainers-go:**

- Spin up a real `postgres:16-alpine` container per test suite.
- Run migrations before test execution.
- Each test runs inside a transaction that is rolled back after the test (isolation without truncation overhead).
- Coverage target: ≥ 80% on `cmd/controllers/` and `pkg/services/`.

```
go test -race -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

CI fails if overall coverage drops below 80%.

### 8.2 Frontend Testing (Vitest + Testing Library)

- Component unit tests for `BookCard`, `FilterSidebar`, `CartItem`, `NotificationBell`.
- Form validation tests for the listing creation flow.
- MSW (Mock Service Worker) to mock API responses in tests.

### 8.3 End-to-End (Playwright)

Critical paths only:

1. Register → verify email → login.
2. Create listing → confirm AI review → listing appears in book wall.
3. Add to cart → checkout → order confirmed → listing status = sold.

### 8.4 Security Hardening

| Area          | Control                                                                                      |
| ------------- | -------------------------------------------------------------------------------------------- |
| Auth          | JWT only in httpOnly cookies; CSRF token on state-changing requests                          |
| Input         | All user inputs validated with Go structs + Fiber's `ctx.BodyParser`; reject unknown fields  |
| SQL           | Parameterised queries everywhere (`sqlx` named queries); zero raw string interpolation       |
| File Uploads  | Server-side MIME type verification (magic bytes, not extension); max size enforced           |
| Rate Limiting | Per-route limits: auth endpoints stricter (5/min), general API (100/min)                     |
| CORS          | Explicit `AllowOrigins` list; no wildcard                                                    |
| Headers       | `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security` added in middleware |
| Dependencies  | `go mod tidy`, `npm audit`; CI blocks on high/critical CVEs                                  |

---

## Phase 9 — Containerization & Deployment (Sprint 4, Week 6)

**Goal:** The entire system runs from a single `docker compose up` command in any environment. Staging is deployed and seeded for the demo.

### 9.1 Server Dockerfile (multi-stage)

```dockerfile
# Stage 1: build
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./main.go
RUN go build -o migrate ./cmd/migrate

# Stage 2: production (non-root, minimal image)
FROM alpine:3.20 AS production
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY --from=builder /app/migrations ./migrations
RUN chown -R appuser:appgroup /app && chmod +x /app/main /app/migrate
USER appuser
EXPOSE 8080
CMD ["sh", "-c", "./migrate && ./main"]
```

### 9.2 Client Dockerfile (Next.js standalone)

```dockerfile
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:20-alpine AS production
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
USER appuser
EXPOSE 3000
CMD ["node", "server.js"]
```

`next.config.mjs` must include `output: 'standalone'`.

### 9.3 AI Service Dockerfile

```dockerfile
FROM python:3.12-slim AS production
RUN useradd -m appuser
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
USER appuser
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

### 9.4 Production Docker Compose

```yaml
services:
  traefik:
    image: traefik:v3.0
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
    ports: ["80:80"]
    volumes: ["/var/run/docker.sock:/var/run/docker.sock:ro"]

  postgres:
    image: postgres:16-alpine
    env_file: ./server/.env
    healthcheck:
      test: "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"
      interval: 3s
      retries: 10
    volumes: [postgres-data:/var/lib/postgresql/data]

  api:
    build: { context: ./server, target: production }
    env_file: ./server/.env
    depends_on: { postgres: { condition: service_healthy } }
    labels:
      - "traefik.http.routers.api.rule=PathPrefix(`/api`)"

  ai:
    build: { context: ./ai-service, target: production }
    env_file: ./ai-service/.env
    labels:
      - "traefik.http.routers.ai.rule=PathPrefix(`/ai`)"

  client:
    build: { context: ./client, target: production }
    env_file: ./client/.env
    depends_on: [api]
    labels:
      - "traefik.http.routers.client.rule=PathPrefix(`/`)"

volumes:
  postgres-data:
```

### 9.5 Staging Deployment

- Cloud provider: any VPS (e.g., AWS EC2 t3.small or Fly.io) sufficient for prototype.
- GitHub Actions `deploy.yml` triggers on merge to `main`: builds images, pushes to GitHub Container Registry (`ghcr.io`), SSH-deploys via `docker compose pull && docker compose up -d` on the staging host.
- `make seed` runs a Go binary (`cmd/seed/main.go`) that inserts 20 demo users, 50 listings, and 5 completed orders into the staging database.

---

## Delivery Milestones

| Week | End-of-Week Deliverable                                                                 |
| ---- | --------------------------------------------------------------------------------------- |
| 1    | Repo, CI, Docker Compose stack boots, `GET /health` returns 200, schema migrations pass |
| 2    | Auth flows (SSO + fallback) functional; JWT middleware protecting routes                |
| 3    | Book listing CRUD, S3 image upload, filterable book wall UI                             |
| 4    | AI metadata extraction + condition classification integrated; messaging functional      |
| 5    | Cart, checkout, order confirmation, auto de-listing, notifications                      |
| 6    | ≥80% backend test coverage; E2E smoke tests pass; staging deployed and seeded           |

---

## Risk Register

| Risk                               | Likelihood | Impact | Mitigation                                                                                             |
| ---------------------------------- | ---------- | ------ | ------------------------------------------------------------------------------------------------------ |
| NDHU SSO not accessible externally | High       | High   | `NDHU_SSO_MOCK=true` flag with CAS ticket mock from day one                                            |
| AI service latency                 | Medium     | Medium | Async processing; listing visible immediately; AI fills in later                                       |
| Timeline slip on AI model accuracy | Medium     | Low    | AI output is always seller-reviewed before publishing                                                  |
| S3 cost overrun                    | Low        | Low    | File size limits (5 MB/image, 6 images/listing); lifecycle policy deletes orphaned images after 7 days |
| Microservice complexity            | Medium     | Medium | Consolidate `ai-service` into Go binary if FastAPI introduces unacceptable ops overhead                |
