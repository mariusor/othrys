package othrys

import (
	vocab "github.com/go-ap/activitypub"
	"regexp"
	"strings"
	"time"
)

func SetIDOf(it vocab.Item, id vocab.ID) {
	if vocab.LinkTypes.Contains(it.GetType()) {
		vocab.OnLink(it, func(lnk *vocab.Link) error {
			lnk.ID = id
			return nil
		})
	} else {
		vocab.OnObject(it, func(ob *vocab.Object) error {
			ob.ID = id
			return nil
		})
	}
}

func NameOf(it vocab.Item) string {
	var name string
	if vocab.LinkTypes.Contains(it.GetType()) {
		vocab.OnLink(it, func(lnk *vocab.Link) error {
			name = lnk.Name.First().String()
			return nil
		})
	} else {
		vocab.OnObject(it, func(ob *vocab.Object) error {
			name = ob.Name.First().String()
			return nil
		})
	}
	return name
}

var NL = vocab.DefaultNaturalLanguageValue

func TagNormalize(t string) string {
	hasHash := len(t) > 2 && t[0] == '#'
	if hasHash {
		t = t[1:]
	}
	if strings.EqualFold(t, "Post-Metal") {
		return "postmetal"
	}
	if strings.EqualFold(t, "Metal") {
		return "metal"
	}
	t = strings.ToLower(t)
	t = removeStrings(t, toRemoveStrings...)
	t = repl.ReplaceAllLiteralString(t, "")
	return t
}

var (
	repl            = regexp.MustCompile("metal$")
	toRemoveStrings = []string{"(early)", "(later)", "(mid)", "-", " ", "#", "'"}
)

func removeStrings(s string, replace ...string) string {
	for _, r := range replace {
		s = strings.ReplaceAll(s, r, "")
	}
	return s
}

func WrapObjectInCreate(actor vocab.Actor, p vocab.Item) vocab.Activity {
	now := time.Now().UTC()
	return vocab.Activity{
		Type:         vocab.CreateType,
		Published:    now,
		Updated:      now,
		AttributedTo: actor.GetLink(),
		Actor:        actor.GetLink(),
		Object:       p,
	}
}
