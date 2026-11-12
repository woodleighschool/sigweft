package config

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Empty returns the template value used when an optional field is absent.
func (f Field) Empty() any {
	switch f.Type {
	case "text-list":
		return []string{}
	case "checkbox":
		return false
	default:
		return ""
	}
}

// Normalize validates the configured type and removes incidental whitespace.
func (f Field) Normalize(input any) (any, error) {
	if input == nil {
		input = f.Empty()
	}
	switch f.Type {
	case "checkbox":
		value, ok := input.(bool)
		if !ok {
			return false, fmt.Errorf("choose yes or no")
		}
		if f.Required && !value {
			return false, fmt.Errorf("this field is required")
		}
		return value, nil
	case "text-list":
		return f.normalizeList(input)
	default:
		return f.normalizeText(input)
	}
}

func (f Field) normalizeList(input any) ([]string, error) {
	var list []any
	switch value := input.(type) {
	case []any:
		list = value
	case []string:
		for _, item := range value {
			list = append(list, item)
		}
	default:
		return nil, fmt.Errorf("enter a list of text values")
	}
	if len(list) > 20 {
		return nil, fmt.Errorf("use at most 20 entries")
	}
	result := make([]string, 0, len(list))
	itemField := f
	itemField.Type = "text"
	itemField.Required = false
	for _, item := range list {
		value, err := itemField.normalizeText(item)
		if err != nil {
			return nil, err
		}
		if value != "" {
			result = append(result, value)
		}
	}
	if f.Required && len(result) == 0 {
		return nil, fmt.Errorf("this field is required")
	}
	return result, nil
}

func (f Field) normalizeText(input any) (string, error) {
	value, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("enter text")
	}
	value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
	if value == "" {
		if f.Required {
			return "", fmt.Errorf("this field is required")
		}
		return "", nil
	}
	if utf8.RuneCountInString(value) > f.MaxLength {
		return "", fmt.Errorf("use at most %d characters", f.MaxLength)
	}
	if strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && (r != '\n' || f.Type != "multiline")
	}) {
		return "", fmt.Errorf("contains unsupported control characters")
	}
	switch f.Type {
	case "email":
		address, err := mail.ParseAddress(value)
		if err != nil || address.Address != value || address.Name != "" {
			return "", fmt.Errorf("enter an email address")
		}
	case "tel":
		if !validPhone(value) {
			return "", fmt.Errorf("use a phone number with digits, spaces, +, brackets or hyphens")
		}
	case "select":
		for _, option := range f.Options {
			if option.Value == value {
				return value, nil
			}
		}
		return "", fmt.Errorf("choose one of the available options")
	}
	return value, nil
}

func validPhone(value string) bool {
	digits := 0
	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' && i == 0, r == ' ', r == '(', r == ')', r == '-', r == '.':
		default:
			return false
		}
	}
	return digits >= 3 && digits <= 15
}
