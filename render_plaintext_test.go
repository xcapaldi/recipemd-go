package recipemd

import (
	"testing"
)

func TestRenderPlainText(t *testing.T) {
	t.Parallel()
	p := NewParser()

	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "plain text unchanged",
			markdown: "Hello world",
			want:     "Hello world",
		},
		{
			name:     "strips bold",
			markdown: "**bold** text",
			want:     "bold text",
		},
		{
			name:     "strips italic",
			markdown: "*italic* text",
			want:     "italic text",
		},
		{
			name:     "strips bold and italic",
			markdown: "**bold** and *italic* text",
			want:     "bold and italic text",
		},
		{
			name:     "extracts link text",
			markdown: "[click here](https://example.com)",
			want:     "click here",
		},
		{
			name:     "heading becomes text",
			markdown: "## Title\n\nBody text",
			want:     "Title\n\nBody text",
		},
		{
			name:     "multiple paragraphs",
			markdown: "First paragraph.\n\nSecond paragraph.",
			want:     "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "code span preserved",
			markdown: "Use `fmt.Println` to print",
			want:     "Use fmt.Println to print",
		},
		{
			name:     "fenced code block",
			markdown: "```\nfoo := 1\nbar := 2\n```",
			want:     "foo := 1\nbar := 2",
		},
		{
			name:     "image alt text",
			markdown: "![a photo](image.jpg)",
			want:     "a photo",
		},
		{
			name:     "image no alt text",
			markdown: "![](image.jpg)",
			want:     "",
		},
		{
			name:     "empty string",
			markdown: "",
			want:     "",
		},
		{
			name:     "thematic break stripped",
			markdown: "Above\n\n---\n\nBelow",
			want:     "Above\n\nBelow",
		},
		{
			name:     "unordered list",
			markdown: "- item one\n- item two",
			want:     "- item one\n- item two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := p.RenderPlainText(tt.markdown)
			if got != tt.want {
				t.Errorf("RenderPlainText(%q)\ngot:  %q\nwant: %q", tt.markdown, got, tt.want)
			}
		})
	}
}

func TestRenderPlainText_GFM(t *testing.T) {
	t.Parallel()
	p := NewParser(WithGithubFormattedMarkdown())

	t.Run("table", func(t *testing.T) {
		t.Parallel()
		md := "| A | B |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |"
		got := p.RenderPlainText(md)
		if got == "" {
			t.Error("expected non-empty output for table")
		}
		// Should contain cell values.
		for _, want := range []string{"A", "B", "1", "2", "3", "4"} {
			if !contains(got, want) {
				t.Errorf("table plain text %q missing %q", got, want)
			}
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
