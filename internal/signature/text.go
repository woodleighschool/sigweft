package signature

import (
	"strings"

	"golang.org/x/net/html"
)

func plainText(source string) string {
	tokens := html.NewTokenizer(strings.NewReader(source))
	var out strings.Builder
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			var paragraphs []string
			for line := range strings.SplitSeq(out.String(), "\x00") {
				if line = strings.Join(strings.Fields(line), " "); line != "" {
					paragraphs = append(paragraphs, line)
				}
			}
			return strings.Join(paragraphs, "\n")
		case html.TextToken:
			out.WriteString(strings.ReplaceAll(string(tokens.Text()), "\u00a0", " "))
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			tag, _ := tokens.TagName()
			switch string(tag) {
			case "br", "p", "div", "tr":
				out.WriteByte(0)
			case "td", "th":
				out.WriteByte(' ')
			}
		case html.CommentToken, html.DoctypeToken:
		}
	}
}
