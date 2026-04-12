package email

import (
	"strings"
	"testing"
)

func TestRenderMeetingInvite_ContainsAllFields(t *testing.T) {
	html, text, err := RenderMeetingInvite(InviteData{
		MemberName:    "Alice",
		CommitteeName: "Board",
		MeetingName:   "Q1 Review",
		JoinURL:       "https://example.com/join?secret=abc",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{"Alice", "Board", "Q1 Review", "https://example.com/join?secret=abc"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q", want)
		}
	}
}

func TestRenderMeetingInvite_HTMLHasLink(t *testing.T) {
	html, _, err := RenderMeetingInvite(InviteData{
		MemberName:    "Bob",
		CommitteeName: "Committee",
		MeetingName:   "Meeting",
		JoinURL:       "https://example.com/join",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, `href="https://example.com/join"`) {
		t.Error("HTML should contain clickable link")
	}
}

func TestInviteSubject_Format(t *testing.T) {
	s := InviteSubject("Budget Meeting", "Finance Committee")
	if s != "Invite: Budget Meeting — Finance Committee" {
		t.Errorf("unexpected subject: %q", s)
	}
}
