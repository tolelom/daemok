package events

type EventType string

const (
	EventCreated EventType = "created"
	EventDeleted EventType = "deleted"
)

func EventTypeValues() []EventType { return []EventType{EventCreated, EventDeleted} }
