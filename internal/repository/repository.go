package repository

import (
	"context"
	"time"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

// AgendaApplyPoint describes one desired agenda entry for bulk replacement.
// Key must be unique within one apply call; ParentKey references another point Key.
type AgendaApplyPoint struct {
	Key        string
	ExistingID *int64
	ParentKey  *string
	Title      string
	Position   int64
}

// VoteOptionInput represents one vote option to be persisted.
type VoteOptionInput struct {
	Label    string
	Position int64
}

// OpenBallotSubmission contains inputs for one open ballot submission.
type OpenBallotSubmission struct {
	VoteDefinitionID int64
	MeetingID        int64
	AttendeeID       int64
	Source           string
	ReceiptToken     string
	OptionIDs        []int64
}

// SecretBallotSubmission contains inputs for one secret ballot submission.
type SecretBallotSubmission struct {
	VoteDefinitionID    int64
	ReceiptToken        string
	EncryptedCommitment []byte
	CommitmentCipher    string
	CommitmentVersion   int64
	OptionIDs           []int64
}

// Repository defines the interface for data persistence and migration management.
type Repository interface {
	// Close closes the underlying database connection.
	Close() error

	// MigrateUp applies all pending migrations.
	MigrateUp() error

	// MigrateDown rolls back all migrations.
	MigrateDown() error

	// MigrateVersion migrates to a specific version.
	// Use version 0 to roll back all migrations.
	MigrateVersion(version uint) error

	// MigrateSteps applies n migration steps.
	// Positive n migrates up, negative n migrates down.
	MigrateSteps(n int) error

	// MigrationVersion returns the current migration version and whether
	// the database is dirty (migration failed midway).
	MigrationVersion() (version uint, dirty bool, err error)

	// User and authentication
	GetAccountByUsername(ctx context.Context, username string) (*model.Account, error)
	GetAccountByID(ctx context.Context, id int64) (*model.Account, error)
	CreateAccount(ctx context.Context, username, fullName, passwordHash string) (*model.Account, error)
	CreateOAuthAccount(ctx context.Context, username, fullName string) (*model.Account, error)
	GetPasswordCredential(ctx context.Context, accountID int64) (*model.PasswordCredential, error)
	GetOAuthIdentityByIssuerSubject(ctx context.Context, issuer, subject string) (*model.OAuthIdentity, error)
	UpsertOAuthIdentity(
		ctx context.Context,
		issuer, subject string,
		accountID int64,
		username, fullName, email *string,
		groupsJSON *string,
	) (*model.OAuthIdentity, error)
	GetUserByCommitteeAndUsername(ctx context.Context, slug, username string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	GetUserMembershipByAccountIDAndSlug(ctx context.Context, accountID int64, slug string) (*model.User, error)
	ListCommitteesByAccountID(ctx context.Context, accountID int64) ([]*model.Committee, error)
	SyncOAuthCommitteeMemberships(ctx context.Context, accountID int64, desired []model.OAuthDesiredMembership) error

	// Committee
	GetCommitteeBySlug(ctx context.Context, slug string) (*model.Committee, error)

	// Session management
	CreateSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context, before time.Time) error

	// Attendees
	CreateAttendee(ctx context.Context, meetingID int64, userID *int64, fullName, secret string, quoted bool) (*model.Attendee, error)
	GetAttendeeByUserIDAndMeetingID(ctx context.Context, userID, meetingID int64) (*model.Attendee, error)
	GetAttendeeByID(ctx context.Context, id int64) (*model.Attendee, error)
	GetAttendeeByMeetingIDAndSecret(ctx context.Context, meetingID int64, secret string) (*model.Attendee, error)
	ListAttendeesForMeeting(ctx context.Context, meetingID int64) ([]*model.Attendee, error)
	DeleteAttendee(ctx context.Context, id int64) error
	SetAttendeeIsChair(ctx context.Context, id int64, isChair bool) error
	SetAttendeeQuoted(ctx context.Context, id int64, quoted bool) error

	// Meetings
	GetMeetingByID(ctx context.Context, id int64) (*model.Meeting, error)
	ListMeetingsForCommittee(ctx context.Context, slug string, limit, offset int) ([]*model.Meeting, error)
	CountMeetingsForCommittee(ctx context.Context, slug string) (int64, error)
	CreateMeeting(ctx context.Context, committeeID int64, name, description, secret string, signupOpen bool) error
	DeleteMeeting(ctx context.Context, id int64) error
	SetActiveMeeting(ctx context.Context, slug string, meetingID *int64) error
	SetMeetingSignupOpen(ctx context.Context, id int64, open bool) error
	// SetMeetingSignupOpenWithVersion atomically sets signup_open and increments the version counter,
	// returning the new version.
	SetMeetingSignupOpenWithVersion(ctx context.Context, id int64, open bool) (int64, error)
	SetMeetingGenderQuotation(ctx context.Context, id int64, enabled bool) error
	SetMeetingFirstSpeakerQuotation(ctx context.Context, id int64, enabled bool) error
	SetMeetingModerator(ctx context.Context, id int64, moderatorID *int64) error

	// Binary blobs
	CreateBlob(ctx context.Context, filename, contentType string, sizeBytes int64, storagePath string) (*model.BinaryBlob, error)
	GetBlobByID(ctx context.Context, id int64) (*model.BinaryBlob, error)
	DeleteBlob(ctx context.Context, id int64) error

	// Agenda attachments
	CreateAttachment(ctx context.Context, agendaPointID, blobID int64, label *string) (*model.AgendaAttachment, error)
	GetAttachmentByID(ctx context.Context, id int64) (*model.AgendaAttachment, error)
	ListAttachmentsForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.AgendaAttachment, error)
	DeleteAttachment(ctx context.Context, id int64) error

	// Voting
	CreateVoteDefinition(ctx context.Context, meetingID, agendaPointID int64, name, visibility string, minSelections, maxSelections int64) (*model.VoteDefinition, error)
	UpdateVoteDefinitionDraft(ctx context.Context, id int64, meetingID, agendaPointID int64, name, visibility string, minSelections, maxSelections int64) (*model.VoteDefinition, error)
	OpenVoteWithEligibleVoters(ctx context.Context, voteDefinitionID int64, attendeeIDs []int64) (*model.VoteDefinition, error)
	CloseVote(ctx context.Context, voteDefinitionID int64) (*model.CloseVoteResult, error)
	ArchiveVote(ctx context.Context, voteDefinitionID int64) (*model.VoteDefinition, error)
	GetVoteDefinitionByID(ctx context.Context, id int64) (*model.VoteDefinition, error)
	ListVoteDefinitionsForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.VoteDefinition, error)
	ReplaceVoteOptions(ctx context.Context, voteDefinitionID int64, options []VoteOptionInput) error
	ListVoteOptions(ctx context.Context, voteDefinitionID int64) ([]*model.VoteOption, error)
	ListEligibleVoters(ctx context.Context, voteDefinitionID int64) ([]*model.EligibleVoter, error)
	RegisterVoteCast(ctx context.Context, voteDefinitionID, meetingID, attendeeID int64, source string) (*model.VoteCast, error)
	ListVoteCasts(ctx context.Context, voteDefinitionID int64) ([]*model.VoteCast, error)
	SubmitOpenBallot(ctx context.Context, submission OpenBallotSubmission) (*model.VoteBallot, error)
	SubmitSecretBallot(ctx context.Context, submission SecretBallotSubmission) (*model.VoteBallot, error)
	VerifyOpenBallotByReceipt(ctx context.Context, voteDefinitionID int64, receiptToken string) (*model.VoteOpenVerification, error)
	VerifySecretBallotByReceipt(ctx context.Context, voteDefinitionID int64, receiptToken string) (*model.VoteSecretVerification, error)
	GetVoteTallies(ctx context.Context, voteDefinitionID int64) ([]*model.VoteTallyRow, error)
	GetVoteSubmissionStats(ctx context.Context, voteDefinitionID int64) (*model.VoteSubmissionStats, error)
	GetVoteSubmissionStatsLive(ctx context.Context, voteDefinitionID int64) (*model.VoteSubmissionStats, error)

	// Agenda points
	CreateAgendaPoint(ctx context.Context, meetingID int64, title string) (*model.AgendaPoint, error)
	CreateSubAgendaPoint(ctx context.Context, meetingID, parentID int64, title string) (*model.AgendaPoint, error)
	ListAgendaPointsForMeeting(ctx context.Context, meetingID int64) ([]*model.AgendaPoint, error)
	ListSubAgendaPointsForParent(ctx context.Context, meetingID, parentID int64) ([]*model.AgendaPoint, error)
	ListSubAgendaPointsForMeeting(ctx context.Context, meetingID int64) ([]*model.AgendaPoint, error)
	GetAgendaPointByID(ctx context.Context, id int64) (*model.AgendaPoint, error)
	DeleteAgendaPoint(ctx context.Context, id int64) error
	MoveAgendaPointUp(ctx context.Context, meetingID, agendaPointID int64) error
	MoveAgendaPointDown(ctx context.Context, meetingID, agendaPointID int64) error
	ApplyAgendaPoints(ctx context.Context, meetingID int64, points []AgendaApplyPoint, deleteIDs []int64) error
	SetCurrentAgendaPoint(ctx context.Context, meetingID int64, agendaPointID *int64) error
	SetCurrentAttachment(ctx context.Context, agendaPointID, attachmentID int64) error
	ClearCurrentDocument(ctx context.Context, agendaPointID int64) error
	SetAgendaPointGenderQuotation(ctx context.Context, id int64, enabled *bool) error
	SetAgendaPointFirstSpeakerQuotation(ctx context.Context, id int64, enabled *bool) error
	SetAgendaPointModerator(ctx context.Context, id int64, moderatorID *int64) error

	// Speakers list
	AddSpeaker(ctx context.Context, agendaPointID, attendeeID int64, speakerType string, genderQuoted, firstSpeaker bool) (*model.SpeakerEntry, error)
	ListSpeakersForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.SpeakerEntry, error)
	GetSpeakerEntryByID(ctx context.Context, id int64) (*model.SpeakerEntry, error)
	DeleteSpeaker(ctx context.Context, id int64) error
	SetSpeakerSpeaking(ctx context.Context, id, agendaPointID int64) error
	SetSpeakerDone(ctx context.Context, id int64) error
	SetSpeakerWithdrawn(ctx context.Context, id int64) error
	HasAttendeeSpokenOnAgendaPoint(ctx context.Context, agendaPointID, attendeeID int64) (bool, error)
	SetSpeakerPriority(ctx context.Context, id int64, priority bool) error
	RecomputeSpeakerOrder(ctx context.Context, agendaPointID int64) error

	// Admin - Committee management
	ListAllCommittees(ctx context.Context, limit, offset int) ([]*model.Committee, error)
	CountAllCommittees(ctx context.Context) (int64, error)
	CreateCommitteeWithSlug(ctx context.Context, name, slug string) error
	DeleteCommitteeBySlug(ctx context.Context, slug string) error
	GetCommitteeIDBySlug(ctx context.Context, slug string) (int64, error)

	// Admin - User management
	ListUsersInCommittee(ctx context.Context, slug string, limit, offset int) ([]*model.User, error)
	CountUsersInCommittee(ctx context.Context, slug string) (int64, error)
	CreateUser(ctx context.Context, committeeID int64, username, passwordHash, fullName string, quoted bool, role string) error
	AssignAccountToCommittee(ctx context.Context, committeeID, accountID int64, quoted bool, role string) error
	UpdateUserMembership(ctx context.Context, userID int64, quoted bool, role string) error
	IsOAuthManagedMembership(ctx context.Context, userID int64) (bool, error)
	DeleteUserByID(ctx context.Context, id int64) error

	// Admin - Account admin flag
	SetAccountIsAdmin(ctx context.Context, accountID int64, isAdmin bool) error
	CountAllAccounts(ctx context.Context) (int64, error)
	ListAllAccounts(ctx context.Context, limit, offset int) ([]*model.Account, error)
	ListUnassignedAccountsForCommittee(ctx context.Context, committeeID int64) ([]*model.Account, error)
	ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx context.Context, slug string) ([]*model.OAuthCommitteeGroupRule, error)
	ListAllOAuthCommitteeGroupRules(ctx context.Context) ([]*model.OAuthCommitteeGroupRule, error)
	CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx context.Context, slug, groupName, role string) (*model.OAuthCommitteeGroupRule, error)
	DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug(ctx context.Context, id int64, slug string) error
}
