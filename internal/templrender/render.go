package templrender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"sort"

	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/session"
	"github.com/Y4shin/conference-tool/internal/templates"
	"github.com/a-h/templ"
	"github.com/invopop/ctxi18n"
)

type ContextProfile struct {
	Locale       string                 `json:"locale"`
	HadURLPrefix bool                   `json:"had_url_prefix"`
	Session      *session.SessionData   `json:"session"`
	CurrentUser  *session.CurrentUser   `json:"current_user"`
	CurrentAttendee *session.CurrentAttendee `json:"current_attendee"`
}

type Spec struct {
	Name     string
	NewInput func() any
	Render   func(ctx context.Context, input any) (templ.Component, error)
}

var registry = map[string]Spec{
	"LoginPageTemplate": {
		Name:     "LoginPageTemplate",
		NewInput: func() any { return &templates.LoginPageInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.LoginPageInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.LoginPageInput, got %T", input)
			}
			return templates.LoginPageTemplate(*value), nil
		},
	},
	"AdminLoginTemplate": {
		Name:     "AdminLoginTemplate",
		NewInput: func() any { return &templates.AdminLoginInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.AdminLoginInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.AdminLoginInput, got %T", input)
			}
			return templates.AdminLoginTemplate(*value), nil
		},
	},
	"HomeTemplate": {
		Name:     "HomeTemplate",
		NewInput: func() any { return &templates.HomeInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.HomeInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.HomeInput, got %T", input)
			}
			return templates.HomeTemplate(*value), nil
		},
	},
	"CommitteePageTemplate": {
		Name:     "CommitteePageTemplate",
		NewInput: func() any { return &templates.CommitteePageInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.CommitteePageInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.CommitteePageInput, got %T", input)
			}
			return templates.CommitteePageTemplate(*value), nil
		},
	},
	"MeetingListPartial": {
		Name:     "MeetingListPartial",
		NewInput: func() any { return &templates.MeetingListPartialInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.MeetingListPartialInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.MeetingListPartialInput, got %T", input)
			}
			return templates.MeetingListPartial(*value), nil
		},
	},
	"AdminDashboardTemplate": {
		Name:     "AdminDashboardTemplate",
		NewInput: func() any { return &templates.AdminDashboardInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.AdminDashboardInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.AdminDashboardInput, got %T", input)
			}
			return templates.AdminDashboardTemplate(*value), nil
		},
	},
	"CommitteeListPartial": {
		Name:     "CommitteeListPartial",
		NewInput: func() any { return &templates.CommitteeListPartialInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.CommitteeListPartialInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.CommitteeListPartialInput, got %T", input)
			}
			return templates.CommitteeListPartial(*value), nil
		},
	},
	"AdminAccountsTemplate": {
		Name:     "AdminAccountsTemplate",
		NewInput: func() any { return &templates.AdminAccountsInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.AdminAccountsInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.AdminAccountsInput, got %T", input)
			}
			return templates.AdminAccountsTemplate(*value), nil
		},
	},
	"AccountListPartial": {
		Name:     "AccountListPartial",
		NewInput: func() any { return &templates.AccountListPartialInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.AccountListPartialInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.AccountListPartialInput, got %T", input)
			}
			return templates.AccountListPartial(*value), nil
		},
	},
	"AdminCommitteeUsersTemplate": {
		Name:     "AdminCommitteeUsersTemplate",
		NewInput: func() any { return &templates.AdminCommitteeUsersInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.AdminCommitteeUsersInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.AdminCommitteeUsersInput, got %T", input)
			}
			return templates.AdminCommitteeUsersTemplate(*value), nil
		},
	},
	"UserListPartial": {
		Name:     "UserListPartial",
		NewInput: func() any { return &templates.UserListPartialInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.UserListPartialInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.UserListPartialInput, got %T", input)
			}
			return templates.UserListPartial(*value), nil
		},
	},
	"MeetingJoinTemplate": {
		Name:     "MeetingJoinTemplate",
		NewInput: func() any { return &templates.MeetingJoinInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.MeetingJoinInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.MeetingJoinInput, got %T", input)
			}
			return templates.MeetingJoinTemplate(*value), nil
		},
	},
	"AttendeeLoginTemplate": {
		Name:     "AttendeeLoginTemplate",
		NewInput: func() any { return &templates.AttendeeLoginInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.AttendeeLoginInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.AttendeeLoginInput, got %T", input)
			}
			return templates.AttendeeLoginTemplate(*value), nil
		},
	},
	"DocsSearchResults": {
		Name:     "DocsSearchResults",
		NewInput: func() any { return &templates.DocsSearchResultsInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.DocsSearchResultsInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.DocsSearchResultsInput, got %T", input)
			}
			return templates.DocsSearchResults(*value), nil
		},
	},
	"ReceiptsVaultTemplate": {
		Name:     "ReceiptsVaultTemplate",
		NewInput: func() any { return &templates.ReceiptsVaultInput{} },
		Render: func(ctx context.Context, input any) (templ.Component, error) {
			value, ok := input.(*templates.ReceiptsVaultInput)
			if !ok {
				return nil, fmt.Errorf("expected *templates.ReceiptsVaultInput, got %T", input)
			}
			return templates.ReceiptsVaultTemplate(*value), nil
		},
	},
}

func Specs() map[string]Spec {
	return maps.Clone(registry)
}

func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func NewInput(name string) (any, error) {
	spec, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown component %q", name)
	}
	return spec.NewInput(), nil
}

func BuildContext(profile ContextProfile) (context.Context, error) {
	ctx := context.Background()
	selectedLocale := profile.Locale
	if selectedLocale == "" {
		selectedLocale = "en"
	}
	ctx = locale.WithLocale(ctx, selectedLocale)
	ctx = locale.WithHadURLPrefix(ctx, profile.HadURLPrefix)
	i18nCtx, err := ctxi18n.WithLocale(ctx, selectedLocale)
	if err != nil {
		return nil, err
	}
	ctx = i18nCtx
	if profile.Session != nil {
		ctx = session.WithSession(ctx, profile.Session)
	}
	if profile.CurrentUser != nil {
		ctx = session.WithCurrentUser(ctx, profile.CurrentUser)
	}
	if profile.CurrentAttendee != nil {
		ctx = session.WithCurrentAttendee(ctx, profile.CurrentAttendee)
	}
	return ctx, nil
}

func Render(name string, profile ContextProfile, input any) ([]byte, error) {
	spec, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown component %q", name)
	}
	ctx, err := BuildContext(profile)
	if err != nil {
		return nil, fmt.Errorf("build render context: %w", err)
	}
	component, err := spec.Render(ctx, input)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Write(w io.Writer, name string, profile ContextProfile, input any) error {
	rendered, err := Render(name, profile, input)
	if err != nil {
		return err
	}
	_, err = w.Write(rendered)
	return err
}
