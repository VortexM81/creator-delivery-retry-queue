package main

import "testing"

func TestNextActionHoldsAfterThreeAttempts(t *testing.T) {
	tests := []struct {
		attempt int
		want    string
	}{{1, "deliver"}, {2, "deliver"}, {3, "hold"}}
	for _, tt := range tests {
		if got := nextAction(DeliveryEvent{Attempt: tt.attempt}); got != tt.want {
			t.Errorf("attempt %d: got %q, want %q", tt.attempt, got, tt.want)
		}
	}
}
