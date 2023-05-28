package othrys

import (
	vocab "github.com/go-ap/activitypub"
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
