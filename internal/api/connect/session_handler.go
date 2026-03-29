package apiconnect

import (
	"context"
	"net/http"

	connect "connectrpc.com/connect"

	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
	sessionv1connect "github.com/Y4shin/conference-tool/gen/go/conference/session/v1/sessionv1connect"
	sessionservice "github.com/Y4shin/conference-tool/internal/services/session"
)

type SessionHandler struct {
	sessionv1connect.UnimplementedSessionServiceHandler
	service *sessionservice.Service
}

func NewSessionHandler(service *sessionservice.Service) *SessionHandler {
	return &SessionHandler{service: service}
}

func (h *SessionHandler) GetSession(ctx context.Context, _ *connect.Request[sessionv1.GetSessionRequest]) (*connect.Response[sessionv1.GetSessionResponse], error) {
	bootstrap, err := h.service.GetSessionBootstrap(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sessionv1.GetSessionResponse{Session: bootstrap}), nil
}

func (h *SessionHandler) Login(ctx context.Context, req *connect.Request[sessionv1.LoginRequest]) (*connect.Response[sessionv1.LoginResponse], error) {
	bootstrap, cookie, err := h.service.Login(ctx, req.Msg.Username, req.Msg.Password)
	if err != nil {
		return nil, err
	}
	resp := connect.NewResponse(&sessionv1.LoginResponse{Session: bootstrap})
	addCookie(resp, cookie)
	return resp, nil
}

func (h *SessionHandler) Logout(ctx context.Context, req *connect.Request[sessionv1.LogoutRequest]) (*connect.Response[sessionv1.LogoutResponse], error) {
	var signedID string
	if cookie, err := requestCookie(req.Header(), "session_id"); err == nil {
		signedID = cookie.Value
	}

	msg, clearCookie, err := h.service.Logout(ctx, signedID)
	if err != nil {
		return nil, err
	}
	resp := connect.NewResponse(msg)
	addCookie(resp, clearCookie)
	return resp, nil
}

func addCookie[T any](resp *connect.Response[T], cookie *http.Cookie) {
	if resp == nil || cookie == nil {
		return
	}
	resp.Header().Add("Set-Cookie", cookie.String())
}

func requestCookie(header http.Header, name string) (*http.Cookie, error) {
	req := &http.Request{Header: header}
	return req.Cookie(name)
}

func headerRequest(header http.Header) *http.Request {
	return &http.Request{Header: header}
}
