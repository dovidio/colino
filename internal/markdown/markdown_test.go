package markdown

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestConverter_ConvertHTMLString(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "empty string",
			html:     "",
			expected: "",
		},
		{
			name:     "whitespace only",
			html:     "   \n\t  ",
			expected: "",
		},
		{
			name:     "simple text",
			html:     "Hello World",
			expected: "Hello World",
		},
		{
			name:     "paragraph",
			html:     "<p>Hello World</p>",
			expected: "\n\nHello World\n\n",
		},
		{
			name:     "heading 1",
			html:     "<h1>Heading 1</h1>",
			expected: "\n# Heading 1\n\n",
		},
		{
			name:     "heading 2",
			html:     "<h2>Heading 2</h2>",
			expected: "\n## Heading 2\n\n",
		},
		{
			name:     "heading 3",
			html:     "<h3>Heading 3</h3>",
			expected: "\n### Heading 3\n\n",
		},
		{
			name:     "heading 4",
			html:     "<h4>Heading 4</h4>",
			expected: "\n#### Heading 4\n\n",
		},
		{
			name:     "heading 5",
			html:     "<h5>Heading 5</h5>",
			expected: "\n##### Heading 5\n\n",
		},
		{
			name:     "heading 6",
			html:     "<h6>Heading 6</h6>",
			expected: "\n###### Heading 6\n\n",
		},
		{
			name:     "strong text",
			html:     "<strong>Bold text</strong>",
			expected: "**Bold text**",
		},
		{
			name:     "bold text",
			html:     "<b>Bold text</b>",
			expected: "**Bold text**",
		},
		{
			name:     "emphasis text",
			html:     "<em>Italic text</em>",
			expected: "*Italic text*",
		},
		{
			name:     "italic text",
			html:     "<i>Italic text</i>",
			expected: "*Italic text*",
		},
		{
			name:     "link",
			html:     `<a href="https://example.com">Example</a>`,
			expected: "[Example](https://example.com)",
		},
		{
			name:     "link without href",
			html:     "<a>Just text</a>",
			expected: "Just text",
		},
		{
			name:     "line break",
			html:     "<br>",
			expected: "\n",
		},
		{
			name:     "horizontal rule",
			html:     "<hr>",
			expected: "\n---\n",
		},
		{
			name:     "inline code",
			html:     "<code>console.log('hello')</code>",
			expected: "`console.log('hello')`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertHTMLString(tt.html)
			if result != tt.expected {
				t.Errorf("ConvertHTMLString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConverter_Convert(t *testing.T) {
	converter := NewConverter()

	// Test with nil node
	result := converter.Convert(nil)
	if result != "" {
		t.Errorf("Convert(nil) = %q, want empty string", result)
	}

	// Test with a simple text node
	textNode := &html.Node{
		Type: html.TextNode,
		Data: "Hello World",
	}
	result = converter.Convert(textNode)
	if result != "Hello World" {
		t.Errorf("Convert(textNode) = %q, want %q", result, "Hello World")
	}
}

func TestConverter_ComplexHTML(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		html     string
		contains []string // strings that should be in the result
	}{
		{
			name: "nested formatting",
			html: "<p>This is <strong>bold and <em>italic</em></strong> text</p>",
			contains: []string{
				"\n\nThis is **bold and *italic*** text\n\n",
			},
		},
		{
			name: "unordered list",
			html: "<ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul>",
			contains: []string{
				"\n\n- Item 1\n- Item 2\n- Item 3\n\n",
			},
		},
		{
			name: "ordered list",
			html: "<ol><li>First</li><li>Second</li><li>Third</li></ol>",
			contains: []string{
				"\n\n1. First\n1. Second\n1. Third\n\n",
			},
		},
		{
			name: "code block",
			html: "<pre>function hello() {\n  console.log('Hello');\n}</pre>",
			contains: []string{
				"\n\n```\nfunction hello() {\n  console.log('Hello');\n}\n```\n\n",
			},
		},
		{
			name: "blockquote",
			html: "<blockquote>This is a quote\nWith multiple lines</blockquote>",
			contains: []string{
				"\n\n> This is a quote\n> With multiple lines\n\n",
			},
		},
		{
			name: "mixed content",
			html: `<h1>Title</h1><p>Introduction paragraph with <a href="http://example.com">a link</a>.</p><ul><li>List item 1</li><li>Item with <strong>bold text</strong></li></ul>`,
			contains: []string{
				"# Title",
				"Introduction paragraph with [a link](http://example.com).",
				"- List item 1",
				"- Item with **bold text**",
			},
		},
		{
			name: "container elements",
			html: "<div><p>Paragraph in div</p><span>Text in span</span></div>",
			contains: []string{
				"Paragraph in div",
				"Text in span",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertHTMLString(tt.html)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertHTMLString() result = %q, should contain %q", result, expected)
				}
			}
		})
	}
}

func TestConverter_EdgeCases(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "empty paragraph",
			html:     "<p></p>",
			expected: "",
		},
		{
			name:     "paragraph with whitespace",
			html:     "<p>   \n\t  </p>",
			expected: "",
		},
		{
			name:     "empty heading",
			html:     "<h1></h1>",
			expected: "",
		},
		{
			name:     "empty strong",
			html:     "<strong></strong>",
			expected: "",
		},
		{
			name:     "empty link",
			html:     `<a href=""></a>`,
			expected: "",
		},
		{
			name:     "malformed HTML",
			html:     "<p>Unclosed paragraph",
			expected: "\n\nUnclosed paragraph\n\n",
		},
		{
			name:     "nested empty elements",
			html:     "<p><strong></strong><em></em></p>",
			expected: "",
		},
		{
			name:     "attributes on elements",
			html:     `<h1 class="title" id="main-title">Title with attributes</h1>`,
			expected: "\n# Title with attributes\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertHTMLString(tt.html)
			if result != tt.expected {
				t.Errorf("ConvertHTMLString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains string
	}{
		{
			name:     "ConvertHTMLString function",
			html:     "<p>Test paragraph</p>",
			contains: "Test paragraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertHTMLString(tt.html)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("ConvertHTMLString() = %q, should contain %q", result, tt.contains)
			}
		})
	}

	// Test ConvertNode function
	doc, err := html.Parse(strings.NewReader("<p>Test</p>"))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	result := ConvertNode(doc)
	if !strings.Contains(result, "Test") {
		t.Errorf("ConvertNode() = %q, should contain %q", result, "Test")
	}
}

func TestConverter_RealWorldExamples(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		html     string
		contains []string
	}{
		{
			name: "blog post excerpt",
			html: `<article>
				<h2>Understanding Go Concurrency</h2>
				<p>Go provides powerful concurrency primitives through <strong>goroutines</strong> and <em>channels</em>.</p>
				<p>Here's a simple example:</p>
				<pre>go func() {
    fmt.Println("Hello from goroutine!")
}()</pre>
				<p>Read more at <a href="https://golang.org">golang.org</a></p>
			</article>`,
			contains: []string{
				"## Understanding Go Concurrency",
				"Go provides powerful concurrency primitives through **goroutines** and *channels*.",
				"```\ngo func() {\n    fmt.Println(\"Hello from goroutine!\")\n}()\n```",
				"[golang.org](https://golang.org)",
			},
		},
		{
			name: "news article structure",
			html: `<div class="article">
				<h1>Breaking News: New Technology Released</h1>
				<p class="subtitle">A revolutionary breakthrough in computing</p>
				<blockquote>
					<p>"This changes everything we know about processing," said the lead researcher.</p>
				</blockquote>
				<p>The technology, which uses <strong>quantum computing</strong> principles, promises to:</p>
				<ul>
					<li>Process information 100x faster</li>
					<li>Reduce energy consumption by 90%</li>
					<li>Enable new applications in AI and ML</li>
				</ul>
				<p><em>Story continues below...</em></p>
			</div>`,
			contains: []string{
				"# Breaking News: New Technology Released",
				"> \"This changes everything we know about processing,\" said the lead researcher.",
				"The technology, which uses **quantum computing** principles, promises to:",
				"- Process information 100x faster",
				"- Reduce energy consumption by 90%",
				"- Enable new applications in AI and ML",
				"*Story continues below...*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ConvertHTMLString(tt.html)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("ConvertHTMLString() result = %q, should contain %q", result, expected)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkConverter_ConvertHTMLString_Simple(b *testing.B) {
	converter := NewConverter()
	html := "<p>This is a simple paragraph with <strong>bold</strong> and <em>italic</em> text.</p>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.ConvertHTMLString(html)
	}
}

func BenchmarkConverter_ConvertHTMLString_Complex(b *testing.B) {
	converter := NewConverter()
	html := `<article>
		<h1>Complex Article Title</h1>
		<p>This is a <strong>complex</strong> article with <em>various</em> formatting elements.</p>
		<ul>
			<li>First item with <a href="http://example.com">a link</a></li>
			<li>Second item with <code>inline code</code></li>
			<li>Third item with <strong>bold text</strong></li>
		</ul>
		<blockquote>
			<p>This is a quote with <em>emphasis</em> inside.</p>
		</blockquote>
		<pre>function example() {
    console.log("Hello, world!");
}</pre>
	</article>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.ConvertHTMLString(html)
	}
}
