package model

import "time"

// OAuthIdentity links an external OIDC subject to a local account.
type OAuthIdentity struct {
	ID         int64
	Issuer     string
	Subject    string
	AccountID  int64
	Username   *string
	FullName   *string
	Email      *string
	GroupsJSON *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

