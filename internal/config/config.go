// Package config loads and validates organisation configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Config defines the form, template and assets for one organisation.
type Config struct {
	Title        string           `json:"title" yaml:"title" jsonschema:"minLength=1"`
	Description  string           `json:"description,omitempty" yaml:"description,omitempty"`
	EmailClients []string         `json:"email_clients,omitempty" yaml:"email_clients,omitempty"`
	Template     string           `json:"template" yaml:"template" jsonschema:"minLength=1"`
	Data         map[string]any   `json:"data,omitempty" yaml:"data,omitempty"`
	Fields       []Field          `json:"fields" yaml:"fields" jsonschema:"minItems=1"`
	Assets       map[string]Asset `json:"assets,omitempty" yaml:"assets,omitempty"`
}

// Field describes a form control. IDs are the template's field keys.
type Field struct {
	ID          string   `json:"id" yaml:"id" jsonschema:"pattern=^[a-z][a-z0-9_]*$"`
	Label       string   `json:"label" yaml:"label" jsonschema:"minLength=1"`
	Type        string   `json:"type" yaml:"type" enum:"text,email,tel,multiline,text-list,select,checkbox" jsonschema:"enum=text,enum=email,enum=tel,enum=multiline,enum=text-list,enum=select,enum=checkbox"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Placeholder string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Required    bool     `json:"required,omitempty" yaml:"required,omitempty"`
	ReadOnly    bool     `json:"read_only,omitempty" yaml:"read_only,omitempty"`
	Prefill     string   `json:"prefill,omitempty" yaml:"prefill,omitempty" enum:"name,email" jsonschema:"enum=name,enum=email"`
	Default     any      `json:"default,omitempty" yaml:"default,omitempty"`
	MaxLength   int      `json:"max_length,omitempty" yaml:"max_length,omitempty" jsonschema:"minimum=1"`
	Options     []Option `json:"options,omitempty" yaml:"options,omitempty"`
}

// Option supplies a label and administrator-owned template data for a selection.
type Option struct {
	Value string         `json:"value" yaml:"value" jsonschema:"minLength=1"`
	Label string         `json:"label" yaml:"label" jsonschema:"minLength=1"`
	Data  map[string]any `json:"data,omitempty" yaml:"data,omitempty"`
}

// Asset defines a raster image source and its display dimensions.
type Asset struct {
	Src    string `json:"src" yaml:"src" jsonschema:"minLength=1"`
	Alt    string `json:"alt" yaml:"alt"`
	Width  int    `json:"width" yaml:"width" jsonschema:"minimum=1"`
	Height int    `json:"height" yaml:"height" jsonschema:"minimum=1"`
	Href   string `json:"href,omitempty" yaml:"href,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Load rejects unknown keys, duplicate mappings and invalid field definitions.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // The administrator supplies the configuration path at startup.
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return Parse(data)
}

// Parse validates a single YAML document without accessing template or asset files.
func Parse(data []byte) (*Config, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("config must contain exactly one YAML document")
	}
	if strings.TrimSpace(cfg.Title) == "" || cfg.Template == "" || len(cfg.Fields) == 0 {
		return nil, fmt.Errorf("title, template and at least one field are required")
	}
	ids := make(map[string]bool)
	for i := range cfg.Fields {
		field := &cfg.Fields[i]
		if !identifier.MatchString(field.ID) || ids[field.ID] {
			return nil, fmt.Errorf("field %q must have a unique identifier", field.ID)
		}
		ids[field.ID] = true
		if err := field.validate(); err != nil {
			return nil, fmt.Errorf("field %q: %w", field.ID, err)
		}
	}
	for id, asset := range cfg.Assets {
		if !identifier.MatchString(id) || strings.TrimSpace(asset.Src) == "" || asset.Width <= 0 || asset.Height <= 0 {
			return nil, fmt.Errorf("asset %q needs an identifier, source and positive dimensions", id)
		}
	}
	for _, client := range cfg.EmailClients {
		switch client {
		case "outlook-web", "outlook-mac", "outlook-windows", "apple-mail":
		default:
			return nil, fmt.Errorf("unknown email client %q", client)
		}
	}
	return &cfg, nil
}

func (f *Field) validate() error {
	if strings.TrimSpace(f.Label) == "" {
		return fmt.Errorf("label is required")
	}
	switch f.Type {
	case "text", "email", "tel", "multiline", "text-list", "select", "checkbox":
	default:
		return fmt.Errorf("unknown type %q", f.Type)
	}
	if f.Prefill != "" {
		if f.Prefill != "name" && f.Prefill != "email" {
			return fmt.Errorf("prefill must be name or email")
		}
		if (f.Type != "text" && f.Type != "email") || f.ReadOnly {
			return fmt.Errorf("prefill requires an editable text or email field")
		}
	}
	if f.MaxLength < 0 {
		return fmt.Errorf("max_length must be positive")
	}
	if f.MaxLength == 0 {
		f.MaxLength = 200
		if f.Type == "multiline" {
			f.MaxLength = 1000
		}
	}
	if f.Type == "select" && len(f.Options) == 0 {
		return fmt.Errorf("select needs options")
	}
	if f.Type != "select" && len(f.Options) != 0 {
		return fmt.Errorf("only select fields support options")
	}
	values := make(map[string]bool)
	for _, option := range f.Options {
		if strings.TrimSpace(option.Value) == "" || strings.TrimSpace(option.Label) == "" || values[option.Value] {
			return fmt.Errorf("options need unique nonempty values and labels")
		}
		values[option.Value] = true
		normalized, err := f.Normalize(option.Value)
		if err != nil || normalized != option.Value {
			return fmt.Errorf("option %q cannot be submitted unchanged", option.Value)
		}
	}
	if f.ReadOnly && f.Default == nil {
		return fmt.Errorf("read-only fields need a default")
	}
	if f.Default != nil {
		value, err := f.Normalize(f.Default)
		if err != nil {
			return fmt.Errorf("default: %w", err)
		}
		f.Default = value
	}
	return nil
}
