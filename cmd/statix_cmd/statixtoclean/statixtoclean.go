package statixtoclean

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func StripStatixWrappers(fragment string) (string, error) {
	// Parse fragment as nodes (not full document)
	nodes, err := html.ParseFragment(strings.NewReader(fragment), nil)
	if err != nil {
		return "", err
	}

	// Create a fake root container
	root := &html.Node{
		Type: html.ElementNode,
		Data: "div",
	}
	for _, n := range nodes {
		root.AppendChild(n)
	}

	// Create selection from root
	doc := goquery.NewDocumentFromNode(root)

	// --- Transformations ---

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
    
    	parent.PrependHtml(prefix)
    
    	s.Remove()
    })

	// --- Serialize children of fake root ---
	var b strings.Builder
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		html.Render(&b, c)
	}

	return b.String(), nil
}


