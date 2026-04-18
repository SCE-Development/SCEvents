# cmd

Standalone Go programs. Run with `go run ./cmd/<name>` from the module root.

## migrate

**Purpose:** Backfills MongoDB documents with default values from `default:"..."` struct tags on registered models. Re-running is safe (`$exists: false` filters).

**Usage:**

```bash
MONGO_URI=mongodb://localhost:27017 go run ./cmd/migrate
```
