package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// InviteData holds the data for rendering a meeting invite email.
type InviteData struct {
	MemberName    string
	CommitteeName string
	MeetingName   string
	JoinURL       string
	HasICS        bool // true if an ICS event is attached
}

var inviteHTMLTmpl = template.Must(template.New("invite_html").Parse(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2>Meeting Invite</h2>
<p>Hello {{.MemberName}},</p>
<p>You are invited to <strong>{{.MeetingName}}</strong> ({{.CommitteeName}}).</p>
<p><a href="{{.JoinURL}}" style="display: inline-block; padding: 10px 20px; background: #4f46e5; color: white; text-decoration: none; border-radius: 6px;">Join Meeting</a></p>
<p style="font-size: 0.85em; color: #666;">Or copy this link: {{.JoinURL}}</p>
</body>
</html>`))

var inviteTextTmpl = template.Must(template.New("invite_text").Parse(
	`Hello {{.MemberName}},

You are invited to "{{.MeetingName}}" ({{.CommitteeName}}).

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
