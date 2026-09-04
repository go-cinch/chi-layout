# Database Guidelines

## Boundaries

- Keep database connection setup, optional database creation, pool configuration, transactions, and embedded migrations in this package.
- Keep business SQL and domain queries inside their owning capability packages.
- Expose small database abstractions only when they are used by multiple capabilities; prefer `*sql.DB`, `*sql.Tx`, and the existing `SQLExecutor` interface.
- Propagate `context.Context` through every database operation.
- Keep transaction boundaries explicit. Use `Store.Tx` and obtain the active executor through `Store.SQL`.

## Connection Lifecycle

- Validate the driver and DSN before opening a connection.
- Use `database.autoCreate` only when the configured account is allowed to create databases. Production deployments should normally provision databases outside the application.
- Configure pool limits and connection lifetime through `conf/database.yml`; do not hardcode environment-specific values.
- Close partially initialized resources on every startup failure path.

## Security and Observability

- Never log database credentials, raw SQL arguments, or sensitive query results.
- Never log a raw DSN. Use `redact.DSN` whenever a DSN appears in a log message.
- Keep SQL values parameterized. Never build values into SQL strings; dynamic identifiers must be validated and quoted.
- Database log messages follow the root logging rules and include trace correlation when a request context is available.

## Schema

- Express every schema change as a SQL migration embedded in the service.
- Give each table an independent primary key. Use `BIGSERIAL PRIMARY KEY` for PostgreSQL or `BIGINT AUTO_INCREMENT PRIMARY KEY` for MySQL unless the schema has a stronger established convention.
- Keep business identity separate from the primary key and enforce natural or composite keys with named unique constraints or indexes.
- Use the singular `t_` table-name prefix, for example `t_user` rather than `users`.
- Every table must define `created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP` and an `updated_at` field. Place them immediately after `id`, in `created_at`, `updated_at` order.
- Name indexes and constraints with descriptive prefixes such as `idx_`, `uk_`, `ck_`, and `fk_`.
- Avoid reserved or ambiguous identifiers when practical. If a reserved name is intentional, quote it consistently for the selected database dialect.
- Declare nullability, defaults, uniqueness, referential integrity, and domain checks explicitly in SQL.
- Add comments for non-obvious identifiers, state fields, counters, timestamps, and JSON columns.
- Prefer database constraints over application-only validation for data integrity.

## Migrations

- Migration filenames use `YYYYMMDDHH-description.sql`.
- Every migration contains both `-- +migrate Up` and `-- +migrate Down` sections.
- Keep `Down` symmetrical with `Up`: remove dependent objects safely before removing the object introduced by the migration.
- The checked-in `.sql.example` file is documentation only. Copy or rename it to a timestamped `.sql` file before adding real migration statements.
- When the embedded migrations directory contains no `.sql` files, migration execution must be skipped without touching the database handle.
- Do not edit a committed migration. Create a new migration for every later correction or extension.
- An uncommitted migration may be revised with its related code. Keep one table's complete initial definition together instead of immediately adding follow-up alterations.
- Keep migrations simple and readable. Seed only data required for local development or first boot.

## Testing

- Put tests requiring a real database in `internal/tests` and use an isolated test database.
