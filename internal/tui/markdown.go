package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	glamouransi "github.com/charmbracelet/glamour/ansi"
	termansi "github.com/charmbracelet/x/ansi"
)

func assistantMarkdownRows(source string, width int) (rows []string) {
	width = max(12, width)
	defer func() {
		if recover() != nil {
			rows = sourceMarkdownRows(source, width)
		}
	}()
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(assistantMarkdownStyle()),
		glamour.WithWordWrap(width),
		glamour.WithTableWrap(true),
	)
	if err != nil {
		return sourceMarkdownRows(source, width)
	}
	rendered, err := renderer.Render(source)
	if err != nil {
		return sourceMarkdownRows(source, width)
	}
	return normalizeMarkdownRows(rendered, width)
}

func assistantMarkdownStyle() glamouransi.StyleConfig {
	return glamouransi.StyleConfig{
		Document: glamouransi.StyleBlock{
			Margin: mdUint(0),
		},
		BlockQuote: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{Color: mdString("245")},
			Indent:         mdUint(1),
			IndentToken:    mdString("| "),
		},
		List: glamouransi.StyleList{
			StyleBlock:  glamouransi.StyleBlock{Margin: mdUint(0)},
			LevelIndent: 2,
		},
		Heading: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{Color: mdString("111"), Bold: mdBool(true)},
			Margin:         mdUint(0),
		},
		H1:     glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{Prefix: "# "}},
		H2:     glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{Prefix: "## "}},
		H3:     glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{Prefix: "### "}},
		H4:     glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{Prefix: "#### "}},
		H5:     glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{Prefix: "##### "}},
		H6:     glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{Prefix: "###### ", Color: mdString("245"), Bold: mdBool(false)}},
		Emph:   glamouransi.StylePrimitive{Italic: mdBool(true)},
		Strong: glamouransi.StylePrimitive{Bold: mdBool(true)},
		HorizontalRule: glamouransi.StylePrimitive{
			Color:  mdString("240"),
			Format: "--------",
		},
		Item:        glamouransi.StylePrimitive{BlockPrefix: "- "},
		Enumeration: glamouransi.StylePrimitive{BlockPrefix: ". "},
		Task:        glamouransi.StyleTask{Ticked: "[x] ", Unticked: "[ ] "},
		Link:        glamouransi.StylePrimitive{Color: mdString("75"), Underline: mdBool(true)},
		LinkText:    glamouransi.StylePrimitive{Color: mdString("111"), Bold: mdBool(true)},
		ImageText:   glamouransi.StylePrimitive{Color: mdString("245"), Format: "Image: {{.text}} ->"},
		Code: glamouransi.StyleBlock{StylePrimitive: glamouransi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           mdString("230"),
			BackgroundColor: mdString("237"),
		}},
		CodeBlock: glamouransi.StyleCodeBlock{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{Color: mdString("252"), BackgroundColor: mdString("236")},
				Margin:         mdUint(0),
			},
			Theme: "github-dark",
		},
		Table: glamouransi.StyleTable{
			StyleBlock:      glamouransi.StyleBlock{Margin: mdUint(0)},
			CenterSeparator: mdString("|"),
			ColumnSeparator: mdString("|"),
			RowSeparator:    mdString("-"),
		},
	}
}

func sourceMarkdownRows(source string, width int) []string {
	rows := wrapText(source, width)
	for i, row := range rows {
		rows[i] = truncateStyled(row, width)
	}
	return rows
}

func normalizeMarkdownRows(rendered string, width int) []string {
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r", "\n")
	rows := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	start := 0
	for start < len(rows) && strings.TrimSpace(termansi.Strip(rows[start])) == "" {
		start++
	}
	end := len(rows)
	for end > start && strings.TrimSpace(termansi.Strip(rows[end-1])) == "" {
		end--
	}
	if start >= end {
		return []string{""}
	}
	rows = rows[start:end]
	for i, row := range rows {
		rows[i] = truncateStyled(row, width)
	}
	return rows
}

func truncateStyled(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.ReplaceAll(value, "\t", "  ")
	value = strings.ReplaceAll(value, "\n", " ")
	if termansi.StringWidth(value) <= width {
		return value
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	return termansi.Truncate(value, width, "...")
}

func styledWidth(value string) int {
	return termansi.StringWidth(value)
}

func mdString(value string) *string {
	return &value
}

func mdBool(value bool) *bool {
	return &value
}

func mdUint(value uint) *uint {
	return &value
}
