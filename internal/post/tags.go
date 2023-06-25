package post

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"git.sr.ht/~mariusor/tagextractor"
	vocab "github.com/go-ap/activitypub"
	"gitlab.com/golang-commonmark/markdown"
)

type tags []string

func renderMarkdown(data string) template.HTML {
	md := markdown.New(
		markdown.HTML(true),
		markdown.Tables(true),
		markdown.Linkify(false),
		markdown.Typographer(true),
		markdown.Breaks(true),
	)
	data = md.RenderToString([]byte(data))
	return template.HTML(data)
}
func renderTagHTML(t vocab.Item) template.HTML {
	render := ""

	vocab.OnObject(t, func(ob *vocab.Object) error {
		typ := "tag"
		if ob.Type == vocab.MentionType {
			typ = "mention"
		}
		render = fmt.Sprintf(`<a rel="%s" href="%s">%s</a>`, typ, ob.ID, ob.Name.First().String())
		return nil
	})
	return template.HTML(render)
}

var nl = vocab.DefaultNaturalLanguageValue

func commonTags() vocab.ItemCollection {
	return vocab.ItemCollection{
		vocab.Object{Name: nl(time.Now().Month().String())},
		vocab.Object{Name: nl("metal")},
		vocab.Object{Name: nl("releases")},
	}
}

func renderTagsText(t tags, tagPref string) string {
	for i, g := range t {
		t[i] = tagPref + tagextractor.TagNormalize(g)
	}

	return strings.Join(uniqueValues(t, stringsContain), " ")
}

func (t tags) Render(tagPref string) string {
	for i, g := range t {
		t[i] = tagPref + tagextractor.TagNormalize(g)
	}

	return strings.Join(uniqueValues(t, stringsContain), " ")
}

var ValidMonths = []time.Month{
	time.January, time.February, time.March, time.April, time.May, time.June,
	time.July, time.August, time.September, time.October, time.November, time.December,
}
