# NDHU Second-Hand Book Store (SHBS)

A campus-exclusive marketplace for National Dong Hwa University students to buy and sell used textbooks, with AI-powered condition classification and automated metadata extraction.

---

## Architecture

```
client/       Next.js 14 (App Router) + TypeScript + TailwindCSS
server/       Go 1.22 + Fiber v2 REST API
ai-service/   Python 3.12 + FastAPI (condition classification & metadata extraction)
```

All services are routed through **Traefik v3** and packaged with **Docker Compose**.

---

## Quick Start

### Prerequisites

- Docker ≥ 24 and Docker Compose v2
- Go 1.22+ (for local server development)
- Node.js 20+ (for local client development)
- Python 3.12+ (for local AI service development)

### 1. Clone & configure

```bash
git clone https://github.com/ntsrigaud/NDHU-SHBS.git
cd NDHU-SHBS
cp server/.env.example server/.env
cp client/.env.example client/.env
cp ai-service/.env.example ai-service/.env
```

Edit each `.env` file and fill in real values where indicated.

### 2. Install dependencies

```bash
make install
```

### 3. Run the full stack (Docker)

```bash
make stack-up
```

| Service | URL                       |
| ------- | ------------------------- |
| Client  | http://localhost:3000     |
| API     | http://localhost:8080/api |
| AI      | http://localhost:8000     |
| Traefik | http://localhost:8081     |

### 4. Local development (without Docker)

```bash
# Start database only
make server-db-up

# Terminal 1 — Go API
make server-dev

# Terminal 2 — Next.js client
make client-dev

# Terminal 3 — AI service
make ai-dev
```

---

## Developer Commands

| Command               | Description                                   |
| --------------------- | --------------------------------------------- |
| `make install`        | Install all client and AI dependencies        |
| `make format`         | Format client (Prettier) and server (gofmt)   |
| `make check`          | Lint + typecheck + tests                      |
| `make build`          | Build client production bundle                |
| `make stack-up`       | Start full Docker Compose stack               |
| `make stack-down`     | Stop containers                               |
| `make stack-logs`     | Follow container logs                         |
| `make server-db-up`   | Start only the PostgreSQL container           |
| `make server-migrate` | Apply pending SQL migrations                  |
| `make server-test`    | Run Go tests with coverage                    |
| `make server-dev`     | Migrate + start Go API locally                |
| `make server-format`  | Format Go source files                        |
| `make client-dev`     | Start Next.js dev server                      |
| `make client-build`   | Build Next.js for production                  |
| `make generate-api`   | Regenerate typed API client from OpenAPI spec |
| `make ai-dev`         | Start AI service locally                      |
| `make seed`           | Seed staging database with demo data          |

---

## Project Structure

```
NDHU-SHBS/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml          # PR validation: lint, test, build
│   │   └── deploy.yml      # Staging deploy on merge to main
│   └── PULL_REQUEST_TEMPLATE.md
├── client/                 # Next.js frontend
├── server/                 # Go Fiber backend
├── ai-service/             # Python FastAPI AI microservice
├── docker-compose.yml      # Full local stack
├── Makefile
└── docs/
    ├── adr-0001-system-boundary.md
    ├── project-proposal.md
    └── implementation-roadmap.md
```

---

## Documentation

- [Project Proposal](docs/project-proposal.md)
- [Implementation Roadmap](docs/implemetation-roadmap.md)
- API docs: `http://localhost:8080/swagger/` (when stack is running)
