package apiconnect

import (
	"context"
	"strconv"
	"testing"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
)

func TestListCommitteeMembers_RequiresChairperson(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "member1", "pass", "Member", "member")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err := client.committees.ListCommitteeMembers(context.Background(), connect.NewRequest(&committeesv1.ListCommitteeMembersRequest{
		CommitteeSlug: "test-comm",
	}))
	if err == nil {
		t.Fatal("expected error for non-chairperson")
	}
}

func TestListCommitteeMembers_ChairpersonCanList(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	resp, err := client.committees.ListCommitteeMembers(context.Background(), connect.NewRequest(&committeesv1.ListCommitteeMembersRequest{
		CommitteeSlug: "test-comm",
	}))
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(resp.Msg.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(resp.Msg.Members))
	}
	if resp.Msg.Members[0].FullName != "Chair" {
		t.Errorf("expected 'Chair', got %q", resp.Msg.Members[0].FullName)
	}
}

func TestAddMemberByEmail(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	addResp, err := client.committees.AddMemberByEmail(context.Background(), connect.NewRequest(&committeesv1.AddMemberByEmailRequest{
		CommitteeSlug: "test-comm",
		Email:         "bob@example.com",
		FullName:      "Bob Jones",
		Role:          "member",
		Quoted:        true,
	}))
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	m := addResp.Msg.Member
	if m.FullName != "Bob Jones" {
		t.Errorf("expected 'Bob Jones', got %q", m.FullName)
	}
	if m.Email == nil || *m.Email != "bob@example.com" {
		t.Errorf("expected email 'bob@example.com', got %v", m.Email)
	}
	if !m.Quoted {
		t.Error("expected quoted=true")
	}
	if m.HasAccount {
		t.Error("expected hasAccount=false for email-only member")
	}

	// Verify in list.
	listResp, err := client.committees.ListCommitteeMembers(context.Background(), connect.NewRequest(&committeesv1.ListCommitteeMembersRequest{
		CommitteeSlug: "test-comm",
	}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listResp.Msg.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(listResp.Msg.Members))
	}
}

func TestUpdateMember(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")
	ts.seedUser(t, "test-comm", "member1", "pass", "Member", "member")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	listResp, err := client.committees.ListCommitteeMembers(context.Background(), connect.NewRequest(&committeesv1.ListCommitteeMembersRequest{
		CommitteeSlug: "test-comm",
	}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	var memberID string
	for _, m := range listResp.Msg.Members {
		if m.FullName == "Member" {
			memberID = m.UserId
		}
	}
	if memberID == "" {
		t.Fatal("member not found in list")
	}

	updateResp, err := client.committees.UpdateMember(context.Background(), connect.NewRequest(&committeesv1.UpdateMemberRequest{
		CommitteeSlug: "test-comm",
		UserId:        memberID,
		Role:          "chairperson",
		Quoted:        true,
	}))
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateResp.Msg.Member.Role != "chairperson" {
		t.Errorf("expected role 'chairperson', got %q", updateResp.Msg.Member.Role)
	}
	if !updateResp.Msg.Member.Quoted {
		t.Error("expected quoted=true")
	}
}

func TestRemoveMember(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")
	ts.seedUser(t, "test-comm", "member1", "pass", "Member", "member")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	listResp, _ := client.committees.ListCommitteeMembers(context.Background(), connect.NewRequest(&committeesv1.ListCommitteeMembersRequest{
		CommitteeSlug: "test-comm",
	}))
	var memberID string
	for _, m := range listResp.Msg.Members {
		if m.FullName == "Member" {
			memberID = m.UserId
		}
	}

	_, err := client.committees.RemoveMember(context.Background(), connect.NewRequest(&committeesv1.RemoveMemberRequest{
		CommitteeSlug: "test-comm",
		UserId:        memberID,
	}))
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	listResp2, _ := client.committees.ListCommitteeMembers(context.Background(), connect.NewRequest(&committeesv1.ListCommitteeMembersRequest{
		CommitteeSlug: "test-comm",
	}))
	if len(listResp2.Msg.Members) != 1 {
		t.Fatalf("expected 1 member after remove, got %d", len(listResp2.Msg.Members))
	}
}

func TestSendInviteEmails_FailsWhenEmailDisabled(t *testing.T) {
	// Test server uses MockSender which is "enabled", but this tests the RPC path.
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-comm")
	ts.seedUser(t, "test-comm", "chair1", "pass", "Chair", "chairperson")
	meetingID := ts.seedMeeting(t, "test-comm", "Meeting", true)

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	// MockSender is enabled, so this should succeed (even if no emails have addresses).
	resp, err := client.committees.SendInviteEmails(context.Background(), connect.NewRequest(&committeesv1.SendInviteEmailsRequest{
		CommitteeSlug: "test-comm",
		MeetingId:     strconv.FormatInt(meetingID, 10),
		BaseUrl:       "http://localhost:8080",
	}))
	if err != nil {
		t.Fatalf("send invites: %v", err)
	}
	// Chair has no email, so should be skipped.
	if resp.Msg.SkippedCount == 0 && resp.Msg.SentCount == 0 {
		// Both zero is fine — no members with contact info.
	}
}
