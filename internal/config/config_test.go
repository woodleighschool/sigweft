package config

import (
	"strings"
	"testing"
)

func TestConfigRejectsInvalidDefinitions(t *testing.T) {
	for name, field := range map[string]string{
		"unknown key":              "id: name\n  label: Name\n  type: text\n  requireds: true",
		"unknown type":             "id: name\n  label: Name\n  type: html",
		"duplicate ID":             "id: name\n  label: Name\n  type: text\n- id: name\n  label: Other\n  type: text",
		"invalid default":          "id: name\n  label: Name\n  type: email\n  default: not-an-email",
		"unknown choice":           "id: office\n  label: Office\n  type: select\n  default: missing\n  options: [{value: a, label: A}]",
		"duplicate choices":        "id: office\n  label: Office\n  type: select\n  options: [{value: a, label: A}, {value: a, label: B}]",
		"choice whitespace":        "id: office\n  label: Office\n  type: select\n  options: [{value: 'city ', label: City}]",
		"choice aliases another":   "id: office\n  label: Office\n  type: select\n  options: [{value: city, label: City}, {value: 'city ', label: Other}]",
		"choice too long":          "id: office\n  label: Office\n  type: select\n  max_length: 3\n  options: [{value: city, label: City}]",
		"choice control character": "id: office\n  label: Office\n  type: select\n  options: [{value: \"ci\\tty\", label: City}]",
		"readonly without default": "id: name\n  label: Name\n  type: text\n  read_only: true",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte("title: Example\ntemplate: signature.html\nfields:\n- " + field + "\n"))
			if err == nil {
				t.Fatal("accepted invalid field definition")
			}
		})
	}
	for _, data := range []string{
		"title: One\ntitle: Two\ntemplate: t.html\nfields: [{id: n, label: Name, type: text}]",
		"title: Example\ntemplate: t.html\nfields: [{id: n, label: Name, type: text}]\n---\ntitle: Other",
	} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatal("accepted duplicate key or extra document")
		}
	}
}

func TestFieldValidation(t *testing.T) {
	for _, tc := range []struct {
		kind      string
		value     any
		wantError bool
	}{
		{"email", "jo@example.org", false},
		{"email", "Jo <jo@example.org>", true},
		{"tel", "+1 (202) 555 0101", false},
		{"tel", "javascript:alert(1)", true},
		{"text", "One\nTwo", true},
		{"multiline", "Monday\nWednesday", false},
		{"checkbox", "false", true},
		{"text-list", []any{" One ", "", "Two"}, false},
		{"text-list", []any{"One", 2}, true},
		{"text", strings.Repeat("a", 201), true},
	} {
		f := Field{Type: tc.kind, MaxLength: 200}
		_, err := f.Normalize(tc.value)
		if (err != nil) != tc.wantError {
			t.Errorf("%s %v: error = %v", tc.kind, tc.value, err)
		}
	}
}

func TestEmailClients(t *testing.T) {
	base := "title: Studio\ntemplate: signature.html\nemail_clients: [outlook-web, apple-mail]\nfields: [{id: name, label: Name, type: text}]\n"
	if _, err := Parse([]byte(base)); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse([]byte(strings.Replace(base, "outlook-web", "invented-client", 1))); err == nil {
		t.Fatal("accepted unknown email client")
	}
}
