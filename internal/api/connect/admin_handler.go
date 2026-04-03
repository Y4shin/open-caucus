package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	adminv1 "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1"
	adminv1connect "github.com/Y4shin/open-caucus/gen/go/conference/admin/v1/adminv1connect"
	adminservice "github.com/Y4shin/open-caucus/internal/services/admin"
)

type AdminHandler struct {
	adminv1connect.UnimplementedAdminServiceHandler
	service *adminservice.Service
}

func NewAdminHandler(service *adminservice.Service) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) GetAdminDashboard(ctx context.Context, req *connect.Request[adminv1.GetAdminDashboardRequest]) (*connect.Response[adminv1.GetAdminDashboardResponse], error) {
	resp, err := h.service.GetAdminDashboard(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) ListAccounts(ctx context.Context, req *connect.Request[adminv1.ListAccountsRequest]) (*connect.Response[adminv1.ListAccountsResponse], error) {
	resp, err := h.service.ListAccounts(ctx, req.Msg.Page, req.Msg.PageSize)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) CreateAccount(ctx context.Context, req *connect.Request[adminv1.CreateAccountRequest]) (*connect.Response[adminv1.CreateAccountResponse], error) {
	resp, err := h.service.CreateAccount(ctx, req.Msg.Username, req.Msg.FullName, req.Msg.Password)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) SetAccountAdmin(ctx context.Context, req *connect.Request[adminv1.SetAccountAdminRequest]) (*connect.Response[adminv1.SetAccountAdminResponse], error) {
	resp, err := h.service.SetAccountAdmin(ctx, req.Msg.AccountId, req.Msg.IsAdmin)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) ListCommittees(ctx context.Context, req *connect.Request[adminv1.ListCommitteesRequest]) (*connect.Response[adminv1.ListCommitteesResponse], error) {
	resp, err := h.service.ListCommittees(ctx, req.Msg.Page, req.Msg.PageSize)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) CreateCommittee(ctx context.Context, req *connect.Request[adminv1.CreateCommitteeRequest]) (*connect.Response[adminv1.CreateCommitteeResponse], error) {
	resp, err := h.service.CreateCommittee(ctx, req.Msg.Name, req.Msg.Slug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) DeleteCommittee(ctx context.Context, req *connect.Request[adminv1.DeleteCommitteeRequest]) (*connect.Response[adminv1.DeleteCommitteeResponse], error) {
	resp, err := h.service.DeleteCommittee(ctx, req.Msg.Slug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) GetCommitteeAdmin(ctx context.Context, req *connect.Request[adminv1.GetCommitteeAdminRequest]) (*connect.Response[adminv1.GetCommitteeAdminResponse], error) {
	resp, err := h.service.GetCommitteeAdmin(ctx, req.Msg.Slug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) ListCommitteeUsers(ctx context.Context, req *connect.Request[adminv1.ListCommitteeUsersRequest]) (*connect.Response[adminv1.ListCommitteeUsersResponse], error) {
	resp, err := h.service.ListCommitteeUsers(ctx, req.Msg.Slug, req.Msg.Page, req.Msg.PageSize)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) CreateCommitteeUser(ctx context.Context, req *connect.Request[adminv1.CreateCommitteeUserRequest]) (*connect.Response[adminv1.CreateCommitteeUserResponse], error) {
	resp, err := h.service.CreateCommitteeUser(ctx, req.Msg.Slug, req.Msg.Username, req.Msg.FullName, req.Msg.Password, req.Msg.Role, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) DeleteCommitteeUser(ctx context.Context, req *connect.Request[adminv1.DeleteCommitteeUserRequest]) (*connect.Response[adminv1.DeleteCommitteeUserResponse], error) {
	resp, err := h.service.DeleteCommitteeUser(ctx, req.Msg.Slug, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) AssignAccountToCommittee(ctx context.Context, req *connect.Request[adminv1.AssignAccountToCommitteeRequest]) (*connect.Response[adminv1.AssignAccountToCommitteeResponse], error) {
	resp, err := h.service.AssignAccountToCommittee(ctx, req.Msg.Slug, req.Msg.AccountId, req.Msg.Role, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) UpdateCommitteeUser(ctx context.Context, req *connect.Request[adminv1.UpdateCommitteeUserRequest]) (*connect.Response[adminv1.UpdateCommitteeUserResponse], error) {
	resp, err := h.service.UpdateCommitteeUser(ctx, req.Msg.Slug, req.Msg.UserId, req.Msg.Role, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) ListOAuthRules(ctx context.Context, req *connect.Request[adminv1.ListOAuthRulesRequest]) (*connect.Response[adminv1.ListOAuthRulesResponse], error) {
	resp, err := h.service.ListOAuthRules(ctx, req.Msg.Slug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) CreateOAuthRule(ctx context.Context, req *connect.Request[adminv1.CreateOAuthRuleRequest]) (*connect.Response[adminv1.CreateOAuthRuleResponse], error) {
	resp, err := h.service.CreateOAuthRule(ctx, req.Msg.Slug, req.Msg.GroupName, req.Msg.Role)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *AdminHandler) DeleteOAuthRule(ctx context.Context, req *connect.Request[adminv1.DeleteOAuthRuleRequest]) (*connect.Response[adminv1.DeleteOAuthRuleResponse], error) {
	resp, err := h.service.DeleteOAuthRule(ctx, req.Msg.Slug, req.Msg.RuleId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
