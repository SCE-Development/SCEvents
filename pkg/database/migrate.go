// Package database hosts cross-cutting MongoDB helpers that aren't tied to
// a specific collection, starting with the auto-migration utility that
// backfills documents using `default:"..."` struct tags on models.
package database

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AutoMigrateDefaults backfills documents in coll that are missing fields
// declared with a `default:"..."` struct tag on the provided model.
//
// For every struct field that has both a `bson:"..."` name and a
// `default:"..."` value, it runs:
//
//	coll.UpdateMany({field: {$exists: false}}, {$set: {field: <parsedDefault>}})
//
// Fields without a `default` tag (or without a usable `bson` name) are
// skipped, so required columns like "_id" / "name" are never backfilled.
//
// Supported field kinds:
//   - string: raw tag value
//   - intN / uintN: parsed via strconv
//   - bool: parsed via strconv.ParseBool
//   - slice: only "[]" is accepted, written as an empty typed slice
//
// Any other kind returns an error so unsupported tags fail loudly.
func AutoMigrateDefaults(ctx context.Context, coll *mongo.Collection, model any) (int64, error) {
	t := reflect.TypeOf(model)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return 0, fmt.Errorf("AutoMigrateDefaults: model must be a struct")
	}

	var totalUpdated int64

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		rawDefault, hasDefault := field.Tag.Lookup("default")
		if !hasDefault {
			continue
		}

		bsonName := bsonFieldName(field)
		if bsonName == "" || bsonName == "-" {
			continue
		}

		value, err := parseDefault(field.Type, rawDefault)
		if err != nil {
			return totalUpdated, fmt.Errorf("%s.%s: %w", t.Name(), field.Name, err)
		}

		filter := bson.M{bsonName: bson.M{"$exists": false}}
		update := bson.M{"$set": bson.M{bsonName: value}}

		res, err := coll.UpdateMany(ctx, filter, update)
		if err != nil {
			return totalUpdated, fmt.Errorf("UpdateMany %s.%s: %w", coll.Name(), bsonName, err)
		}

		totalUpdated += res.ModifiedCount
		log.Printf("  %s.%s -> backfilled %d docs", coll.Name(), bsonName, res.ModifiedCount)
	}

	return totalUpdated, nil
}

// bsonFieldName pulls the document field name from a `bson:"name,opts"` tag,
// stripping modifiers like ",omitempty".
func bsonFieldName(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup("bson")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return strings.TrimSpace(name)
}

// parseDefault converts the raw tag string into a Go value typed to match
// the struct field. Unsupported kinds return an explicit error.
func parseDefault(t reflect.Type, raw string) (any, error) {
	switch t.Kind() {
	case reflect.String:
		return raw, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid int default %q: %w", raw, err)
		}
		return n, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid uint default %q: %w", raw, err)
		}
		return n, nil

	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bool default %q: %w", raw, err)
		}
		return b, nil

	case reflect.Slice:
		if raw != "[]" {
			return nil, fmt.Errorf("slice default must be \"[]\" (got %q)", raw)
		}
		return reflect.MakeSlice(t, 0, 0).Interface(), nil

	default:
		return nil, fmt.Errorf("unsupported default tag on %s field", t.Kind())
	}
}
