package adminservice

import (
	"context"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	adminv1 "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1"
	apierrors "github.com/Y4shin/open-caucus/internal/api/errors"
	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	"github.com/Y4shin/open-caucus/internal/session"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// GetAdminDashboard returns total accounts and committees.
func (s *Service) GetAdminDashboard(ctx context.Context) (*adminv1.GetAdminDashboardResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	totalAccounts, err := s.repo.CountAllAccounts(ctx)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count accounts", err)
	}

	totalCommittees, err := s.repo.CountAllCommittees(ctx)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count committees", err)
	}

	return &adminv1.GetAdminDashboardResponse{
		TotalAccounts:   totalAccounts,
		TotalCommittees: totalCommittees,
	}, nil
}

// ListAccounts returns a paginated list of all accounts.
func (s *Service) ListAccounts(ctx context.Context, page, pageSize int32) (*adminv1.ListAccountsResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	total, err := s.repo.CountAllAccounts(ctx)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count accounts", err)
	}

	limit, offset := pageToLimitOffset(page, pageSize)
	accounts, err := s.repo.ListAllAccounts(ctx, limit, offset)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list accounts", err)
	}

	records := make([]*adminv1.AccountRecord, len(accounts))
	for i, a := range accounts {
		records[i] = toAccountRecord(a)
	}

	return &adminv1.ListAccountsResponse{
		Accounts:   records,
		TotalCount: total,
	}, nil
}

// CreateAccount creates a new global account with local credentials.
func (s *Service) CreateAccount(ctx context.Context, username, fullName, password string) (*adminv1.CreateAccountResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if username == "" || fullName == "" || password == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "username, full_name, and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to hash password", err)
	}

	account, err := s.repo.CreateAccount(ctx, username, fullName, string(hash))
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create account", err)
	}

	return &adminv1.CreateAccountResponse{Account: toAccountRecord(account)}, nil
}

// SetAccountAdmin grants or revokes admin privileges.
func (s *Service) SetAccountAdmin(ctx context.Context, accountIDStr string, isAdmin bool) (*adminv1.SetAccountAdminResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid account id")
	}

	if err := s.repo.SetAccountIsAdmin(ctx, accountID, isAdmin); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update account", err)
	}

	account, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload account", err)
	}

	return &adminv1.SetAccountAdminResponse{Account: toAccountRecord(account)}, nil
}

// ListCommittees returns a paginated list of all committees.
func (s *Service) ListCommittees(ctx context.Context, page, pageSize int32) (*adminv1.ListCommitteesResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	total, err := s.repo.CountAllCommittees(ctx)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count committees", err)
	}

	limit, offset := pageToLimitOffset(page, pageSize)
	committees, err := s.repo.ListAllCommittees(ctx, limit, offset)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list committees", err)
	}

	records := make([]*adminv1.CommitteeRecord, len(committees))
	for i, c := range committees {
		records[i] = toCommitteeRecord(c, 0)
	}

	return &adminv1.ListCommitteesResponse{
		Committees: records,
		TotalCount: total,
	}, nil
}

// CreateCommittee creates a new committee.
func (s *Service) CreateCommittee(ctx context.Context, name, slug string) (*adminv1.CreateCommitteeResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if name == "" || slug == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "name and slug are required")
	}

	if err := s.repo.CreateCommitteeWithSlug(ctx, name, slug); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create committee", err)
	}

	committee, err := s.repo.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload committee", err)
	}

	return &adminv1.CreateCommitteeResponse{Committee: toCommitteeRecord(committee, 0)}, nil
}

// DeleteCommittee deletes a committee by slug.
func (s *Service) DeleteCommittee(ctx context.Context, slug string) (*adminv1.DeleteCommitteeResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if slug == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "slug is required")
	}

	if err := s.repo.DeleteCommitteeBySlug(ctx, slug); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete committee", err)
	}

	return &adminv1.DeleteCommitteeResponse{}, nil
}

// GetCommitteeAdmin returns committee details, users, and OAuth rules.
func (s *Service) GetCommitteeAdmin(ctx context.Context, slug string) (*adminv1.GetCommitteeAdminResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	committee, err := s.repo.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	total, err := s.repo.CountUsersInCommittee(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count users", err)
	}

	users, err := s.repo.ListUsersInCommittee(ctx, slug, int(total), 0)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list users", err)
	}

	rules, err := s.repo.ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list oauth rules", err)
	}

	userRecords := make([]*adminv1.CommitteeUserRecord, len(users))
	for i, u := range users {
		oauthManaged, _ := s.repo.IsOAuthManagedMembership(ctx, u.ID)
		userRecords[i] = toCommitteeUserRecord(u, oauthManaged)
	}

	ruleRecords := make([]*adminv1.OAuthRuleRecord, len(rules))
	for i, r := range rules {
		ruleRecords[i] = toOAuthRuleRecord(r)
	}

	return &adminv1.GetCommitteeAdminResponse{
		Committee:  toCommitteeRecord(committee, total),
		Users:      userRecords,
		OauthRules: ruleRecords,
	}, nil
}

// ListCommitteeUsers returns paginated members of a committee.
func (s *Service) ListCommitteeUsers(ctx context.Context, slug string, page, pageSize int32) (*adminv1.ListCommitteeUsersResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	total, err := s.repo.CountUsersInCommittee(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to count users", err)
	}

	limit, offset := pageToLimitOffset(page, pageSize)
	users, err := s.repo.ListUsersInCommittee(ctx, slug, limit, offset)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list users", err)
	}

	records := make([]*adminv1.CommitteeUserRecord, len(users))
	for i, u := range users {
		oauthManaged, _ := s.repo.IsOAuthManagedMembership(ctx, u.ID)
		records[i] = toCommitteeUserRecord(u, oauthManaged)
	}

	return &adminv1.ListCommitteeUsersResponse{
		Users:      records,
		TotalCount: total,
	}, nil
}

// CreateCommitteeUser creates a new local user account for a committee.
func (s *Service) CreateCommitteeUser(ctx context.Context, slug, username, fullName, password, role string, quoted bool) (*adminv1.CreateCommitteeUserResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if username == "" || fullName == "" || password == "" || role == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "username, full_name, password, and role are required")
	}

	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to hash password", err)
	}

	if err := s.repo.CreateUser(ctx, committeeID, username, string(hash), fullName, quoted, role); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create user", err)
	}

	user, err := s.repo.GetUserByCommitteeAndUsername(ctx, slug, username)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload user", err)
	}

	return &adminv1.CreateCommitteeUserResponse{User: toCommitteeUserRecord(user, false)}, nil
}

// DeleteCommitteeUser removes a user from a committee.
func (s *Service) DeleteCommitteeUser(ctx context.Context, slug, userIDStr string) (*adminv1.DeleteCommitteeUserResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid user id")
	}

	if err := s.repo.DeleteUserByID(ctx, userID); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete user", err)
	}

	return &adminv1.DeleteCommitteeUserResponse{}, nil
}

// AssignAccountToCommittee assigns an existing account to a committee.
func (s *Service) AssignAccountToCommittee(ctx context.Context, slug, accountIDStr, role string, quoted bool) (*adminv1.AssignAccountToCommitteeResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if slug == "" || accountIDStr == "" || role == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "slug, account_id, and role are required")
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid account id")
	}

	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		return nil, apierrors.New(apierrors.KindNotFound, "committee not found")
	}

	if err := s.repo.AssignAccountToCommittee(ctx, committeeID, accountID, quoted, role); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to assign account", err)
	}

	account, err := s.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload account", err)
	}

	user, err := s.repo.GetUserByCommitteeAndUsername(ctx, slug, account.Username)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload user", err)
	}

	oauthManaged, _ := s.repo.IsOAuthManagedMembership(ctx, user.ID)
	return &adminv1.AssignAccountToCommitteeResponse{User: toCommitteeUserRecord(user, oauthManaged)}, nil
}

// UpdateCommitteeUser updates role and quotation flag for a committee member.
func (s *Service) UpdateCommitteeUser(ctx context.Context, slug, userIDStr, role string, quoted bool) (*adminv1.UpdateCommitteeUserResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid user id")
	}

	if err := s.repo.UpdateUserMembership(ctx, userID, quoted, role); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to update user", err)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to reload user", err)
	}

	oauthManaged, _ := s.repo.IsOAuthManagedMembership(ctx, userID)
	return &adminv1.UpdateCommitteeUserResponse{User: toCommitteeUserRecord(user, oauthManaged)}, nil
}

// ListOAuthRules returns OAuth rules for a committee.
func (s *Service) ListOAuthRules(ctx context.Context, slug string) (*adminv1.ListOAuthRulesResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	rules, err := s.repo.ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx, slug)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to list oauth rules", err)
	}

	records := make([]*adminv1.OAuthRuleRecord, len(rules))
	for i, r := range rules {
		records[i] = toOAuthRuleRecord(r)
	}

	return &adminv1.ListOAuthRulesResponse{Rules: records}, nil
}

// CreateOAuthRule creates a new OAuth group-to-role rule for a committee.
func (s *Service) CreateOAuthRule(ctx context.Context, slug, groupName, role string) (*adminv1.CreateOAuthRuleResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if groupName == "" || role == "" {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "group_name and role are required")
	}

	rule, err := s.repo.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, slug, groupName, role)
	if err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to create oauth rule", err)
	}

	return &adminv1.CreateOAuthRuleResponse{Rule: toOAuthRuleRecord(rule)}, nil
}

// DeleteOAuthRule removes an OAuth rule.
func (s *Service) DeleteOAuthRule(ctx context.Context, slug, ruleIDStr string) (*adminv1.DeleteOAuthRuleResponse, error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	ruleID, err := strconv.ParseInt(ruleIDStr, 10, 64)
	if err != nil {
		return nil, apierrors.New(apierrors.KindInvalidArgument, "invalid rule id")
	}

	if err := s.repo.DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug(ctx, ruleID, slug); err != nil {
		return nil, apierrors.Wrap(apierrors.KindInternal, "failed to delete oauth rule", err)
	}

	return &adminv1.DeleteOAuthRuleResponse{}, nil
}

func (s *Service) requireAdmin(ctx context.Context) error {
	sd, ok := session.GetSession(ctx)
	if !ok || sd == nil || sd.IsExpired() || !sd.IsAccountSession() || sd.AccountID == nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account session required")
	}
	account, err := s.repo.GetAccountByID(ctx, *sd.AccountID)
	if err != nil {
		return apierrors.New(apierrors.KindUnauthenticated, "account not found")
	}
	if !account.IsAdmin {
		return apierrors.New(apierrors.KindPermissionDenied, "admin privileges required")
	}
	return nil
}

func toAccountRecord(a *model.Account) *adminv1.AccountRecord {
	return &adminv1.AccountRecord{
		AccountId: strconv.FormatInt(a.ID, 10),
		Username:  a.Username,
		FullName:  a.FullName,
		IsAdmin:   a.IsAdmin,
	}
}

func toCommitteeRecord(c *model.Committee, memberCount int64) *adminv1.CommitteeRecord {
	return &adminv1.CommitteeRecord{
		CommitteeId: strconv.FormatInt(c.ID, 10),
		Slug:        c.Slug,
		Name:        c.Name,
		MemberCount: memberCount,
	}
}

func toCommitteeUserRecord(u *model.User, oauthManaged bool) *adminv1.CommitteeUserRecord {
	return &adminv1.CommitteeUserRecord{
		UserId:         strconv.FormatInt(u.ID, 10),
		Username:       u.Username,
		FullName:       u.FullName,
		Role:           u.Role,
		Quoted:         u.Quoted,
		IsOauthManaged: oauthManaged,
	}
}

func toOAuthRuleRecord(r *model.OAuthCommitteeGroupRule) *adminv1.OAuthRuleRecord {
	return &adminv1.OAuthRuleRecord{
		RuleId:    strconv.FormatInt(r.ID, 10),
		GroupName: r.GroupName,
		Role:      r.Role,
	}
}

// pageToLimitOffset converts 1-based page number and page size to limit/offset.
// page <= 0 defaults to page 1; pageSize <= 0 defaults to 25.
func pageToLimitOffset(page, pageSize int32) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	return int(pageSize), int((page - 1) * pageSize)
}
