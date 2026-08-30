package testutils

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type FieldInfo struct {
	Declaration    reflect.StructField
	Initialization reflect.Value
}

func InspectFields(data any, includeUnexportedFields bool) ([]FieldInfo, error) {
	rv := reflect.ValueOf(data)
	if !rv.IsValid() {
		return nil, errors.New("data is nil")
	}

	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data must be a struct or a struct pointer, got %T", data)
	}

	result := []FieldInfo{}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fieldType := rt.Field(i)

		// skip unexported fields unless explicitly requested
		if fieldType.PkgPath != "" && !includeUnexportedFields {
			continue
		}

		result = append(result, FieldInfo{
			Declaration:    fieldType,
			Initialization: rv.Field(i),
		})
	}

	return result, nil
}

func RequireFieldsMetadata(t *testing.T, data any, includeUnexportedFields bool) []FieldInfo {
	t.Helper()

	fieldsInfo, err := InspectFields(data, includeUnexportedFields)
	if err != nil {
		t.Fatalf("failed to inspect fields: %v", err)
	}

	return fieldsInfo
}
