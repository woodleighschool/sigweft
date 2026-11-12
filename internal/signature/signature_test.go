package signature

import (
	"encoding/base64"
	htmltext "html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func studio(t *testing.T) *Renderer {
	t.Helper()
	r, err := Load("../../examples/studio/sigweft.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func person() map[string]any {
	return map[string]any{"display_name": "Jo Example", "disciplines": []string{"Designer", "Creative director"}, "contact": "jo@example.org", "base": "city"}
}

func TestChoiceResolution(t *testing.T) {
	r := studio(t)
	for location, phone := range map[string]string{"city": "+1 202 555 0101", "riverside": "+1 202 555 0102", "remote": "+1 202 555 0103"} {
		input := person()
		input["base"] = location
		result, err := r.Render(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Errors) != 0 || !strings.Contains(result.Text, phone) {
			t.Fatalf("%s: %v %s", location, result.Errors, result.Text)
		}
		if !strings.Contains(result.Text, "City studio") || !strings.Contains(result.Text, "Riverside studio") {
			t.Fatal("decorative locations changed")
		}
	}
	input := person()
	input["telephone"] = "+61 3 5550 0123"
	result, err := r.Render(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors["telephone"] == "" || strings.Contains(result.Text, "5550 0123") || !strings.Contains(htmltext.UnescapeString(result.HTML), `href="tel:+12025550101"`) {
		t.Fatal("submitted phone replaced location data")
	}
}

func TestSignatureEscapesFieldsAndDerivesLinks(t *testing.T) {
	r := studio(t)
	input := person()
	input["display_name"] = `<img src=x onerror=alert(1)>`
	input["schedule"] = "Monday at City studio\nFriday at Riverside studio"
	result, err := r.Render(input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.HTML, `<img src=x`) || !strings.Contains(result.HTML, "&lt;img") {
		t.Fatal("name was not escaped")
	}
	for _, expected := range []string{`href="mailto:jo@example.org"`, ">jo@example.org</a>"} {
		if !strings.Contains(result.HTML, expected) {
			t.Errorf("missing %s", expected)
		}
	}
	for _, expected := range []string{"Designer\nCreative director", "Monday at City studio\nFriday at Riverside studio"} {
		if !strings.Contains(result.Text, expected) {
			t.Errorf("missing lines %q in %q", expected, result.Text)
		}
	}
	if strings.Contains(result.HTML, "()") || strings.Contains(result.Text, "Mobile") || strings.Contains(result.HTML, "old@example.org") {
		t.Fatal("empty optional values or old links leaked")
	}
}

func TestInvalidChoicesAndInputTypesCannotReachTemplate(t *testing.T) {
	r := studio(t)
	input := person()
	input["base"] = "not-a-location"
	input["disciplines"] = []any{"Designer", map[string]string{"html": "<b>Bad</b>"}}
	input["other"] = "injected"
	result, err := r.Render(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"base", "disciplines", "other"} {
		if result.Errors[id] == "" {
			t.Errorf("missing error for %s", id)
		}
	}
	if strings.Contains(result.HTML, "12 Example Street") || strings.Contains(result.HTML, "injected") {
		t.Fatal("invalid choice or unknown data rendered")
	}
}

func TestGenericExampleAndReadOnlyFields(t *testing.T) {
	r, err := Load("../../examples/studio/sigweft.yaml")
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Render(map[string]any{"display_name": "Alex Example", "contact": "alex@example.org", "studio_name": "Forged", "available": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || strings.Contains(result.HTML, "Forged") || !strings.Contains(result.Text, "North Studio") || !strings.Contains(result.Text, "Available for new projects") {
		t.Fatalf("unexpected output: %v %s", result.Errors, result.HTML)
	}
	for _, field := range studio(t).Form().Fields {
		for _, option := range field.Options {
			if option.Data != nil {
				t.Fatal("form exposes administrator-owned data")
			}
		}
	}
}

func TestImageBytesAndDimensionsSurviveRendering(t *testing.T) {
	result, err := studio(t).Render(person())
	if err != nil {
		t.Fatal(err)
	}
	tokens := html.NewTokenizer(strings.NewReader(result.HTML))
	count := 0
	for {
		kind := tokens.Next()
		if kind == html.ErrorToken {
			break
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		token := tokens.Token()
		if token.Data != "img" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range token.Attr {
			attrs[attr.Key] = attr.Val
		}
		encoded, ok := strings.CutPrefix(attrs["src"], "data:image/png;base64,")
		if !ok {
			t.Fatal("image is not self-contained")
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		file, dimensions := "emblem.png", "48x48"
		if count == 1 {
			file, dimensions = "campaign.png", "520x104"
		}
		original, err := os.ReadFile(filepath.Join("../../examples/studio/assets", file))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(original) || attrs["width"]+"x"+attrs["height"] != dimensions || attrs["alt"] == "" {
			t.Fatalf("image %d changed", count)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("got %d images", count)
	}
}

func TestConfigFilesStayInsideTheirDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "outside.html"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config/config.yaml")
	data := "title: Test\ntemplate: ../outside.html\nfields: [{id: name, label: Name, type: text}]\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("allowed template outside config directory")
	}
}
