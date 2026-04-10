package config

// AuthConfig controls enabled authentication providers and OAuth/OIDC behavior.
type AuthConfig struct {
	PasswordEnabled bool `mapstructure:"password_enabled" env:"AUTH_PASSWORD_ENABLED" default:"true" env_doc:"Enable password login provider" env_default:"true"`
	OAuthEnabled    bool `mapstructure:"oauth_enabled" env:"AUTH_OAUTH_ENABLED" default:"false" env_doc:"Enable OAuth/OIDC login provider" env_default:"false"`

	OAuthIssuerURL   string `mapstructure:"oauth_issuer_url" env:"OAUTH_ISSUER_URL" default:"" env_doc:"OIDC issuer URL (required when OAuth is enabled)"`
	OAuthClientID    string `mapstructure:"oauth_client_id" env:"OAUTH_CLIENT_ID" default:"" env_doc:"OIDC client ID (required when OAuth is enabled)"`
	OAuthClientSecret string `mapstructure:"oauth_client_secret" env:"OAUTH_CLIENT_SECRET" default:"" env_doc:"OIDC client secret (required when OAuth is enabled)"`
	OAuthRedirectURL string `mapstructure:"oauth_redirect_url" env:"OAUTH_REDIRECT_URL" default:"" env_doc:"OAuth callback URL (required when OAuth is enabled)"`
	OAuthScopes      []string `mapstructure:"oauth_scopes" env:"OAUTH_SCOPES" default:"openid,profile,email" env_doc:"OAuth scopes as comma-separated values" env_default:"openid,profile,email"`

	OAuthGroupsClaim    string   `mapstructure:"oauth_groups_claim" env:"OAUTH_GROUPS_CLAIM" default:"groups" env_doc:"JWT claim name used for group extraction" env_default:"groups"`
	OAuthUsernameClaims []string `mapstructure:"oauth_username_claims" env:"OAUTH_USERNAME_CLAIMS" default:"preferred_username,email,sub" env_doc:"Fallback JWT claims for username, comma-separated" env_default:"preferred_username,email,sub"`
	OAuthFullNameClaims []string `mapstructure:"oauth_full_name_claims" env:"OAUTH_FULL_NAME_CLAIMS" default:"name,preferred_username,email" env_doc:"Fallback JWT claims for full name, comma-separated" env_default:"name,preferred_username,email"`

	OAuthProvisioningMode string   `mapstructure:"oauth_provisioning_mode" env:"OAUTH_PROVISIONING_MODE" default:"preprovisioned" env_doc:"OAuth provisioning mode (preprovisioned, auto_create)" validate:"oneOf(preprovisioned,auto_create)" env_default:"preprovisioned"`
	OAuthRequiredGroups   []string `mapstructure:"oauth_required_groups" env:"OAUTH_REQUIRED_GROUPS" default:"" env_doc:"Optional required groups for OAuth provisioning, comma-separated"`
	OAuthAdminGroup              string   `mapstructure:"oauth_admin_group" env:"OAUTH_ADMIN_GROUP" default:"" env_doc:"Optional group that toggles local admin flag on each OAuth login"`
	OAuthCommitteeGroupPrefix    string   `mapstructure:"oauth_committee_group_prefix" env:"OAUTH_COMMITTEE_GROUP_PREFIX" default:"" env_doc:"When set, only OIDC groups with this prefix can be used in committee group rules"`
	OAuthStateTTLSeconds         int      `mapstructure:"oauth_state_ttl_seconds" env:"OAUTH_STATE_TTL_SECONDS" default:"300" env_doc:"Max age for OAuth state cookie in seconds" validate:"enforceMin(1)" env_default:"300"`
}

