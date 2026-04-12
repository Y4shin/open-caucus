package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// AgendaItem represents one agenda point for the email template.
type AgendaItem struct {
	Number   string // e.g. "1", "1.1"
	Title    string
	Children []AgendaItem
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
	CustomMessage string // optional personal message from chairperson
	Language      string // "en" or "de"
}

// Localized strings for email templates.
type emailStrings struct {
	MeetingInvite string
	Hello         string
	YouAreInvited string
	When          string
	Agenda        string
	JoinMeeting   string
	CopyLink      string
	JoinHere      string
}

func getStrings(lang string) emailStrings {
	if lang == "de" {
		return emailStrings{
			MeetingInvite: "Einladung zur Sitzung",
			Hello:         "Hallo",
			YouAreInvited: "Sie sind eingeladen zu",
			When:          "Wann",
			Agenda:        "Tagesordnung",
			JoinMeeting:   "Zur Sitzung",
			CopyLink:      "Oder diesen Link kopieren",
			JoinHere:      "Hier beitreten",
		}
	}
	return emailStrings{
		MeetingInvite: "Meeting Invite",
		Hello:         "Hello",
		YouAreInvited: "You are invited to",
		When:          "When",
		Agenda:        "Agenda",
		JoinMeeting:   "Join Meeting",
		CopyLink:      "Or copy this link",
		JoinHere:      "Join here",
	}
}

type templateData struct {
	InviteData
	S emailStrings
}

var inviteHTMLTmpl = template.Must(template.New("invite_html").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin: 0; padding: 0;">
<table width="100%" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" style="padding: 20px 0;">
<table width="600" cellpadding="0" cellspacing="0" border="0" style="font-family: Arial, Helvetica, sans-serif; font-size: 14px; color: #333333;">
<tr><td style="padding: 0 0 16px 0;"><h2 style="margin: 0; font-size: 22px; color: #111;">{{.S.MeetingInvite}}</h2></td></tr>
<tr><td style="padding: 0 0 8px 0;">{{.S.Hello}} {{.MemberName}},</td></tr>
<tr><td style="padding: 0 0 8px 0;">{{.S.YouAreInvited}} <strong>{{.MeetingName}}</strong> ({{.CommitteeName}}).</td></tr>
{{if .MeetingDesc}}<tr><td style="padding: 0 0 8px 0; color: #555555;">{{.MeetingDesc}}</td></tr>{{end}}
{{if .StartAt}}<tr><td style="padding: 0 0 8px 0;"><strong>{{.S.When}}:</strong> {{.StartAt}}{{if .EndAt}} &#8212; {{.EndAt}}{{end}}</td></tr>{{end}}
{{if .CustomMessage}}<tr><td style="padding: 12px 16px; background-color: #f3f4f6; border-left: 4px solid #4f46e5;">{{.CustomMessage}}</td></tr>{{end}}
{{if .Agenda}}<tr><td style="padding: 16px 0 4px 0;"><strong style="font-size: 16px;">{{.S.Agenda}}</strong></td></tr>
<tr><td style="padding: 0 0 8px 0;">
<ol style="padding-left: 20px; margin: 4px 0;">
{{range .Agenda}}<li style="padding: 2px 0;">{{.Title}}{{if .Children}}<ol style="padding-left: 16px; margin: 2px 0;">{{range .Children}}<li style="padding: 1px 0;">{{.Title}}</li>{{end}}</ol>{{end}}</li>
{{end}}</ol>
</td></tr>{{end}}
<tr><td style="padding: 16px 0;">
<table cellpadding="0" cellspacing="0" border="0"><tr>
<td style="background-color: #4f46e5; padding: 10px 24px;"><a href="{{.JoinURL}}" style="color: #ffffff; font-weight: bold; font-size: 14px;">{{.S.JoinMeeting}}</a></td>
</tr></table>
</td></tr>
<tr><td style="padding: 0 0 8px 0; font-size: 12px; color: #666666;">{{.S.CopyLink}}: {{.JoinURL}}</td></tr>
</table>
</td></tr></table>
</body>
</html>`))

var inviteTextTmpl = template.Must(template.New("invite_text").Parse(
	`{{.S.Hello}} {{.MemberName}},

{{.S.YouAreInvited}} "{{.MeetingName}}" ({{.CommitteeName}}).
{{if .MeetingDesc}}
{{.MeetingDesc}}
{{end}}{{if .StartAt}}
{{.S.When}}: {{.StartAt}}{{if .EndAt}} — {{.EndAt}}{{end}}
{{end}}{{if .CustomMessage}}
> {{.CustomMessage}}
{{end}}{{if .Agenda}}
{{.S.Agenda}}:
{{range .Agenda}}  {{.Number}}. {{.Title}}{{range .Children}}
    - {{.Title}}{{end}}
{{end}}{{end}}
{{.S.JoinHere}}: {{.JoinURL}}
`))

// RenderMeetingInvite renders the HTML and plain-text bodies for a meeting invite.
func RenderMeetingInvite(data InviteData) (htmlBody, textBody string, err error) {
	lang := data.Language
	if lang == "" {
		lang = "en"
	}
	td := templateData{InviteData: data, S: getStrings(lang)}

	var hBuf, tBuf bytes.Buffer
	if err := inviteHTMLTmpl.Execute(&hBuf, td); err != nil {
		return "", "", fmt.Errorf("render invite html: %w", err)
	}
	if err := inviteTextTmpl.Execute(&tBuf, td); err != nil {
		return "", "", fmt.Errorf("render invite text: %w", err)
	}
	return hBuf.String(), tBuf.String(), nil
}

// InviteSubject returns the email subject line for a meeting invite.
func InviteSubject(meetingName, committeeName string) string {
	return strings.TrimSpace(fmt.Sprintf("Invite: %s — %s", meetingName, committeeName))
}
