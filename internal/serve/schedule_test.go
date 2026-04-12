package serve

import (
	"testing"
)

func TestResolveSchedule(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"hourly", "0 * * * *"},
		{"daily", "0 6 * * *"},
		{"weekly", "0 6 * * 1"},
		{"monthly", "0 6 1 * *"},
		{"0 20 * * 0", "0 20 * * 0"},
		{"0 0,6,12,18 * * 1-5", "0 0,6,12,18 * * 1-5"},
		{"0 21 * * 0", "0 21 * * 0"},
		{"0 22 * * 0", "0 22 * * 0"},
		{"0 8 * * 1", "0 8 * * 1"},
		{"0 10 * * 1-5", "0 10 * * 1-5"},
	}

	for _, tt := range tests {
		got := resolveSchedule(tt.input)
		if got != tt.want {
			t.Errorf("resolveSchedule(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCronFromTrigger(t *testing.T) {
	tests := []struct {
		trigger string
		want    string
	}{
		{"schedule.hourly", "0 * * * *"},
		{"schedule.daily", "0 6 * * *"},
		{"schedule.weekly", "0 6 * * 1"},
		{"schedule.monthly", "0 6 1 * *"},
		{"pull_request.opened", ""},
		{"workflow_run.conclusion == \"failure\"", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := cronFromTrigger(tt.trigger)
		if got != tt.want {
			t.Errorf("cronFromTrigger(%q) = %q, want %q", tt.trigger, got, tt.want)
		}
	}
}
