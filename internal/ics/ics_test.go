package ics

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateMeetingEvent_Basic(t *testing.T) {
	start := time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 15, 16, 0, 0, 0, time.UTC)

	ical := GenerateMeetingEvent("uid-123", "Budget Meeting", "Quarterly review", start, end)

	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Open Caucus//EN",
		"BEGIN:VEVENT",
		"UID:uid-123",
		"DTSTART:20260515T140000Z",
		"DTEND:20260515T160000Z",
		"SUMMARY:Budget Meeting",
		"DESCRIPTION:Quarterly review",
		"END:VEVENT",
		"END:VCALENDAR",
	} {
		if !strings.Contains(ical, want) {
			t.Errorf("expected %q in output, got:\n%s", want, ical)
		}
	}
}

func TestGenerateMeetingEvent_DefaultDuration(t *testing.T) {
	start := time.Date(2026, 5, 15, 14, 0, 0, 0, time.UTC)

	ical := GenerateMeetingEvent("uid-456", "Open Meeting", "", start, time.Time{})

	if !strings.Contains(ical, "DTEND:20260515T160000Z") {
		t.Errorf("expected 2-hour default end time, got:\n%s", ical)
	}
}

func TestGenerateMeetingEvent_ConvertsToUTC(t *testing.T) {
	loc := time.FixedZone("CET", 1*60*60)
	start := time.Date(2026, 5, 15, 15, 0, 0, 0, loc) // 15:00 CET = 14:00 UTC
	end := time.Date(2026, 5, 15, 17, 0, 0, 0, loc)

	ical := GenerateMeetingEvent("uid-tz", "TZ Test", "", start, end)

	if !strings.Contains(ical, "DTSTART:20260515T140000Z") {
		t.Errorf("expected UTC start time, got:\n%s", ical)
	}
}

func TestGenerateMeetingEvent_EscapesSpecialChars(t *testing.T) {
	ical := GenerateMeetingEvent("uid-esc", "Meeting; important", "Line1\nLine2, more", time.Now(), time.Time{})

	if !strings.Contains(ical, "SUMMARY:Meeting\\; important") {
		t.Errorf("expected escaped semicolon in summary, got:\n%s", ical)
	}
	if !strings.Contains(ical, "DESCRIPTION:Line1\\nLine2\\, more") {
		t.Errorf("expected escaped description, got:\n%s", ical)
	}
}
