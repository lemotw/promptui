package promptui

import (
	"fmt"
	"testing"
)

func TestMultidimSelectTemplateRender(t *testing.T) {
	t.Run("when using default style", func(t *testing.T) {
		items := []interface{}{[]interface{}{
			"Option 1.1",
			[]interface{}{"Option 1.2.1", "Option 1.2.2"},
			"Option 1.3",
		},
			"Option 2",
			[]interface{}{
				"Option 3.1",
				"Option 3.2",
			},
			"Option 6",
			"Option 7",
			"Option 8",
			"Option 9",
			"Option 10",
			"Option 11",
		}

		s := MultidimSelect{
			Label: "Select Number",
			Items: items,
		}
		err := s.prepareTemplates()
		if err != nil {
			t.Fatalf("Unexpected error preparing templates %v", err)
		}

		result := string(render(s.Templates.label, s.Label))
		exp := "\x1b[34m?\x1b[0m Select Number: "
		if result != exp {
			t.Errorf("Expected label to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.active, items[0]))
		exp = "\x1b[1m▸\x1b[0m \x1b[4mOption 1.1 & [Option 1.2.1 Option 1.2.2] & Option 1.3\x1b[0m"
		if result != exp {
			t.Errorf("Expected active item to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.inactive, items[0]))
		exp = "Option 1.1 & [Option 1.2.1 Option 1.2.2] & Option 1.3"
		if result != exp {
			t.Errorf("Expected inactive item to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.selected, items[0]))
		exp = "\n\t\t\t\n\t\t\t\t\x1b[32m\x1b[32m✔\x1b[0m Option 1.1 & [Option 1.2.1 Option 1.2.2] & Option 1.3\n\t\t\t\n\t\t"
		if result != exp {
			t.Errorf("Expected selected item to eq %q, got %q", exp, result)
		}
	})

	t.Run("when using custom style", func(t *testing.T) {
		type pepper struct {
			Name        string
			HeatUnit    int
			Peppers     int
			Description string
		}
		peppers := []pepper{
			{
				Name:        "Bell Pepper",
				HeatUnit:    0,
				Peppers:     1,
				Description: "Not very spicy!",
			},
		}

		templates := &MultidimSelectTemplates{
			Label: "{{ . }}?",

			Active: fmt.Sprintf(`
            {{- if isSlice . -}}
                %s {{ joinSlice " : " . | underline }}
            {{- else -}}
                %s {{ . | underline }}
            {{- end -}}
			`, "\U0001F525", "\U0001F525"),
			Inactive: `
			{{- if isSlice . -}}
				{{ joinSlice " & " . }}
			{{- else -}}
				{{ . }}
			{{- end -}}`,
			Selected: fmt.Sprintf(`
			{{ if isSlice . }}
				{{ "%s" | green}} {{ joinSlice " : " . }}
			{{ else }}
			 	{{ "%s" | green}} {{ . }}
			{{ end }}
			`, "\U0001F525", "\U0001F525"),
			Details: `this is details, you select len {{ if isSlice . }} {{ sliceLen . }} {{ else }} 1 {{ end }}`,
		}

		s := MultidimSelect{
			Label:     "Spicy Level",
			Items:     peppers,
			Templates: templates,
		}

		err := s.prepareTemplates()
		if err != nil {
			t.Fatalf("Unexpected error preparing templates %v", err)
		}

		result := string(render(s.Templates.label, s.Label))
		exp := "Spicy Level?"
		if result != exp {
			t.Errorf("Expected label to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.active, peppers[0]))
		exp = "🔥 \x1b[4m{Bell Pepper 0 1 Not very spicy!}\x1b[0m"
		if result != exp {
			t.Errorf("Expected active item to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.inactive, peppers[0]))
		exp = "{Bell Pepper 0 1 Not very spicy!}"
		if result != exp {
			t.Errorf("Expected inactive item to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.selected, peppers[0]))
		exp = "\n\t\t\t\n\t\t\t \t\x1b[32m🔥\x1b[0m {Bell Pepper 0 1 Not very spicy!}\n\t\t\t\n\t\t\t"
		if result != exp {
			t.Errorf("Expected selected item to eq %q, got %q", exp, result)
		}

		result = string(render(s.Templates.details, peppers[0]))
		exp = "this is details, you select len  1 "
		if result != exp {
			t.Errorf("Expected selected item to eq %q, got %q", exp, result)
		}
	})

	t.Run("when a template is invalid", func(t *testing.T) {
		templates := &MultidimSelectTemplates{
			Label: "{{ . ",
		}

		s := MultidimSelect{
			Label:     "Spicy Level",
			Templates: templates,
		}

		err := s.prepareTemplates()
		if err == nil {
			t.Fatalf("Expected error got none")
		}
	})

	t.Run("when a template render fails", func(t *testing.T) {
		templates := &MultidimSelectTemplates{
			Label: "{{ .InvalidName }}",
		}

		s := MultidimSelect{
			Label:     struct{ Name string }{Name: "Pepper"},
			Items:     []string{},
			Templates: templates,
		}

		err := s.prepareTemplates()
		if err != nil {
			t.Fatalf("Unexpected error preparing templates %v", err)
		}

		result := string(render(s.Templates.label, s.Label))
		exp := "{Pepper}"
		if result != exp {
			t.Errorf("Expected label to eq %q, got %q", exp, result)
		}
	})
}
