package feed

import (
	ev "example.com/scan/events"
	_ "example.com/scan/model"
	_ "example.com/scan/post"
)

var all = ev.EventTypeValues()
