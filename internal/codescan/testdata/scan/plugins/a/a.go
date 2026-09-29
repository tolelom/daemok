package a

import (
	"example.com/scan/events"
	_ "example.com/scan/plugins/b"
)

var on = events.EventCreated
