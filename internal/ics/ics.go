// Package ics generates iCalendar (RFC 5545) events for meeting invites.
package ics

import (
	"fmt"
	"strings"
	"time"
)

// GenerateMeetingEvent returns a VCALENDAR string containing a single VEVENT.
// All times are emitted in UTC. If end is zero, a default 2-hour duration is used.
func GenerateMeetingEvent(uid, summary, description string, start, end time.Time) string {
	start = start.UTC()
	if end.IsZero() {
		end = start.Add(2 * time.Hour)
	}
	end = end.UTC()

	var b strings.Builder
	fmt.Fprint(&b, "BEGIN:VCALENDAR\r\n")
	fmt.Fprint(&b, "VERSION:2.0\r\n")
	fmt.Fprint(&b, "PRODID:-//Open Caucus//EN\r\n")
	fmt.Fprint(&b, "BEGIN:VEVENT\r\n")
	fmt.Fprintf(&b, "UID:%s\r\n", uid)
	fmt.Fprintf(&b, "DTSTART:%s\r\n", start.Format("20060102T150405Z"))
	fmt.Fprintf(&b, "DTEND:%s\r\n", end.Format("20060102T150405Z"))
	fmt.Fprintf(&b, "SUMMARY:%s\r\n", escapeICS(summary))
	fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", escapeICS(description))
	fmt.Fprint(&b, "END:VEVENT\r\n")
	fmt.Fprint(&b, "END:VCALENDAR\r\n")
	return b.String()
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
