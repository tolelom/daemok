package b

import (
	"example.com/scan/events"
	_ "example.com/scan/plugins/a"
)

var seen = events.EventDeleted
