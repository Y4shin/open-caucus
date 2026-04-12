package apiconnect

import (
	"context"
	"strconv"
	"strings"
	"testing"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
	"github.com/Y4shin/open-caucus/internal/email"
)

func TestSendInviteEmails_EmailMemberReceivesInvite(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair Person", "chairperson")
	meetingID := ts.seedMeeting(t, "test-comm", "Invite Test Meeting", false)

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	// Add an email-only member.
	_, err := client.committees.AddMemberByEmail(context.Background(), connect.NewRequest(&committeesv1.AddMemberByEmailRequest{
		CommitteeSlug: "test-comm",
		Email:         "bob@example.com",
		FullName:      "Bob Jones",
		Role:          "member",
	}))
	if err != nil {
		t.Fatalf("add email member: %v", err)
	}

	// Send invites.
	ts.mockSender.Sent = nil // reset
	resp, err := client.committees.SendInviteEmails(context.Background(), connect.NewRequest(&committeesv1.SendInviteEmailsRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     strconv.FormatInt(meetingID, 10),
		BaseUrl:       "http://localhost:8080",
		Language:      "en",
		Timezone:      "UTC",
	}))
	if err != nil {
		t.Fatalf("send invites: %v", err)
	}

	if resp.Msg.SentCount == 0 {
		t.Fatal("expected at least one email sent")
	}

	// Verify the email-only member received an email.
	found := false
	for _, sent := range ts.mockSender.Sent {
		if sent.To == "bob@example.com" {
			found = true
			if !strings.Contains(sent.HTMLBody, "Invite Test Meeting") {
				t.Errorf("email HTML missing meeting name, got: %s", sent.HTMLBody[:200])
			}
			if !strings.Contains(sent.HTMLBody, "Bob Jones") {
				t.Errorf("email HTML missing member name")
			}
			if !strings.Contains(sent.TextBody, "invite_secret=") {
				t.Errorf("email-only member should get personalized invite_secret link, got: %s", sent.TextBody)
			}
		}
	}
	if !found {
		t.Fatalf("no email sent to bob@example.com; sent %d emails to: %v",
			len(ts.mockSender.Sent), sentRecipients(ts.mockSender.Sent))
	}
}

func TestSendInviteEmails_AccountMemberWithEmailReceivesInvite(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair Person", "chairperson")
	meetingID := ts.seedMeeting(t, "test-comm", "Account Invite Meeting", false)

	// Set email on the chair's account directly.
	if err := ts.repo.UpdateAccountProfile(context.Background(), 1, "Chair Person", "chair@example.com"); err != nil {
		t.Fatalf("set account email: %v", err)
	}

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	ts.mockSender.Sent = nil
	resp, err := client.committees.SendInviteEmails(context.Background(), connect.NewRequest(&committeesv1.SendInviteEmailsRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     strconv.FormatInt(meetingID, 10),
		BaseUrl:       "http://localhost:8080",
		Language:      "en",
	}))
	if err != nil {
		t.Fatalf("send invites: %v", err)
	}

	if resp.Msg.SentCount == 0 {
		t.Fatal("expected email sent to account member with email")
	}

	found := false
	for _, sent := range ts.mockSender.Sent {
		if sent.To == "chair@example.com" {
			found = true
			if !strings.Contains(sent.HTMLBody, "Account Invite Meeting") {
				t.Errorf("email HTML missing meeting name")
			}
			// Account members get a direct meeting link (no invite_secret).
			if strings.Contains(sent.TextBody, "invite_secret=") {
				t.Errorf("account member should NOT get invite_secret link")
			}
		}
	}
	if !found {
		t.Fatalf("no email sent to chair@example.com; sent to: %v", sentRecipients(ts.mockSender.Sent))
	}
}

func TestSendInviteEmails_MemberWithoutEmailSkipped(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair Person", "chairperson")
	ts.seedUser(t, "test-comm", "member1", "pass", "No Email Member", "member")
	meetingID := ts.seedMeeting(t, "test-comm", "Skip Test Meeting", false)

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	ts.mockSender.Sent = nil
	resp, err := client.committees.SendInviteEmails(context.Background(), connect.NewRequest(&committeesv1.SendInviteEmailsRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     strconv.FormatInt(meetingID, 10),
		BaseUrl:       "http://localhost:8080",
	}))
	if err != nil {
		t.Fatalf("send invites: %v", err)
	}

	// Both members have no email set — should all be skipped.
	if resp.Msg.SentCount != 0 {
		t.Errorf("expected 0 sent (no emails configured), got %d", resp.Msg.SentCount)
	}
	if resp.Msg.SkippedCount < 2 {
		t.Errorf("expected at least 2 skipped, got %d", resp.Msg.SkippedCount)
	}
	if len(ts.mockSender.Sent) != 0 {
		t.Errorf("expected no emails sent, got %d", len(ts.mockSender.Sent))
	}
}

func TestSendInviteEmails_IncludesCustomMessageAndLanguage(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair Person", "chairperson")
	meetingID := ts.seedMeeting(t, "test-comm", "Custom Msg Meeting", false)

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	// Add email member.
	client.committees.AddMemberByEmail(context.Background(), connect.NewRequest(&committeesv1.AddMemberByEmailRequest{
		CommitteeSlug: "test-comm",
		Email:         "alice@example.com",
		FullName:      "Alice",
		Role:          "member",
	}))

	ts.mockSender.Sent = nil
	_, err := client.committees.SendInviteEmails(context.Background(), connect.NewRequest(&committeesv1.SendInviteEmailsRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     strconv.FormatInt(meetingID, 10),
		BaseUrl:       "http://localhost:8080",
		CustomMessage: "Please bring your laptop!",
		Language:      "de",
		Timezone:      "Europe/Berlin",
	}))
	if err != nil {
		t.Fatalf("send invites: %v", err)
	}

	if len(ts.mockSender.Sent) == 0 {
		t.Fatal("expected at least one email sent")
	}
	sent := ts.mockSender.Sent[0]

	// Check custom message appears in both HTML and text.
	if !strings.Contains(sent.HTMLBody, "Please bring your laptop!") {
		t.Error("custom message missing from HTML body")
	}
	if !strings.Contains(sent.TextBody, "Please bring your laptop!") {
		t.Error("custom message missing from text body")
	}

	// Check German language strings are used.
	if !strings.Contains(sent.HTMLBody, "Einladung zur Sitzung") {
		t.Error("expected German heading 'Einladung zur Sitzung' in HTML")
	}
}

func sentRecipients(emails []email.MockEmail) []string {
	r := make([]string, len(emails))
	for i, e := range emails {
		r[i] = e.To
	}
	return r
}
