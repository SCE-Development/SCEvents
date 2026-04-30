package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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

func bsonFieldName(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup("bson")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return strings.TrimSpace(name)
}

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
		slicePtr := reflect.New(t)
		if err := json.Unmarshal([]byte(raw), slicePtr.Interface()); err != nil {
			return nil, fmt.Errorf("failed to parse slice default %q as JSON: %w", raw, err)
		}
		return slicePtr.Elem().Interface(), nil

	default:
		return nil, fmt.Errorf("unsupported default tag on %s field", t.Kind())
	}
}
