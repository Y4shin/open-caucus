package handlers

func (h *Handler) passwordAuthEnabled() bool {
	if h.AuthConfig == nil {
		return true
	}
	return h.AuthConfig.PasswordEnabled
}

func (h *Handler) oauthAuthEnabled() bool {
	if h.AuthConfig == nil {
		return false
	}
	return h.AuthConfig.OAuthEnabled
}

