// Package signature resolves configured fields and renders email HTML.
package signature

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/woodleighschool/sigweft/internal/config"
)

// Renderer holds one validated configuration and its compiled template.
type Renderer struct {
	config   *config.Config
	template *template.Template
	assets   map[string]Asset
}

// Asset exposes a validated, self-contained image to templates.
type Asset struct {
	Src    template.URL
	Alt    string
	Width  int
	Height int
	Href   string
}

// Result is the exact HTML copied by the browser, with its plain-text alternative.
type Result struct {
	HTML   string            `json:"html"`
	Text   string            `json:"text"`
	Errors map[string]string `json:"errors"`
}

// Form contains only browser-facing field metadata, without templates or option data.
type Form struct {
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	EmailClients []string       `json:"email_clients"`
	Fields       []config.Field `json:"fields"`
}

type templateData struct {
	Fields  map[string]any
	Choices map[string]map[string]any
	Data    map[string]any
	Assets  map[string]Asset
}

// Load reads configuration, templates and images once. Local paths are confined to the configuration directory.
func Load(ctx context.Context, path string) (*Renderer, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("open config directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	source, err := root.ReadFile(cfg.Template)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	tmpl, err := template.New(cfg.Template).Option("missingkey=error").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
		"lines": lines,
		"tel":   phoneURL,
	}).Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	r := &Renderer{config: cfg, template: tmpl, assets: make(map[string]Asset)}
	for id, definition := range cfg.Assets {
		asset, err := loadAsset(ctx, root, definition)
		if err != nil {
			return nil, fmt.Errorf("asset %q: %w", id, err)
		}
		r.assets[id] = asset
	}
	if _, err := r.Render(nil); err != nil {
		return nil, fmt.Errorf("validate template: %w", err)
	}
	return r, nil
}

// Form strips administrator-owned data before returning form metadata.
func (r *Renderer) Form() Form {
	fields := make([]config.Field, len(r.config.Fields))
	for i, field := range r.config.Fields {
		fields[i] = field
		if field.Type == "select" {
			fields[i].Options = make([]config.Option, len(field.Options))
			for j, option := range field.Options {
				fields[i].Options[j] = config.Option{Value: option.Value, Label: option.Label}
			}
		}
	}
	return Form{Title: r.config.Title, Description: r.config.Description, EmailClients: r.config.EmailClients, Fields: fields}
}

// Render resolves every value on the server. Invalid fields render empty and prevent copying.
func (r *Renderer) Render(input map[string]any) (Result, error) {
	ctx := templateData{
		Fields: make(map[string]any), Choices: make(map[string]map[string]any),
		Data: r.config.Data, Assets: r.assets,
	}
	issues := make(map[string]string)
	for _, field := range r.config.Fields {
		value, exists := input[field.ID]
		if !exists || field.ReadOnly {
			value = field.Default
		}
		normalized, err := field.Normalize(value)
		if err != nil {
			issues[field.ID] = err.Error()
			normalized = field.Empty()
		}
		ctx.Fields[field.ID] = normalized
		if field.Type == "select" {
			ctx.Choices[field.ID] = nil
			for _, option := range field.Options {
				if option.Value == normalized {
					ctx.Choices[field.ID] = option.Data
				}
			}
		}
	}
	for id := range input {
		if _, exists := ctx.Fields[id]; !exists {
			issues[id] = "unknown field"
		}
	}
	var out bytes.Buffer
	if err := r.template.Execute(&out, ctx); err != nil {
		return Result{}, fmt.Errorf("render signature: %w", err)
	}
	html := out.String()
	return Result{HTML: html, Text: plainText(html), Errors: issues}, nil
}

func lines(value string) []string {
	result := make([]string, 0)
	for line := range strings.SplitSeq(value, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result
}

func phoneURL(value string) template.URL {
	var number strings.Builder
	for i, r := range value {
		if (r >= '0' && r <= '9') || (r == '+' && i == 0) {
			number.WriteRune(r)
		}
	}
	return template.URL("tel:" + number.String()) //nolint:gosec // The scheme is fixed and the suffix contains only digits and an initial plus.
}
