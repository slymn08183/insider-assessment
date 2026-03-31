package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice — store []string as JSONB
// Scan: DB -> Go  Value: Go -> DB
type StringSlice []string

func (s *StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringSlice) Scan(val interface{}) error {
	bytes, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan String Slice")
	}
	return json.Unmarshal(bytes, s)
}

// JSONMap — store map[string] as JSONB
// Scan: DB -> Go  Value: Go -> DB
type JSONMap map[string]interface{}

func (j *JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(val interface{}) error {
	bytes, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSON Map")
	}
	return json.Unmarshal(bytes, j)
}
