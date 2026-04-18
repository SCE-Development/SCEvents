# cmd

Standalone Go programs. Run with `go run ./cmd/<name>` from the module root.

## migrate

**Purpose**

- Backfills MongoDB documents with default values from `default:"..."` struct tags on registered models.
- Re-running is safe (`$exists: false` filters).

**Usage**

```bash
# Set MONGO_URI if you haven't already. By default the server reads it from docker-compose.
# Example when Mongo is published on the host (see ports in docker-compose.yml):
# MONGO_URI=mongodb://localhost:8100

export MONGO_URI=mongodb://localhost:8100

go run ./cmd/migrate
```
