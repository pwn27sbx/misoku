package tutor

// Event is one action the app reports to the tutorial (F3b/F3c). Data holds
// the event's named values; an action check matches Name and, when it sets
// one, the "value" (E9).
type Event struct {
	Name string
	Data map[string]string
}

// Value is the event's "value", "" when it has none.
func (e Event) Value() string { return e.Data["value"] }
