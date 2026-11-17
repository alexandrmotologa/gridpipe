package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TransformationRule specifies targeted cleanups and synthesized calculated fields.
type TransformationRule struct {
	ColumnKey         string            `json:"column_key"`
	StripPrefix       string            `json:"strip_prefix,omitempty"`
	StripSuffix       string            `json:"strip_suffix,omitempty"`
	Uppercase         bool              `json:"uppercase,omitempty"`
	Lowercase         bool              `json:"lowercase,omitempty"`
	Trim              bool              `json:"trim,omitempty"`
	ValueReplacements map[string]string `json:"value_replacements,omitempty"`
	CalculatedExpr    string            `json:"calculated_expr,omitempty"` // e.g. "{Quantity} * {Unit_Price}"
}

// PipelineConfig defines runtime options for gridpipe operations.
type PipelineConfig struct {
	HeaderScanRows      int                  `json:"header_scan_rows"`
	FootnoteScanRows    int                  `json:"footnote_scan_rows"`
	DateTargetFormat    string               `json:"date_target_format"`
	NumericLocale       string               `json:"numeric_locale"` // auto, us, eu
	TrimFootnotes       bool                 `json:"trim_footnotes"`
	SynthesizeMath      bool                 `json:"synthesize_math"`
	RepairBrokenRefs    bool                 `json:"repair_broken_refs"`
	VarianceEpsilon     float64              `json:"variance_epsilon"`
	TransformationRules []TransformationRule `json:"transformation_rules,omitempty"`
}

// DefaultPipelineConfig returns recommended default processing parameters.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		HeaderScanRows:   10,
		FootnoteScanRows: 15,
		DateTargetFormat: "2006-01-02",
		NumericLocale:    "auto",
		TrimFootnotes:    true,
		SynthesizeMath:   true,
		RepairBrokenRefs: true,
		VarianceEpsilon:  0.0001,
	}
}

// LoadConfigFile reads a JSON configuration file if specified.
func LoadConfigFile(path string) (PipelineConfig, error) {
	cfg := DefaultPipelineConfig()
	if path == "" {
		return cfg, nil
	}

	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
