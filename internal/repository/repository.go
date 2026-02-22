package repository

import (
	"context"
	"time"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

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
	GetUserByCommitteeAndUsername(ctx context.Context, slug, username string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)

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
	SetActiveMeeting(ctx context.Context, slug string, meetingID int64) error
	SetMeetingSignupOpen(ctx context.Context, id int64, open bool) error
	SetProtocolWriter(ctx context.Context, meetingID int64, attendeeID *int64) error
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

	// Motions
	CreateMotion(ctx context.Context, agendaPointID, blobID int64, title string) (*model.Motion, error)
	GetMotionByID(ctx context.Context, id int64) (*model.Motion, error)
	ListMotionsForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.Motion, error)
	DeleteMotion(ctx context.Context, id int64) error
	SetMotionVotes(ctx context.Context, id, votesFor, votesAgainst, votesAbstained, votesEligible int64) error

	// Agenda points
	CreateAgendaPoint(ctx context.Context, meetingID int64, title string) (*model.AgendaPoint, error)
	CreateSubAgendaPoint(ctx context.Context, meetingID, parentID int64, title string) (*model.AgendaPoint, error)
	ListAgendaPointsForMeeting(ctx context.Context, meetingID int64) ([]*model.AgendaPoint, error)
	ListSubAgendaPointsForMeeting(ctx context.Context, meetingID int64) ([]*model.AgendaPoint, error)
	GetAgendaPointByID(ctx context.Context, id int64) (*model.AgendaPoint, error)
	DeleteAgendaPoint(ctx context.Context, id int64) error
	SetCurrentAgendaPoint(ctx context.Context, meetingID int64, agendaPointID *int64) error
	UpdateAgendaPointProtocol(ctx context.Context, agendaPointID int64, protocol string) error
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
	DeleteUserByID(ctx context.Context, id int64) error
}
