package database

import (
	"context"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

type testModel struct {
	StrField   string   `bson:"str_field" default:"hello"`
	IntField   int      `bson:"int_field" default:"42"`
	BoolField  bool     `bson:"bool_field" default:"true"`
	SliceField []string `bson:"slice_field" default:"[\"a\", \"b\"]"`
	NoDefault  string   `bson:"no_default"`
	NoBson     string   `default:"world"`
	Ignored    string   `bson:"-" default:"ignore_me"`
}

func TestBsonFieldName(t *testing.T) {
	modelType := reflect.TypeOf(testModel{})

	tests := []struct {
		fieldName string
		expected  string
	}{
		{"StrField", "str_field"},
		{"NoDefault", "no_default"},
		{"NoBson", ""},
		{"Ignored", "-"},
	}

	for _, tc := range tests {
		field, _ := modelType.FieldByName(tc.fieldName)
		actual := bsonFieldName(field)
		if actual != tc.expected {
			t.Errorf("bsonFieldName(%s) = %s, expected %s", tc.fieldName, actual, tc.expected)
		}
	}
}

func TestParseDefault(t *testing.T) {
	tests := []struct {
		name        string
		typ         reflect.Type
		raw         string
		expected    any
		expectError bool
	}{
		{"String", reflect.TypeOf(""), "hello", "hello", false},
		{"Int", reflect.TypeOf(int(0)), "42", int64(42), false},
		{"InvalidInt", reflect.TypeOf(int(0)), "invalid", nil, true},
		{"BoolTrue", reflect.TypeOf(false), "true", true, false},
		{"BoolFalse", reflect.TypeOf(false), "false", false, false},
		{"InvalidBool", reflect.TypeOf(false), "invalid", nil, true},
		{"Slice", reflect.TypeOf([]string{}), `["a", "b"]`, []string{"a", "b"}, false},
		{"InvalidSlice", reflect.TypeOf([]string{}), `["a",`, nil, true},
		{"Unsupported", reflect.TypeOf(struct{}{}), "something", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseDefault(tc.typ, tc.raw)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !reflect.DeepEqual(actual, tc.expected) {
					t.Errorf("parseDefault() = %v, expected %v", actual, tc.expected)
				}
			}
		})
	}
}

func TestAutoMigrateDefaults(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("Success", func(mt *mtest.T) {
		// Mock UpdateMany responses for the 4 fields with valid default tags in testModel
		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 2}, bson.E{Key: "nModified", Value: 2}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}, bson.E{Key: "nModified", Value: 0}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
		)

		updated, err := AutoMigrateDefaults(context.Background(), mt.Coll, testModel{})
		if err != nil {
			mt.Fatalf("Unexpected error: %v", err)
		}

		if updated != 4 {
			mt.Errorf("Expected 4 total updated docs, got %d", updated)
		}
	})

	mt.Run("NotAStruct", func(mt *mtest.T) {
		_, err := AutoMigrateDefaults(context.Background(), mt.Coll, "not a struct")
		if err == nil {
			mt.Fatalf("Expected error when passing non-struct")
		}
	})
}
