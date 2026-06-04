# ADR 0001: System Boundary Standardization

## Status

Accepted

## Context

The NDHU Second-Hand Book Store is a greenfield project that could be implemented in various architectural styles. The team must agree on clear ownership boundaries before implementation begins to prevent duplicated responsibilities and unclear operational ownership.

## Decision

The standardized architecture will use:

- `client/` as the **frontend-only** application boundary (Next.js, UI, browser state).
- `server/` as the **sole backend** application boundary (Go Fiber REST API).
- `ai-service/` as a **dedicated AI microservice** boundary (Python FastAPI), called internally by the Go server only — never directly from the client.
- **PostgreSQL** as the single system of record for all structured data.
- **PostgreSQL JSONB** for semi-structured data (messages, notifications) — MongoDB is not used at prototype scale.
- **Versioned SQL migration files** (`server/migrations/`) instead of schema creation at application startup.
- **Typed API contract**: OpenAPI 3.0 generated from Swag annotations → `swagger-typescript-api` → `client/generated/api.ts`. Client always consumes this generated type, never raw `fetch`.

## Consequences

1. The client has no backend logic, no database access, and no direct AI calls.
2. All authentication and authorization rules live exclusively in the Go server.
3. All new backend features are implemented in `server/` only.
4. All new AI features are implemented in `ai-service/` only, exposed to Go via internal HTTP.
5. Client data access is exclusively through the generated typed API module.
6. Any future decision to replace or upgrade a service boundary (e.g., swap FastAPI for Go AI inference) does not affect other boundaries.
