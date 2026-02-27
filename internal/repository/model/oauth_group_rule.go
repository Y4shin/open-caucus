package model

import "time"

// OAuthCommitteeGroupRule maps an external group to committee access.
type OAuthCommitteeGroupRule struct {
	ID            int64
	CommitteeID   int64
	CommitteeSlug string
	GroupName     string
	Role          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OAuthDesiredMembership expresses desired committee access after evaluating group rules.
type OAuthDesiredMembership struct {
	CommitteeID int64
	Role        string
}

