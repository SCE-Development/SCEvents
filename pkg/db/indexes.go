package db

import "fmt"

// InitMongoIndexes ensures every MongoDB collection used by the app has its
// indexes created. Add a line to inits when you introduce Init*Indexes for a new collection.
func InitMongoIndexes() error {
	inits := []struct {
		name string
		fn   func() error
	}{
		{"waitlist", InitWaitlistIndexes},
		{"registration", InitRegistrationIndexes},
		{"event", InitEventIndexes},
	}
	for _, idx := range inits {
		if err := idx.fn(); err != nil {
			return fmt.Errorf("%s indexes: %w", idx.name, err)
		}
	}
	return nil
}
