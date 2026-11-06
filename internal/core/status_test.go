package core

import "testing"

func TestParseStatus(t *testing.T) {
	tests := []struct {
		in   string
		want NoteStatus
	}{
		{"active", StatusActive},
		{"Active", StatusActive},
		{" ACTIVE ", StatusActive},
		{"onhold", StatusOnHold},
		{"on hold", StatusOnHold},
		{"On-Hold", StatusOnHold},
		{"completed", StatusCompleted},
		{"Completed", StatusCompleted},
		{"dropped", StatusDropped},
		{"DROPPED", StatusDropped},
		{"", StatusNone},
		{"done", StatusNone},
		{"on  hold", StatusNone},
		{"activ", StatusNone},
	}
	for _, tt := range tests {
		if got := ParseStatus(tt.in); got != tt.want {
			t.Errorf("ParseStatus(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
