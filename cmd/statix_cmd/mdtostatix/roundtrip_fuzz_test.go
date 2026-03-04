package mdtostatix

import (
	"testing"
    "strings"
    "fmt"
    "unicode/utf8"

	htmlmd "github.com/JohannesKaufmann/html-to-markdown"

	"blog/cmd/statix_cmd/statixtoclean"
)

func FuzzMarkdownRoundTrip(f *testing.F) {

    tokens := []string{
    	"#", "##", "###",
    	"*", "**", "_", "__",
    	"`", "```",
    	"- ", "* ",
    	"[x]", "[ ]",
    	"|", "---",
    	"(", ")", "[", "]",
    }
    
    for _, t := range tokens {
    	f.Add(t)
    }

	//f.Add("# hello")
	//f.Add("**bold** *italic*")
	//f.Add("`code`")
	//f.Add("- item\n- item2")
	//f.Add("| a | b |\n|---|---|\n| 1 | 2 |")

	converter := htmlmd.NewConverter("", true, nil)

	f.Fuzz(func(t *testing.T, input string) {

		if !utf8.ValidString(input) {
			t.Skip()
		}

		html1, err := MarkdownToStatixHTML(input)
		if err != nil {
			t.Skip()
		}

		cleanHTML, tables, err := statixtoclean.StripStatixWrappers(html1)
		if err != nil {
			t.Skip()
		}

		md2, err := converter.ConvertString(cleanHTML)
		if err != nil {
			t.Skip()
		}

		for i, tbl := range tables {
			token := fmt.Sprintf("STATIXTABLETOKEN%d", i)
			md2 = strings.ReplaceAll(md2, token, tbl)
		}

		// Ensure roundtrip Markdown is still valid
		_, err = MarkdownToStatixHTML(md2)
		if err != nil {
			t.Fatalf(
				"roundtrip render failed\n\nINPUT:\n%s\n\nROUNDTRIP:\n%s\n",
				input,
				md2,
			)
		}
	})
}


