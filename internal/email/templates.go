package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// AgendaItem represents one agenda point for the email template.
type AgendaItem struct {
	Number string // e.g. "1", "1.1"
	Title  string
}

// InviteData holds the data for rendering a meeting invite email.
type InviteData struct {
	MemberName    string
	CommitteeName string
	MeetingName   string
	MeetingDesc   string
	JoinURL       string
	StartAt       string // formatted datetime, empty if not set
	EndAt         string // formatted datetime, empty if not set
	Agenda        []AgendaItem
}

var inviteHTMLTmpl = template.Must(template.New("invite_html").Parse(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2>Meeting Invite</h2>
<p>Hello {{.MemberName}},</p>
<p>You are invited to <strong>{{.MeetingName}}</strong> ({{.CommitteeName}}).</p>
{{if .MeetingDesc}}<p style="color: #555;">{{.MeetingDesc}}</p>{{end}}
{{if .StartAt}}<p><strong>When:</strong> {{.StartAt}}{{if .EndAt}} — {{.EndAt}}{{end}}</p>{{end}}
{{if .Agenda}}<h3 style="margin-top: 16px;">Agenda</h3>
<ol style="padding-left: 20px; color: #333;">
{{range .Agenda}}<li>{{.Title}}</li>
{{end}}</ol>{{end}}
<p style="margin-top: 16px;"><a href="{{.JoinURL}}" style="display: inline-block; padding: 10px 20px; background: #4f46e5; color: white; text-decoration: none; border-radius: 6px;">Join Meeting</a></p>
<p style="font-size: 0.85em; color: #666;">Or copy this link: {{.JoinURL}}</p>
</body>
</html>`))

var inviteTextTmpl = template.Must(template.New("invite_text").Parse(
	`Hello {{.MemberName}},

You are invited to "{{.MeetingName}}" ({{.CommitteeName}}).
{{if .MeetingDesc}}
{{.MeetingDesc}}
{{end}}{{if .StartAt}}
When: {{.StartAt}}{{if .EndAt}} — {{.EndAt}}{{end}}
{{end}}{{if .Agenda}}
Agenda:
{{range .Agenda}}  {{.Number}}. {{.Title}}
{{end}}{{end}}
Join here: {{.JoinURL}}
`))

// RenderMeetingInvite renders the HTML and plain-text bodies for a meeting invite.
func RenderMeetingInvite(data InviteData) (htmlBody, textBody string, err error) {
	var hBuf, tBuf bytes.Buffer
	if err := inviteHTMLTmpl.Execute(&hBuf, data); err != nil {
		return "", "", fmt.Errorf("render invite html: %w", err)
	}
	if err := inviteTextTmpl.Execute(&tBuf, data); err != nil {
		return "", "", fmt.Errorf("render invite text: %w", err)
	}
	return hBuf.String(), tBuf.String(), nil
}

// InviteSubject returns the email subject line for a meeting invite.
func InviteSubject(meetingName, committeeName string) string {
	return strings.TrimSpace(fmt.Sprintf("Invite: %s — %s", meetingName, committeeName))
}
