package post

import (
	"example.com/scan/events"
	_ "example.com/scan/model"
	_ "example.com/scan/post/util"
)

var published = []events.EventType{events.EventCreated, events.EventDeleted}
