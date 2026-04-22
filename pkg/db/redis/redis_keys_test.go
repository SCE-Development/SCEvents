package db

import "testing"

func TestEventHeadcountKey(t *testing.T) {
	got := EventHeadcountKey("event-123")
	want := "event:event-123:headcount"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEventRegistrantsKey(t *testing.T) {
	got := EventRegistrantsKey("event-123")
	want := "event:event-123:registrants"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
