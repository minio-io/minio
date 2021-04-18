// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package data

import (
	"reflect"
	"testing"

	"github.com/minio/minio/pkg/s3select/internal/parquet-go/gen-go/parquet"
	"github.com/minio/minio/pkg/s3select/internal/parquet-go/schema"
)

func TestPopulateMap(t *testing.T) {
	t.Skip("Broken")
	requiredMap1 := schema.NewTree()
	{
		mapElement, err := schema.NewElement("map", parquet.FieldRepetitionType_REQUIRED,
			nil, parquet.ConvertedTypePtr(parquet.ConvertedType_MAP),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		keyValue, err := schema.NewElement("key_value", parquet.FieldRepetitionType_REPEATED,
			nil, nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		requiredKey, err := schema.NewElement("key", parquet.FieldRepetitionType_REQUIRED,
			parquet.TypePtr(parquet.Type_BYTE_ARRAY), parquet.ConvertedTypePtr(parquet.ConvertedType_UTF8),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		requiredValue, err := schema.NewElement("value", parquet.FieldRepetitionType_REQUIRED,
			parquet.TypePtr(parquet.Type_INT32), nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err = requiredMap1.Set("map", mapElement); err != nil {
			t.Fatal(err)
		}

		if err = requiredMap1.Set("map.key_value", keyValue); err != nil {
			t.Fatal(err)
		}

		if err = requiredMap1.Set("map.key_value.key", requiredKey); err != nil {
			t.Fatal(err)
		}

		if err = requiredMap1.Set("map.key_value.value", requiredValue); err != nil {
			t.Fatal(err)
		}

		if _, _, err = requiredMap1.ToParquetSchema(); err != nil {
			t.Fatal(err)
		}
	}

	requiredMap2 := schema.NewTree()
	{
		mapElement, err := schema.NewElement("map", parquet.FieldRepetitionType_REQUIRED,
			nil, parquet.ConvertedTypePtr(parquet.ConvertedType_MAP),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		keyValue, err := schema.NewElement("key_value", parquet.FieldRepetitionType_REPEATED,
			nil, nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		requiredKey, err := schema.NewElement("key", parquet.FieldRepetitionType_REQUIRED,
			parquet.TypePtr(parquet.Type_BYTE_ARRAY), parquet.ConvertedTypePtr(parquet.ConvertedType_UTF8),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		optionalValue, err := schema.NewElement("value", parquet.FieldRepetitionType_OPTIONAL,
			parquet.TypePtr(parquet.Type_INT32), nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err = requiredMap2.Set("map", mapElement); err != nil {
			t.Fatal(err)
		}

		if err = requiredMap2.Set("map.key_value", keyValue); err != nil {
			t.Fatal(err)
		}

		if err = requiredMap2.Set("map.key_value.key", requiredKey); err != nil {
			t.Fatal(err)
		}

		if err = requiredMap2.Set("map.key_value.value", optionalValue); err != nil {
			t.Fatal(err)
		}

		if _, _, err = requiredMap2.ToParquetSchema(); err != nil {
			t.Fatal(err)
		}
	}

	optionalMap1 := schema.NewTree()
	{
		mapElement, err := schema.NewElement("map", parquet.FieldRepetitionType_OPTIONAL,
			nil, parquet.ConvertedTypePtr(parquet.ConvertedType_MAP),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		keyValue, err := schema.NewElement("key_value", parquet.FieldRepetitionType_REPEATED,
			nil, nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		requiredKey, err := schema.NewElement("key", parquet.FieldRepetitionType_REQUIRED,
			parquet.TypePtr(parquet.Type_BYTE_ARRAY), parquet.ConvertedTypePtr(parquet.ConvertedType_UTF8),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		requiredValue, err := schema.NewElement("value", parquet.FieldRepetitionType_REQUIRED,
			parquet.TypePtr(parquet.Type_INT32), nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err = optionalMap1.Set("map", mapElement); err != nil {
			t.Fatal(err)
		}

		if err = optionalMap1.Set("map.key_value", keyValue); err != nil {
			t.Fatal(err)
		}

		if err = optionalMap1.Set("map.key_value.key", requiredKey); err != nil {
			t.Fatal(err)
		}

		if err = optionalMap1.Set("map.key_value.value", requiredValue); err != nil {
			t.Fatal(err)
		}

		if _, _, err = optionalMap1.ToParquetSchema(); err != nil {
			t.Fatal(err)
		}
	}

	optionalMap2 := schema.NewTree()
	{
		mapElement, err := schema.NewElement("map", parquet.FieldRepetitionType_OPTIONAL,
			nil, parquet.ConvertedTypePtr(parquet.ConvertedType_MAP),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		keyValue, err := schema.NewElement("key_value", parquet.FieldRepetitionType_REPEATED,
			nil, nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		requiredKey, err := schema.NewElement("key", parquet.FieldRepetitionType_REQUIRED,
			parquet.TypePtr(parquet.Type_BYTE_ARRAY), parquet.ConvertedTypePtr(parquet.ConvertedType_UTF8),
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		optionalValue, err := schema.NewElement("value", parquet.FieldRepetitionType_OPTIONAL,
			parquet.TypePtr(parquet.Type_INT32), nil,
			nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		if err = optionalMap2.Set("map", mapElement); err != nil {
			t.Fatal(err)
		}

		if err = optionalMap2.Set("map.key_value", keyValue); err != nil {
			t.Fatal(err)
		}

		if err = optionalMap2.Set("map.key_value.key", requiredKey); err != nil {
			t.Fatal(err)
		}

		if err = optionalMap2.Set("map.key_value.value", optionalValue); err != nil {
			t.Fatal(err)
		}

		if _, _, err = optionalMap2.ToParquetSchema(); err != nil {
			t.Fatal(err)
		}
	}

	result1 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{ten},
			definitionLevels: []int64{1},
			repetitionLevels: []int64{0},
		},
		"map.key_value.value": {
			parquetType:      parquet.Type_INT32,
			values:           []interface{}{v10},
			definitionLevels: []int64{1},
			repetitionLevels: []int64{1},
		},
	}

	result2 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{ten},
			definitionLevels: []int64{1},
			repetitionLevels: []int64{0},
		},
		"map.key_value.value": {
			parquetType:      parquet.Type_INT32,
			values:           []interface{}{nil},
			definitionLevels: []int64{1},
			repetitionLevels: []int64{1},
		},
	}

	result3 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{ten},
			definitionLevels: []int64{1},
			repetitionLevels: []int64{0},
		},
		"map.key_value.value": {
			parquetType:      parquet.Type_INT32,
			values:           []interface{}{v10},
			definitionLevels: []int64{2},
			repetitionLevels: []int64{1},
		},
	}

	result4 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{nil},
			definitionLevels: []int64{0},
			repetitionLevels: []int64{0},
		},
	}

	result5 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{ten},
			definitionLevels: []int64{2},
			repetitionLevels: []int64{0},
		},
		"map.key_value.value": {
			parquetType:      parquet.Type_INT32,
			values:           []interface{}{v10},
			definitionLevels: []int64{2},
			repetitionLevels: []int64{1},
		},
	}

	result6 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{ten},
			definitionLevels: []int64{2},
			repetitionLevels: []int64{0},
		},
		"map.key_value.value": {
			parquetType:      parquet.Type_INT32,
			values:           []interface{}{nil},
			definitionLevels: []int64{2},
			repetitionLevels: []int64{1},
		},
	}

	result7 := map[string]*Column{
		"map.key_value.key": {
			parquetType:      parquet.Type_BYTE_ARRAY,
			values:           []interface{}{ten},
			definitionLevels: []int64{2},
			repetitionLevels: []int64{0},
		},
		"map.key_value.value": {
			parquetType:      parquet.Type_INT32,
			values:           []interface{}{v10},
			definitionLevels: []int64{3},
			repetitionLevels: []int64{1},
		},
	}

	testCases := []struct {
		schemaTree     *schema.Tree
		data           string
		expectedResult map[string]*Column
		expectErr      bool
	}{
		{requiredMap1, `{}`, nil, true},                     // err: map: nil value for required field
		{requiredMap1, `{"map": null}`, nil, true},          // err: map: nil value for required field
		{requiredMap1, `{"map": {"ten": null}}`, nil, true}, // err: map.key_value.value: nil value for required field
		{requiredMap1, `{"map": {"ten": 10}}`, result1, false},
		{requiredMap2, `{}`, nil, true},            // err: map: nil value for required field
		{requiredMap2, `{"map": null}`, nil, true}, // err: map: nil value for required field
		{requiredMap2, `{"map": {"ten": null}}`, result2, false},
		{requiredMap2, `{"map": {"ten": 10}}`, result3, false},
		{optionalMap1, `{}`, result4, false},
		{optionalMap1, `{"map": null}`, result4, false},
		{optionalMap1, `{"map": {"ten": null}}`, nil, true}, // err: map.key_value.value: nil value for required field
		{optionalMap1, `{"map": {"ten": 10}}`, result5, false},
		{optionalMap2, `{}`, result4, false},
		{optionalMap2, `{"map": null}`, result4, false},
		{optionalMap2, `{"map": {"ten": null}}`, result6, false},
		{optionalMap2, `{"map": {"ten": 10}}`, result7, false},
	}

	for i, testCase := range testCases {
		result, err := UnmarshalJSON([]byte(testCase.data), testCase.schemaTree)
		expectErr := (err != nil)

		if testCase.expectErr != expectErr {
			t.Fatalf("case %v: error: expected: %v, got: %v", i+1, testCase.expectErr, expectErr)
		}

		if !testCase.expectErr {
			if !reflect.DeepEqual(result, testCase.expectedResult) {
				t.Errorf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedResult, result)
			}
		}
	}
}
