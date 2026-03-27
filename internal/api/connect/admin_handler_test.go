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

func TestAdminService_CreateAccount(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedAdmin(t, "admin", "adminpass")

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "admin",
		Password: "adminpass",
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	resp, err := client.admin.CreateAccount(context.Background(), connect.NewRequest(&adminv1.CreateAccountRequest{
		Username: "newaccount",
		FullName: "New Account",
		Password: "password123",
	}))
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if got := resp.Msg.GetAccount().GetUsername(); got != "newaccount" {
		t.Fatalf("unexpected created username: got %q want %q", got, "newaccount")
	}
}

func TestAdminService_AssignAccountToCommittee_AndUpdateMembership(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedAdmin(t, "admin", "adminpass")
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "member1", "pass123", "Alice Member", "member")

	account, err := ts.repo.GetAccountByUsername(context.Background(), "member1")
	if err != nil {
		t.Fatalf("get account: %v", err)
	}

	user, err := ts.repo.GetUserByCommitteeAndUsername(context.Background(), "test-committee", "member1")
	if err != nil {
		t.Fatalf("get seeded membership: %v", err)
	}
	if err := ts.repo.DeleteUserByID(context.Background(), user.ID); err != nil {
		t.Fatalf("delete seeded membership: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "admin",
		Password: "adminpass",
	})); err != nil {
		t.Fatalf("admin login: %v", err)
	}

	assignResp, err := client.admin.AssignAccountToCommittee(context.Background(), connect.NewRequest(&adminv1.AssignAccountToCommitteeRequest{
		Slug:      "test-committee",
		AccountId: fmt.Sprintf("%d", account.ID),
		Role:      "chairperson",
		Quoted:    true,
	}))
	if err != nil {
		t.Fatalf("assign account to committee: %v", err)
	}
	if assignResp.Msg.GetUser().GetUsername() != "member1" {
		t.Fatalf("unexpected assigned username: %q", assignResp.Msg.GetUser().GetUsername())
	}
	if assignResp.Msg.GetUser().GetRole() != "chairperson" {
		t.Fatalf("unexpected assigned role: %q", assignResp.Msg.GetUser().GetRole())
	}
	if !assignResp.Msg.GetUser().GetQuoted() {
		t.Fatal("expected assigned membership to be quoted")
	}

	adminResp, err := client.admin.GetCommitteeAdmin(context.Background(), connect.NewRequest(&adminv1.GetCommitteeAdminRequest{
		Slug: "test-committee",
	}))
	if err != nil {
		t.Fatalf("get committee admin: %v", err)
	}
	if len(adminResp.Msg.GetUsers()) != 1 {
		t.Fatalf("expected 1 committee user after assignment, got %d", len(adminResp.Msg.GetUsers()))
	}

	userID := adminResp.Msg.GetUsers()[0].GetUserId()
	updateResp, err := client.admin.UpdateCommitteeUser(context.Background(), connect.NewRequest(&adminv1.UpdateCommitteeUserRequest{
		Slug:   "test-committee",
		UserId: userID,
		Role:   "member",
		Quoted: false,
	}))
	if err != nil {
		t.Fatalf("update committee user: %v", err)
	}
	if updateResp.Msg.GetUser().GetRole() != "member" {
		t.Fatalf("unexpected updated role: %q", updateResp.Msg.GetUser().GetRole())
	}
	if updateResp.Msg.GetUser().GetQuoted() {
		t.Fatal("expected quoted=false after update")
	}
}

func TestAdminService_CreateListAndDeleteOAuthRule(t *testing.T) {
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

	createResp, err := client.admin.CreateOAuthRule(context.Background(), connect.NewRequest(&adminv1.CreateOAuthRuleRequest{
		Slug:      "test-committee",
		GroupName: "committee-test-chairs",
		Role:      "chairperson",
	}))
	if err != nil {
		t.Fatalf("create oauth rule: %v", err)
	}
	if createResp.Msg.GetRule().GetGroupName() != "committee-test-chairs" {
		t.Fatalf("unexpected group name: %q", createResp.Msg.GetRule().GetGroupName())
	}

	listResp, err := client.admin.ListOAuthRules(context.Background(), connect.NewRequest(&adminv1.ListOAuthRulesRequest{
		Slug: "test-committee",
	}))
	if err != nil {
		t.Fatalf("list oauth rules: %v", err)
	}
	if len(listResp.Msg.GetRules()) != 1 {
		t.Fatalf("expected 1 oauth rule, got %d", len(listResp.Msg.GetRules()))
	}

	if _, err := client.admin.DeleteOAuthRule(context.Background(), connect.NewRequest(&adminv1.DeleteOAuthRuleRequest{
		Slug:   "test-committee",
		RuleId: createResp.Msg.GetRule().GetRuleId(),
	})); err != nil {
		t.Fatalf("delete oauth rule: %v", err)
	}

	listResp, err = client.admin.ListOAuthRules(context.Background(), connect.NewRequest(&adminv1.ListOAuthRulesRequest{
		Slug: "test-committee",
	}))
	if err != nil {
		t.Fatalf("list oauth rules after delete: %v", err)
	}
	if len(listResp.Msg.GetRules()) != 0 {
		t.Fatalf("expected 0 oauth rules after delete, got %d", len(listResp.Msg.GetRules()))
	}
}
