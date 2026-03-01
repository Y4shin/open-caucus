package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/Y4shin/conference-tool/internal/broker"
	sqliterepo "github.com/Y4shin/conference-tool/internal/repository/sqlite"
	"github.com/Y4shin/conference-tool/internal/routes"
)

func TestModerateVoteCreate_ReturnsCreatedVoteInPanelList(t *testing.T) {
	repo, err := sqliterepo.New(":memory:")
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	defer repo.Close()

	if err := repo.MigrateUp(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	ctx := context.Background()
	if err := repo.CreateCommitteeWithSlug(ctx, "Committee", "committee"); err != nil {
		t.Fatalf("create committee: %v", err)
	}
	committeeID, err := repo.GetCommitteeIDBySlug(ctx, "committee")
	if err != nil {
		t.Fatalf("get committee id: %v", err)
	}
	if err := repo.CreateMeeting(ctx, committeeID, "Meeting", "", "secret", false); err != nil {
		t.Fatalf("create meeting: %v", err)
	}
	meetings, err := repo.ListMeetingsForCommittee(ctx, "committee", 10, 0)
	if err != nil {
		t.Fatalf("list meetings: %v", err)
	}
	if len(meetings) != 1 {
		t.Fatalf("expected 1 meeting, got %d", len(meetings))
	}
	meetingID := meetings[0].ID

	agendaPoint, err := repo.CreateAgendaPoint(ctx, meetingID, "Agenda")
	if err != nil {
		t.Fatalf("create agenda point: %v", err)
	}
	if err := repo.SetCurrentAgendaPoint(ctx, meetingID, &agendaPoint.ID); err != nil {
		t.Fatalf("set current agenda point: %v", err)
	}

	h := &Handler{Broker: broker.NewMemoryBroker(), Repository: repo}
	meetingIDStr := strconv.FormatInt(meetingID, 10)
	form := url.Values{
		"name":           {"Budget Vote"},
		"visibility":     {"open"},
		"min_selections": {"1"},
		"max_selections": {"1"},
		"options_text":   {"Yes\nNo\nAbstain"},
	}
	req := httptest.NewRequest(http.MethodPost, "/committee/committee/meeting/"+meetingIDStr+"/votes/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	input, _, err := h.ModerateVoteCreate(ctx, req, routes.RouteParams{Slug: "committee", MeetingId: meetingIDStr})
	if err != nil {
		t.Fatalf("moderate vote create: %v", err)
	}
	if input == nil {
		t.Fatal("expected non-nil panel input")
	}
	if len(input.Votes) != 1 {
		t.Fatalf("expected exactly 1 vote in panel list, got %d", len(input.Votes))
	}
	if input.Votes[0].Name != "Budget Vote" {
		t.Fatalf("expected vote name Budget Vote, got %q", input.Votes[0].Name)
	}
}
