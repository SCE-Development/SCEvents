package models

// MigrationRegistryEntry maps a Mongo collection name to its corresponding model struct.
type MigrationRegistryEntry struct {
	Collection string
	Model      any
}

// MigrationRegistry maps CLI subcommands (or collection names) to their model structures.
// Add an entry here when a new collection needs automatic default migrations.
var MigrationRegistry = map[string]MigrationRegistryEntry{
	"events":        {Collection: "events", Model: Event{}},
	"registrations": {Collection: "registrations", Model: RegistrationRequest{}},
}
