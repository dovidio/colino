package markdown

import (
	"golang.org/x/net/html"
	"strings"
)

// Converter handles HTML to Markdown conversion using node traversal
type Converter struct {
	// Configuration options can be added here in the future
}

// NewConverter creates a new HTML to Markdown converter
func NewConverter() *Converter {
	return &Converter{}
}

// Convert converts an HTML node to markdown
func (c *Converter) Convert(node *html.Node) string {
	if node == nil {
		return ""
	}
	return c.convertNode(node)
}

// ConvertHTMLString converts an HTML string to markdown
func (c *Converter) ConvertHTMLString(htmlStr string) string {
	if strings.TrimSpace(htmlStr) == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	// Find the body element and convert from there
	var body *html.Node
	var findBody func(*html.Node)
	findBody = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			findBody(child)
		}
	}
	findBody(doc)

	if body != nil {
		return c.convertNode(body)
	}

	// If no body found, convert from the document root
	return c.convertNode(doc)
}

// convertNode recursively converts HTML nodes to markdown
func (c *Converter) convertNode(node *html.Node) string {
	if node == nil {
		return ""
	}

	switch node.Type {
	case html.TextNode:
		return node.Data
	case html.ElementNode:
		return c.convertElement(node)
	case html.DocumentNode:
		return c.convertChildren(node)
	default:
		return ""
	}
}

// convertElement handles HTML element nodes
func (c *Converter) convertElement(node *html.Node) string {
	switch strings.ToLower(node.Data) {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return c.convertHeading(node)
	case "p":
		return c.convertParagraph(node)
	case "strong", "b":
		return c.convertStrong(node)
	case "em", "i":
		return c.convertEmphasis(node)
	case "a":
		return c.convertLink(node)
	case "br":
		return "\n"
	case "ul", "ol":
		return c.convertList(node)
	case "li":
		return c.convertListItem(node)
	case "code":
		return c.convertCode(node)
	case "pre":
		return c.convertPre(node)
	case "blockquote":
		return c.convertBlockquote(node)
	case "hr":
		return "\n---\n"
	case "div", "span", "section", "article", "header", "footer", "main", "aside":
		// Container elements - just process children
		return c.convertChildren(node)
	default:
		// Unknown elements - just process children
		return c.convertChildren(node)
	}
}

// convertHeading converts heading elements (h1-h6) to markdown
func (c *Converter) convertHeading(node *html.Node) string {
	level := 1
	switch strings.ToLower(node.Data) {
	case "h1":
		level = 1
	case "h2":
		level = 2
	case "h3":
		level = 3
	case "h4":
		level = 4
	case "h5":
		level = 5
	case "h6":
		level = 6
	}

	content := strings.TrimSpace(c.convertChildren(node))
	if content == "" {
		return ""
	}

	return "\n" + strings.Repeat("#", level) + " " + content + "\n\n"
}

// convertParagraph converts paragraph elements to markdown
func (c *Converter) convertParagraph(node *html.Node) string {
	content := strings.TrimSpace(c.convertChildren(node))
	if content == "" {
		return ""
	}

	return "\n\n" + content + "\n\n"
}

// convertStrong converts strong/bold elements to markdown
func (c *Converter) convertStrong(node *html.Node) string {
	content := c.convertChildren(node)
	if content == "" {
		return ""
	}

	return "**" + content + "**"
}

// convertEmphasis converts italic/emphasis elements to markdown
func (c *Converter) convertEmphasis(node *html.Node) string {
	content := c.convertChildren(node)
	if content == "" {
		return ""
	}

	return "*" + content + "*"
}

// convertLink converts anchor elements to markdown
func (c *Converter) convertLink(node *html.Node) string {
	content := c.convertChildren(node)
	if content == "" {
		return ""
	}

	href := ""
	for _, attr := range node.Attr {
		if strings.ToLower(attr.Key) == "href" {
			href = attr.Val
			break
		}
	}

	if href == "" {
		return content
	}

	return "[" + content + "](" + href + ")"
}

// convertList converts list elements (ul/ol) to markdown
func (c *Converter) convertList(node *html.Node) string {
	content := c.convertChildren(node)
	if strings.TrimSpace(content) == "" {
		return ""
	}

	// Add proper spacing around lists
	return "\n\n" + content + "\n"
}

// convertListItem converts list item elements to markdown
func (c *Converter) convertListItem(node *html.Node) string {
	content := strings.TrimSpace(c.convertChildren(node))
	if content == "" {
		return ""
	}

	// Check if this is an ordered list by looking at parent
	parent := node.Parent
	isOrdered := parent != nil && strings.ToLower(parent.Data) == "ol"

	if isOrdered {
		// For ordered lists, we'd need to track the index
		// For now, using a simple approach
		return "1. " + content + "\n"
	}

	return "- " + content + "\n"
}

// convertCode converts inline code elements to markdown
func (c *Converter) convertCode(node *html.Node) string {
	content := c.convertChildren(node)
	if content == "" {
		return ""
	}

	// Use backticks for inline code
	return "`" + content + "`"
}

// convertPre converts preformatted text elements to markdown
func (c *Converter) convertPre(node *html.Node) string {
	content := c.convertChildren(node)
	if content == "" {
		return ""
	}

	// Use fenced code blocks
	return "\n\n```\n" + content + "\n```\n\n"
}

// convertBlockquote converts blockquote elements to markdown
func (c *Converter) convertBlockquote(node *html.Node) string {
	content := c.convertChildren(node)
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var quotedLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			quotedLines = append(quotedLines, ">")
		} else {
			quotedLines = append(quotedLines, "> "+strings.TrimSpace(line))
		}
	}

	return "\n\n" + strings.Join(quotedLines, "\n") + "\n\n"
}

// convertChildren processes all child nodes of a given node
func (c *Converter) convertChildren(node *html.Node) string {
	var result strings.Builder

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result.WriteString(c.convertNode(child))
	}

	return result.String()
}

// ConvertHTMLString is a convenience function that converts an HTML string to markdown
func ConvertHTMLString(htmlStr string) string {
	converter := NewConverter()
	return converter.ConvertHTMLString(htmlStr)
}

// ConvertNode is a convenience function that converts an HTML node to markdown
func ConvertNode(node *html.Node) string {
	converter := NewConverter()
	return converter.Convert(node)
}
