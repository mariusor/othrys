package post

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"

	othrys "github.com/mariusor/esports-calendar"
)

type tags []string

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
		t[i] = tagPref + othrys.TagNormalize(g)
	}

	return strings.Join(uniqueValues(t, stringsContain), " ")
}

func (t tags) Render(tagPref string) string {
	for i, g := range t {
		t[i] = tagPref + othrys.TagNormalize(g)
	}

	return strings.Join(uniqueValues(t, stringsContain), " ")
}

var ValidMonths = []time.Month{
	time.January, time.February, time.March, time.April, time.May, time.June,
	time.July, time.August, time.September, time.October, time.November, time.December,
}
