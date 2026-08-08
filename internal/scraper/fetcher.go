package scraper

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func SelectHTML(html, selector string) ([]Selection, error) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	selections := []Selection{}
	document.Find(selector).Each(func(_ int, element *goquery.Selection) {
		attrs := map[string]string{}
		for _, attribute := range element.Nodes[0].Attr {
			attrs[attribute.Key] = attribute.Val
		}
		inner, _ := element.Html()
		selections = append(selections, Selection{Text: element.Text(), HTML: inner, Attrs: attrs})
	})
	return selections, nil
}

type Selection struct {
	Text, HTML string
	Attrs      map[string]string
}
