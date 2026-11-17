package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TargetColumn represents an expected schema column.
type TargetColumn struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // text, integer, float, currency, date, boolean
	Aliases     []string `json:"aliases,omitempty"`
	Required    bool     `json:"required"`
	DefaultVal  string   `json:"default_val,omitempty"`
	DateFormat  string   `json:"date_format,omitempty"`
	Synthesize  string   `json:"synthesize,omitempty"` // e.g. SUM, AVERAGE, XLOOKUP
}

// TableSchema defines mapping expectations for an enterprise tabular report.
type TableSchema struct {
	SchemaName string         `json:"schema_name"`
	Columns    []TargetColumn `json:"columns"`
	KeyColumns []string       `json:"key_columns"`
}

// LoadSchemaFile reads a schema JSON file from disk.
func LoadSchemaFile(path string) (*TableSchema, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var schema TableSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}
