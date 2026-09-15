# Architecture

## Dependency direction

```text
HTTP request
  → router / middleware / handler
  → service
  → domain ports
  → repository / queue / storage adapters
  → PostgreSQL / Redis / filesystem or S3
```

Handlers know HTTP but not SQL. Services know business rules but not Gin, pgx, Redis commands, filesystem paths, or AWS request types. Repositories and infrastructure adapters translate domain calls to external systems. `internal/app` is the only package that assembles concrete implementations.

## Package responsibilities

- `cmd`: signal-aware, thin executable entry points
- `internal/app`: resource creation, dependency injection, shutdown ownership
- `internal/auth`: bcrypt and JWT primitives
- `internal/config`: environment parsing, defaults, and startup validation
- `internal/domain`: entities, actors, pagination, jobs, and infrastructure ports
- `internal/handler`: request decoding and response encoding
- `internal/middleware`: request ID and access-token authentication
- `internal/queue`: Redis reliable-list queue adapter
- `internal/repository`: SQLC-to-domain adapters and transaction boundaries
- `internal/router`: endpoint registration
- `internal/service`: authentication, document, and worker behavior
- `internal/storage`: hardened local filesystem and S3 adapters
- `migrations/list`: ordered Go migrations and SQLC schema mirrors
- `pkg`: reusable logging, errors, validation, and client construction

## Upload flow

1. JWT middleware creates a typed actor with an `int64` user ID and role.
2. The handler opens the multipart stream without loading the complete object into memory.
3. The service validates filename metadata, media type, declared size, and actual streamed size.
4. The service generates an `<AWS_S3_PREFIX>/documents/<owner>/<random>` key
   (omitting the prefix segment when unset), isolating objects in shared buckets.
5. The repository inserts the pending metadata row and stores that key in
   `documents.storage_key`.
6. A failed insert triggers compensating object deletion.
7. The service enqueues the document ID after persistence.
8. A queue failure returns a retryable error while keeping the pending row available for operational re-enqueue.
9. API responses derive `url` as `CDN_URL + "/" + storage_key`; the database
   remains the source of truth for the key, and the API does not expose it as a
   separate field.

## Worker flow

Redis moves a payload from the main list to the processing list before returning it to a worker. The worker compare-and-sets the row from `pending` or `failed` to `processing`, streams the object through SHA-256 and a byte counter, and records `ready` only when stored and recorded sizes agree.

Duplicate jobs are safe: ready documents are skipped, while the database claim prevents concurrent processors from winning the same transition. Failures are persisted as `failed`; retried jobs eventually move to the dead-letter list after the configured maximum attempts.

## Authentication flow

Access and refresh tokens use HS256 but have distinct `token_type` claims. Validators enforce algorithm, issuer, audience, expiry, and type. Refresh tokens contain unique IDs, while the database stores only SHA-256 hashes of their full token values. Refresh rotates the previous persisted session and creates its replacement in one repository transaction.

## Consistency boundaries

PostgreSQL transactions cannot atomically include object storage or Redis. The example makes each boundary visible:

- object write followed by failed metadata insert: compensate by deleting the object
- metadata insert followed by failed enqueue: preserve `pending` metadata and return a retryable queue error
- soft delete: preserve the physical object for a future retention workflow

A service requiring stronger delivery guarantees should add a transactional outbox rather than hiding a dual-write assumption inside the handler.

## Migration lifecycle

The migration system has three deliberately separate representations:

1. `migrations/list/NN_name.go` is executable runtime history. Its `Up` and `Down` methods receive a transaction owned by the runner.
2. `migrations/list/migrations.go` is the authoritative execution order.
3. `migrations/list/schema/*.sql` is SQLC's description of the final database shape; it is not executed by the migration runner.

The runner tracks completed names in the database `migrations` table. Applying a migration and recording its name happen in the same transaction, so a failed migration cannot be marked as executed. Rollback similarly runs `Down` and removes the changelog row in one transaction. The registry is validated before any database work, preventing duplicate or malformed names from creating ambiguous history.

Schema evolution follows one direction:

```text
new requirement
  → numbered Go migration
  → migration registry
  → SQLC schema mirror
  → parameterized query file
  → generated db package
  → repository adapter
  → service and handler
```

Once a migration is shared, its name and behavior are immutable. Corrections use the next sequence rather than editing history. `Down` exists for controlled rollback but may be destructive for tables or columns; stable seed migrations can intentionally preserve data with a no-op rollback. Production rollback requires reviewing the migration and ensuring a suitable backup or forward-fix plan.

This separation keeps deployment history explicit while allowing SQLC to type-check queries without connecting to a live database. Its cost is that runtime DDL and SQLC schema must be changed together and verified with registry tests, `sqlc vet`, generated-code review, and the full unit suite.

## Error handling

Domain-facing sentinel errors carry a stable numeric code, safe message, HTTP status, and retryability. Infrastructure causes remain wrapped for logging and error inspection, while the Gin writer sends only the safe envelope. Request IDs are returned and attached to the request context for correlation.

Codes follow `E + Domain(2) + Type(2) + Sequence(4)`. Domain `00` is common, `01` is auth, `02` is document, `03` is storage, and `04` is queue. Type `40` maps to bad request, `41` unauthorized, `42` forbidden, `44` not found, `09` conflict, `50` internal error, and `53` service unavailable. Thus `E01410002` is the second auth unauthorized-class error (invalid credentials), while `E03530001` is the first storage service-unavailable error.

`pkg/errors/codes.go` is the stable external registry. `pkg/errors/errors.go` attaches runtime semantics. Callers use the domain-specific error rather than a generic status error so clients can branch on the code. Published codes must never be reused or renumbered; new domains reserve the next two-digit ID and extend the uniqueness/format contract test.

## Testing boundaries

Core tests use handwritten implementations of domain ports. Repository conversion tests exercise SQLC types without a live database. Storage tests use temporary directories and a narrow fake S3 API. Queue tests use a narrow fake list backend. Handler tests use Gin and `httptest`; application tests verify resource ownership and shutdown order.

The root `api-spec.yml` is the authoritative HTTP contract. Route, request,
response, authentication, and public schema changes must update it in the same
change, and handler tests must remain aligned with that contract.
