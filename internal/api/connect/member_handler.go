package apiconnect

import (
	"context"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	memberservice "github.com/Y4shin/open-caucus/internal/services/members"
)

// MemberHandler implements the member management RPCs on CommitteeService.
type MemberHandler struct {
	service *memberservice.Service
}

func NewMemberHandler(service *memberservice.Service) *MemberHandler {
	return &MemberHandler{service: service}
}

func (h *MemberHandler) ListCommitteeMembers(ctx context.Context, req *connect.Request[committeesv1.ListCommitteeMembersRequest]) (*connect.Response[committeesv1.ListCommitteeMembersResponse], error) {
	resp, err := h.service.ListCommitteeMembers(ctx, req.Msg.CommitteeSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MemberHandler) ListAssignableAccounts(ctx context.Context, req *connect.Request[committeesv1.ListAssignableAccountsRequest]) (*connect.Response[committeesv1.ListAssignableAccountsResponse], error) {
	resp, err := h.service.ListAssignableAccounts(ctx, req.Msg.CommitteeSlug)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MemberHandler) AddMemberByEmail(ctx context.Context, req *connect.Request[committeesv1.AddMemberByEmailRequest]) (*connect.Response[committeesv1.AddMemberByEmailResponse], error) {
	resp, err := h.service.AddMemberByEmail(ctx, req.Msg.CommitteeSlug, req.Msg.Email, req.Msg.FullName, req.Msg.Role, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MemberHandler) AssignAccountToCommittee(ctx context.Context, req *connect.Request[committeesv1.CommitteeAssignAccountRequest]) (*connect.Response[committeesv1.CommitteeAssignAccountResponse], error) {
	resp, err := h.service.AssignAccountToCommittee(ctx, req.Msg.CommitteeSlug, req.Msg.AccountId, req.Msg.Role, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MemberHandler) UpdateMember(ctx context.Context, req *connect.Request[committeesv1.UpdateMemberRequest]) (*connect.Response[committeesv1.UpdateMemberResponse], error) {
	resp, err := h.service.UpdateMember(ctx, req.Msg.CommitteeSlug, req.Msg.UserId, req.Msg.Role, req.Msg.Quoted)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MemberHandler) RemoveMember(ctx context.Context, req *connect.Request[committeesv1.RemoveMemberRequest]) (*connect.Response[committeesv1.RemoveMemberResponse], error) {
	resp, err := h.service.RemoveMember(ctx, req.Msg.CommitteeSlug, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (h *MemberHandler) SendInviteEmails(ctx context.Context, req *connect.Request[committeesv1.SendInviteEmailsRequest]) (*connect.Response[committeesv1.SendInviteEmailsResponse], error) {
	resp, err := h.service.SendInviteEmails(ctx, req.Msg.CommitteeSlug, req.Msg.MeetingId, req.Msg.BaseUrl, req.Msg.MemberIds)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
