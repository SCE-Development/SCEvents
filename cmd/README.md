# cmd

Standalone Go programs. Run with `go run ./cmd/<name>` from the module root.

## migrate

**Purpose**

- Backfills MongoDB documents with default values from `default:"..."` struct tags on registered models.
- Re-running is safe (`$exists: false` filters).

**Usage**

The first argument is required: which collection to migrate (`events` or `registrations`). Omitting it prints `Please specify a collection` and exits; an unknown name prints `Provided type doesn't exist`.

```bash
# Set MONGO_URI if you haven't already. By default the server reads it from docker-compose.
# Example when Mongo is published on the host (docker-compose maps 8100 -> 27017):
# export MONGO_URI=mongodb://localhost:8100

export MONGO_URI=mongodb://localhost:8100

go run ./cmd/migrate events
go run ./cmd/migrate registrations
```

**Examples**

- `events` — runs defaults migration against the `events` collection using `models.Event` (e.g. backfill `admins`, `registration_form`, etc. when those fields were added with `default` tags).
- `registrations` — runs the same logic against the `registrations` collection using `models.RegistrationRequest`. It only updates fields that have a `default` struct tag; if none exist yet, the run succeeds with zero backfills.

**Testing locally**

1. Start Mongo (e.g. `docker compose up -d mongodb`) and set `MONGO_URI` to match your host port (often `mongodb://localhost:8100` from this repo’s compose file).
2. Seed or reuse documents that are missing a field you recently added with a `default` tag (older rows won’t have that key in BSON).
3. Run `go run ./cmd/migrate events` (or `registrations`) and check the log for per-field backfill counts.
4. Optional: in `mongosh`, open the collection and confirm missing keys were set (e.g. `db.events.findOne()`).
