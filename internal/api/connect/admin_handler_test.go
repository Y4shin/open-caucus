package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	adminv1 "github.com/Y4shin/conference-tool/gen/go/conference/admin/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
)

// seedAdmin creates an admin account directly in the repo and returns the username.
func (ts *combinedTestServer) seedAdmin(t *testing.T, username, password string) {
	t.Helper()
	hash := hashPassword(t, password)
	if _, err := ts.repo.CreateAccount(context.Background(), username, "Admin User", hash); err != nil {
		t.Fatalf("create admin account %q: %v", username, err)
	}
	account, err := ts.repo.GetAccountByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("get admin account: %v", err)
	}
	if err := ts.repo.SetAccountIsAdmin(context.Background(), account.ID, true); err != nil {
		t.Fatalf("set admin: %v", err)
	}
}

func TestAdminService_GetAdminDashboard(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedAdmin(t, "admin", "adminpass")
	ts.seedCommittee(t, "Committee A", "committee-a")
	ts.seedCommittee(t, "Committee B", "committee-b")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "admin",
		Password: "adminpass",
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	resp, err := client.admin.GetAdminDashboard(context.Background(), connect.NewRequest(&adminv1.GetAdminDashboardRequest{}))
	if err != nil {
		t.Fatalf("get admin dashboard: %v", err)
	}
	if resp.Msg.GetTotalCommittees() < 2 {
		t.Fatalf("expected at least 2 committees, got %d", resp.Msg.GetTotalCommittees())
	}
	if resp.Msg.GetTotalAccounts() < 1 {
		t.Fatalf("expected at least 1 account, got %d", resp.Msg.GetTotalAccounts())
	}
}

func TestAdminService_GetAdminDashboard_NonAdminForbidden(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "member1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err := client.admin.GetAdminDashboard(context.Background(), connect.NewRequest(&adminv1.GetAdminDashboardRequest{}))
	if err == nil {
		t.Fatal("expected permission error for non-admin")
	}
}

func TestAdminService_CreateAndDeleteCommittee(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedAdmin(t, "admin", "adminpass")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "admin",
		Password: "adminpass",
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	createResp, err := client.admin.CreateCommittee(context.Background(), connect.NewRequest(&adminv1.CreateCommitteeRequest{
		Name: "New Committee",
		Slug: "new-committee",
	}))
	if err != nil {
		t.Fatalf("create committee: %v", err)
	}
	if createResp.Msg.GetCommittee().GetSlug() != "new-committee" {
		t.Fatalf("unexpected slug: %q", createResp.Msg.GetCommittee().GetSlug())
	}

	_, err = client.admin.DeleteCommittee(context.Background(), connect.NewRequest(&adminv1.DeleteCommitteeRequest{
		Slug: "new-committee",
	}))
	if err != nil {
		t.Fatalf("delete committee: %v", err)
	}
}

func TestAdminService_CreateCommitteeUser(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedAdmin(t, "admin", "adminpass")
	ts.seedCommittee(t, "Test Committee", "test-committee")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "admin",
		Password: "adminpass",
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	resp, err := client.admin.CreateCommitteeUser(context.Background(), connect.NewRequest(&adminv1.CreateCommitteeUserRequest{
		Slug:     "test-committee",
		Username: "newuser",
		FullName: "New User",
		Password: "newpass123",
		Role:     "member",
		Quoted:   false,
	}))
	if err != nil {
		t.Fatalf("create committee user: %v", err)
	}
	if resp.Msg.GetUser().GetUsername() != "newuser" {
		t.Fatalf("unexpected username: %q", resp.Msg.GetUser().GetUsername())
	}
	if resp.Msg.GetUser().GetRole() != "member" {
		t.Fatalf("unexpected role: %q", resp.Msg.GetUser().GetRole())
	}
}

func TestAdminService_SetAccountAdmin(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedAdmin(t, "admin", "adminpass")
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")

	account, err := ts.repo.GetAccountByUsername(context.Background(), "member1")
	if err != nil {
		t.Fatalf("get account: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "admin",
		Password: "adminpass",
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	resp, err := client.admin.SetAccountAdmin(context.Background(), connect.NewRequest(&adminv1.SetAccountAdminRequest{
		AccountId: fmt.Sprintf("%d", account.ID),
		IsAdmin:   true,
	}))
	if err != nil {
		t.Fatalf("set account admin: %v", err)
	}
	if !resp.Msg.GetAccount().GetIsAdmin() {
		t.Fatal("expected is_admin=true after setting admin")
	}
}
