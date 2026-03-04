package statixtoclean

import (
	"strings"
    "fmt"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func StripStatixWrappers(fragment string) (string, []string, error) {
	
    nodes, err := html.ParseFragment(strings.NewReader(fragment), nil)
	if err != nil {
		return "", []string{}, err
	}

	root := &html.Node{
		Type: html.ElementNode,
		Data: "div",
	}
	for _, n := range nodes {
		root.AppendChild(n)
	}

	doc := goquery.NewDocumentFromNode(root)

	doc.Find("button.copy-btn").Remove()

	doc.Find("div.admin-table-scroll-top").Remove()

	doc.Find("div.admin-table-wrapper").Each(func(i int, s *goquery.Selection) {
		s.Children().Unwrap()
	})

	doc.Find("section.form-section").Each(func(i int, s *goquery.Selection) {
		s.Children().Unwrap()
	})

	doc.Find("div.code-block").Each(func(i int, s *goquery.Selection) {
		s.Children().Unwrap()
	})

    doc.Find("input[type='checkbox']").Each(func(i int, s *goquery.Selection) {
    
    	parent := s.Parent()
    
        prefix := "[ ] "
        if _, ok := s.Attr("checked"); ok {
            prefix = "[x] "
        }
        
        // prepends [ ] or [X] to
        // the text inside the input type checkbox 
        parent.PrependHtml(prefix)

        s.Remove()

    })

    doc.Find("table.admin-table").RemoveAttr("class")

    tables := []string{}
    i := 0
    
    doc.Find("table").Each(func(_ int, table *goquery.Selection) {
    
    	md := convertTableToMarkdown(table)
    
    	placeholder := fmt.Sprintf("STATIXTABLETOKEN%d", i)
    	i++
    
    	tables = append(tables, md)
    
    	table.ReplaceWithHtml(placeholder)
    })

	var b strings.Builder
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		html.Render(&b, c)
	}

	return b.String(), tables, nil
}

func convertTableToMarkdown(table *goquery.Selection) string {

	var md strings.Builder

	headers := []string{}

	table.Find("thead tr th").Each(func(_ int, th *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(th.Text()))
	})

	if len(headers) > 0 {
		md.WriteString("| " + strings.Join(headers, " | ") + " |\n")

		sep := make([]string, len(headers))
		for i := range sep {
			sep[i] = "---"
		}

		md.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	}

	table.Find("tbody tr").Each(func(_ int, tr *goquery.Selection) {

		row := []string{}

		tr.Find("td").Each(func(_ int, td *goquery.Selection) {

			html, _ := td.Html()
			row = append(row, strings.TrimSpace(html))

		})

		if len(row) > 0 {
			md.WriteString("| " + strings.Join(row, " | ") + " |\n")
		}
	})

	return md.String()
}


