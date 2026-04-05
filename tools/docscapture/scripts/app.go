package scripts

import (
	"context"
	"fmt"
	"strings"

	playwright "github.com/playwright-community/playwright-go"

	"github.com/Y4shin/open-caucus/internal/repository"
	"github.com/Y4shin/open-caucus/internal/repository/model"
	"github.com/Y4shin/open-caucus/tools/docscapture"
)

const (
	gifCursorMoveSteps    = 28
	gifClickDelayMS       = 130.0
	gifPauseAfterMoveMS   = 260.0
	gifPauseAfterClickMS  = 520.0
	gifPauseAfterActionMS = 720.0
)

type appFixture struct {
	CommitteeSlug string
	MeetingID     int64
	AgendaPointID int64

	ChairUsername string
	ChairPassword string

	MemberUsername string
	MemberPassword string

	JoinMemberUsername string
	JoinMemberPassword string

	GuestSecret string
}

func appScreenshotHomeCommitteesScript() Script {
	return Script{
		Name:        "app.screenshot-home-committees",
		Description: "Home page listing committee memberships",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:     common,
				Variant:    variant,
				OutputName: appOutputName("app-home-committees", variant, "png"),
				FullPage:   true,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.MemberUsername, fixture.MemberPassword); err != nil {
						return err
					}
					if err := page.Locator("ul.list").First().WaitFor(); err != nil {
						return fmt.Errorf("wait home committees list: %w", err)
					}
					if err := highlight(page, "ul.list"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotCommitteeDashboardChairScript() Script {
	return Script{
		Name:        "app.screenshot-committee-dashboard-chair",
		Description: "Chairperson view of committee dashboard",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-committee-dashboard-chair", variant, "png"),
				FullPage:    true,
				InitialPath: committeePath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, committeePath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("#meeting-list-container").WaitFor(); err != nil {
						return fmt.Errorf("wait chair committee dashboard: %w", err)
					}
					if err := highlight(page, "#meeting-list-container"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotCommitteeDashboardMemberActiveScript() Script {
	return Script{
		Name:        "app.screenshot-committee-dashboard-member-active",
		Description: "Member view with active meeting card and join action",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:     common,
				Variant:    variant,
				OutputName: appOutputName("app-committee-dashboard-member-active", variant, "png"),
				FullPage:   true,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.MemberUsername, fixture.MemberPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, committeePath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("[data-testid='committee-active-meeting-card']").WaitFor(); err != nil {
						return fmt.Errorf("wait member active meeting card: %w", err)
					}
					if err := highlight(page, "[data-testid='committee-join-active-meeting']"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotModerateOverviewScript() Script {
	return Script{
		Name:        "app.screenshot-moderate-overview",
		Description: "Moderate workspace overview with agenda, attendees, and speakers panes",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-moderate-overview", variant, "png"),
				FullPage:    true,
				InitialPath: moderatePath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, moderatePath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("#moderate-sse-root").WaitFor(); err != nil {
						return fmt.Errorf("wait moderate root: %w", err)
					}
					if err := highlightMany(page, "#moderate-left-controls", "#moderate-right-resizable-stack"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotAgendaToolsAttachmentsScript() Script {
	return Script{
		Name:        "app.screenshot-agenda-tools-attachments",
		Description: "Agenda point tools page with attachments and current document controls",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-agenda-tools-attachments", variant, "png"),
				FullPage:    true,
				InitialPath: agendaToolsPath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, agendaToolsPath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("#attachment-list-ap-" + fmt.Sprintf("%d", fixture.AgendaPointID)).WaitFor(); err != nil {
						return fmt.Errorf("wait attachment list: %w", err)
					}
					attachmentContainer := "#attachment-list-ap-" + fmt.Sprintf("%d", fixture.AgendaPointID)
					if err := highlight(page, attachmentContainer); err != nil {
						return err
					}
					if err := highlightFirstAvailable(
						page,
						attachmentContainer+" button:has-text('Clear')",
						attachmentContainer+" button:has-text('Set as Current')",
					); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotLiveViewWithSpeakersScript() Script {
	return Script{
		Name:        "app.screenshot-live-view-with-speakers",
		Description: "Live attendee page showing active speakers and vote panel",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-live-view-with-speakers", variant, "png"),
				FullPage:    true,
				InitialPath: attendeeLoginPath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := loginAttendee(page, common.BaseURL, attendeeLoginPath(fixture), livePath(fixture), fixture.GuestSecret); err != nil {
						return err
					}
					if err := page.Locator("#attendee-speakers-list").WaitFor(); err != nil {
						return fmt.Errorf("wait attendee speakers panel: %w", err)
					}
					if err := highlightMany(page, "#attendee-speakers-list", "#live-votes-panel"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotJoinPageMemberScript() Script {
	return Script{
		Name:        "app.screenshot-join-page-member",
		Description: "Join page for authenticated member before signup",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-join-page-member", variant, "png"),
				FullPage:    true,
				InitialPath: joinPath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.JoinMemberUsername, fixture.JoinMemberPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, joinPath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("form[action$='/join']").First().WaitFor(); err != nil {
						return fmt.Errorf("wait member join form: %w", err)
					}
					if err := highlight(page, "form[action$='/join'] button[type='submit']"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotGuestSignupFormScript() Script {
	return Script{
		Name:        "app.screenshot-guest-signup-form",
		Description: "Guest signup form on meeting join page",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-guest-signup-form", variant, "png"),
				FullPage:    true,
				InitialPath: joinPath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := gotoPath(page, common.BaseURL, joinPath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("input[name='full_name']").WaitFor(); err != nil {
						return fmt.Errorf("wait guest signup form: %w", err)
					}
					if err := highlight(page, "input[name='full_name']"); err != nil {
						return err
					}
					if err := highlightFirstAvailable(
						page,
						"form button[type='submit']",
						"form input[type='submit']",
					); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotAttendeeLoginScript() Script {
	return Script{
		Name:        "app.screenshot-attendee-login",
		Description: "Attendee access-code login page",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-attendee-login", variant, "png"),
				FullPage:    true,
				InitialPath: attendeeLoginPath(fixture),
				Scenario: func(page playwright.Page) error {
					if err := gotoPath(page, common.BaseURL, attendeeLoginPath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("input[name='secret']").WaitFor(); err != nil {
						return fmt.Errorf("wait attendee login form: %w", err)
					}
					if err := highlight(page, "input[name='secret']"); err != nil {
						return err
					}
					if err := highlightFirstAvailable(
						page,
						"form button[type='submit']",
						"form input[type='submit']",
					); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotReceiptsVaultScript() Script {
	return Script{
		Name:        "app.screenshot-receipts-vault",
		Description: "Public receipts vault and verification workspace",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-receipts-vault", variant, "png"),
				FullPage:    true,
				InitialPath: "/receipts",
				Scenario: func(page playwright.Page) error {
					if err := page.Locator("#receipts-refresh").WaitFor(); err != nil {
						return fmt.Errorf("wait receipts controls: %w", err)
					}
					if err := highlightMany(page, "#receipts-refresh", "#receipts-clear", "#receipts-list"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotAdminLoginScript() Script {
	return Script{
		Name:        "app.screenshot-admin-login",
		Description: "Admin login page with card styling and back button",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:      common,
				Variant:     variant,
				OutputName:  appOutputName("app-admin-login", variant, "png"),
				FullPage:    true,
				InitialPath: "/admin/login",
				Scenario: func(page playwright.Page) error {
					if err := page.Locator("fieldset.fieldset").WaitFor(); err != nil {
						return fmt.Errorf("wait admin login card: %w", err)
					}
					if err := highlight(page, "fieldset.fieldset"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotAdminDashboardScript() Script {
	return Script{
		Name:        "app.screenshot-admin-dashboard",
		Description: "Admin dashboard with committee list and responsive form",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			if _, err := seedAppFixture(ctx, env.Seeder()); err != nil {
				return "", err
			}
			if _, err := env.Seeder().CreateAdminAccount(ctx, "admin-docs", "admin-docs-pass", "Admin Demo"); err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:     common,
				Variant:    variant,
				OutputName: appOutputName("app-admin-dashboard", variant, "png"),
				FullPage:   true,
				Scenario: func(page playwright.Page) error {
					if err := gotoPath(page, common.BaseURL, "/admin/login"); err != nil {
						return err
					}
					if err := page.Locator("input[name='username']").Fill("admin-docs"); err != nil {
						return fmt.Errorf("fill admin username: %w", err)
					}
					if err := page.Locator("input[name='password']").Fill("admin-docs-pass"); err != nil {
						return fmt.Errorf("fill admin password: %w", err)
					}
					if err := page.Locator("input[name='password']").Press("Enter"); err != nil {
						return fmt.Errorf("submit admin login: %w", err)
					}
					if err := page.WaitForURL(absoluteURL(common.BaseURL, "/admin")); err != nil {
						return fmt.Errorf("wait admin dashboard: %w", err)
					}
					if err := page.Locator("#committee-list").WaitFor(); err != nil {
						return fmt.Errorf("wait committee list: %w", err)
					}
					if err := highlightMany(page, "#create-committee-form", "#committee-list"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotMeetingWizardScript() Script {
	return Script{
		Name:        "app.screenshot-meeting-wizard",
		Description: "Meeting creation wizard dialog showing the agenda step",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:     common,
				Variant:    variant,
				OutputName: appOutputName("app-meeting-wizard", variant, "png"),
				FullPage:   false,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, committeePath(fixture)); err != nil {
						return err
					}
					createBtn := page.Locator("[data-testid='committee-create-form']")
					if err := createBtn.WaitFor(); err != nil {
						return fmt.Errorf("wait create meeting button: %w", err)
					}
					if err := createBtn.Click(); err != nil {
						return fmt.Errorf("click create meeting: %w", err)
					}
					// Fill basics and advance to agenda step
					if err := page.Locator("#wizard-name").Fill("Example Meeting"); err != nil {
						return fmt.Errorf("fill wizard name: %w", err)
					}
					nextBtn := page.Locator("dialog .modal-action button.btn-primary")
					if err := nextBtn.Click(); err != nil {
						return fmt.Errorf("click next: %w", err)
					}
					// Wait for the agenda step editor to appear
					if err := page.Locator("[data-agenda-import-lines]").WaitFor(); err != nil {
						return fmt.Errorf("wait agenda editor: %w", err)
					}
					// Type example agenda
					if err := page.Locator("#agenda-import-source").Fill("1. Opening\n2. Reports\n  2.1 Chair report\n  2.2 Treasurer report\n3. Motions\n4. Closing"); err != nil {
						return fmt.Errorf("fill agenda: %w", err)
					}
					page.WaitForTimeout(500)
					if err := highlight(page, "dialog .modal-box"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appScreenshotJoinQRDialogScript() Script {
	return Script{
		Name:        "app.screenshot-join-qr-dialog",
		Description: "Join QR code dialog with QR image and copy URL button",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotScenario(docscapture.ScreenshotScenarioOptions{
				Common:     common,
				Variant:    variant,
				OutputName: appOutputName("app-join-qr-dialog", variant, "png"),
				FullPage:   false,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := gotoPath(page, common.BaseURL, moderatePath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("#moderate-sse-root").WaitFor(); err != nil {
						return fmt.Errorf("wait moderate root: %w", err)
					}
					// Switch to attendees tab
					if err := page.Locator("[data-moderate-left-tab='attendees']").Click(); err != nil {
						return fmt.Errorf("click attendees tab: %w", err)
					}
					// Click show signup QR
					qrBtn := page.Locator("button[title='Show signup QR']")
					if err := qrBtn.WaitFor(); err != nil {
						return fmt.Errorf("wait QR button: %w", err)
					}
					if err := qrBtn.Click(); err != nil {
						return fmt.Errorf("click QR button: %w", err)
					}
					// Wait for QR image in dialog
					if err := page.Locator("#join-qr-dialog #join-qr-code").WaitFor(); err != nil {
						return fmt.Errorf("wait QR code image: %w", err)
					}
					if err := highlight(page, "#join-qr-dialog .modal-box"); err != nil {
						return err
					}
					return nil
				},
			})
		},
	}
}

func appGIFMemberJoinToLiveScript() Script {
	return Script{
		Name:        "app.gif-member-join-to-live",
		Description: "Member workflow from committee dashboard join action to live page",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			result, err := docscapture.RunGIFScenario(docscapture.GIFScenarioOptions{
				Common:       common,
				Variant:      variant,
				InitialPath:  "/",
				OutputName:   appOutputName("app-member-join-to-live", variant, "gif"),
				FinalPauseMS: 1400,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.JoinMemberUsername, fixture.JoinMemberPassword); err != nil {
						return err
					}
					if err := ensureGIFCursor(page); err != nil {
						return err
					}
					page.WaitForTimeout(gifPauseAfterMoveMS)
					if err := gotoPath(page, common.BaseURL, committeePath(fixture)); err != nil {
						return err
					}
					joinButton := page.Locator("[data-testid='committee-join-active-meeting']")
					if err := joinButton.WaitFor(); err != nil {
						return fmt.Errorf("wait member active meeting join button: %w", err)
					}
					if err := highlight(page, "[data-testid='committee-join-active-meeting']"); err != nil {
						return err
					}
					if err := clickLocatorWithCursor(page, joinButton, "member active meeting join button"); err != nil {
						return err
					}
					if err := page.WaitForURL(absoluteURL(common.BaseURL, joinPath(fixture))); err != nil {
						return fmt.Errorf("wait join page URL: %w", err)
					}
					page.WaitForTimeout(gifPauseAfterActionMS)
					submit := page.Locator("form[action$='/join'] button[type='submit']").First()
					if err := submit.WaitFor(); err != nil {
						return fmt.Errorf("wait join submit button: %w", err)
					}
					if err := highlight(page, "form[action$='/join'] button[type='submit']"); err != nil {
						return err
					}
					if err := clickLocatorWithCursor(page, submit, "join form submit button"); err != nil {
						return err
					}
					if err := page.WaitForURL(absoluteURL(common.BaseURL, livePath(fixture))); err != nil {
						return fmt.Errorf("wait live page URL after join: %w", err)
					}
					page.WaitForTimeout(gifPauseAfterActionMS)
					return nil
				},
			})
			if err != nil {
				return "", err
			}
			return result.GIFPath, nil
		},
	}
}

func appGIFSpeakerLifecycleModerateToLiveScript() Script {
	return Script{
		Name:        "app.gif-speaker-lifecycle-moderate-to-live",
		Description: "Chairperson starts next speaker in moderate view and transitions to live attendee view",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			result, err := docscapture.RunGIFScenario(docscapture.GIFScenarioOptions{
				Common:       common,
				Variant:      variant,
				InitialPath:  "/",
				OutputName:   appOutputName("app-speaker-lifecycle-moderate-to-live", variant, "gif"),
				FinalPauseMS: 1400,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := ensureGIFCursor(page); err != nil {
						return err
					}
					page.WaitForTimeout(gifPauseAfterMoveMS)
					if err := gotoPath(page, common.BaseURL, moderatePath(fixture)); err != nil {
						return err
					}
					startNext := page.Locator("[data-testid='manage-start-next-speaker']")
					if err := startNext.WaitFor(); err != nil {
						return fmt.Errorf("wait start-next-speaker button: %w", err)
					}
					if err := highlight(page, "[data-testid='manage-start-next-speaker']"); err != nil {
						return err
					}
					if err := clickLocatorWithCursor(page, startNext, "start-next-speaker button"); err != nil {
						return err
					}
					page.WaitForTimeout(gifPauseAfterActionMS)
					if err := gotoPath(page, common.BaseURL, livePath(fixture)); err != nil {
						return err
					}
					if err := page.Locator("#attendee-speakers-list").WaitFor(); err != nil {
						return fmt.Errorf("wait live speakers panel: %w", err)
					}
					page.WaitForTimeout(gifPauseAfterActionMS)
					return nil
				},
			})
			if err != nil {
				return "", err
			}
			return result.GIFPath, nil
		},
	}
}

func appGIFVoteLifecycleOpenAndSecretScript() Script {
	return Script{
		Name:        "app.gif-vote-lifecycle-open-and-secret",
		Description: "Moderate workflow creating and advancing an open and secret vote lifecycle",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			fixture, err := seedAppFixture(ctx, env.Seeder())
			if err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
			result, err := docscapture.RunGIFScenario(docscapture.GIFScenarioOptions{
				Common:       common,
				Variant:      variant,
				InitialPath:  "/",
				OutputName:   appOutputName("app-vote-lifecycle-open-and-secret", variant, "gif"),
				FinalPauseMS: 1400,
				Scenario: func(page playwright.Page) error {
					if err := loginUser(page, common.BaseURL, fixture.ChairUsername, fixture.ChairPassword); err != nil {
						return err
					}
					if err := ensureGIFCursor(page); err != nil {
						return err
					}
					page.WaitForTimeout(gifPauseAfterMoveMS)
					if err := gotoPath(page, common.BaseURL, moderatePath(fixture)); err != nil {
						return err
					}
					toolsTab := page.Locator("#moderate-left-controls [data-moderate-left-tab='tools']")
					if err := toolsTab.WaitFor(); err != nil {
						return fmt.Errorf("wait moderate tools tab: %w", err)
					}
					if err := clickLocatorWithCursor(page, toolsTab, "moderate tools tab"); err != nil {
						return err
					}
					page.WaitForTimeout(gifPauseAfterMoveMS)
					panel := page.Locator("#moderate-votes-panel")
					if err := panel.WaitFor(); err != nil {
						return fmt.Errorf("wait moderate votes panel: %w", err)
					}

					secretDetails := panel.Locator("details.collapse").Filter(playwright.LocatorFilterOptions{HasText: "Confidential Election"}).First()
					if err := secretDetails.WaitFor(); err != nil {
						return fmt.Errorf("wait secret vote accordion: %w", err)
					}
					if err := clickLocatorWithCursor(page, secretDetails.Locator("summary").First(), "secret vote accordion"); err != nil {
						return fmt.Errorf("open secret vote accordion: %w", err)
					}
					openButton := secretDetails.Locator("button:has-text('Open Vote')").First()
					if err := openButton.WaitFor(); err != nil {
						return fmt.Errorf("wait open vote button: %w", err)
					}
					if err := highlight(page, "button:has-text('Open Vote')"); err != nil {
						return err
					}
					if err := clickLocatorWithCursor(page, openButton, "open vote button"); err != nil {
						return fmt.Errorf("click open vote button: %w", err)
					}
					closeButton := secretDetails.Locator("button:has-text('Close Vote')").First()
					if err := closeButton.WaitFor(); err != nil {
						return fmt.Errorf("wait close vote button: %w", err)
					}
					if err := highlight(page, "button:has-text('Close Vote')"); err != nil {
						return err
					}
					if err := clickLocatorWithCursor(page, closeButton, "close vote button"); err != nil {
						return fmt.Errorf("click close vote button: %w", err)
					}

					openDetails := panel.Locator("details.collapse").Filter(playwright.LocatorFilterOptions{HasText: "Budget Approval"}).First()
					if err := openDetails.WaitFor(); err != nil {
						return fmt.Errorf("wait open vote accordion: %w", err)
					}
					if err := clickLocatorWithCursor(page, openDetails.Locator("summary").First(), "open vote accordion"); err != nil {
						return fmt.Errorf("open open-vote accordion: %w", err)
					}
					page.WaitForTimeout(gifPauseAfterActionMS)
					return nil
				},
			})
			if err != nil {
				return "", err
			}
			return result.GIFPath, nil
		},
	}
}

func seedAppFixture(ctx context.Context, seeder *docscapture.Seeder) (appFixture, error) {
	fixture := appFixture{
		CommitteeSlug:      "docs-committee",
		ChairUsername:      "chair-docs",
		ChairPassword:      "chair-docs-pass",
		MemberUsername:     "member-docs",
		MemberPassword:     "member-docs-pass",
		JoinMemberUsername: "member-join",
		JoinMemberPassword: "member-join-pass",
		GuestSecret:        "guest-docs-secret",
	}

	if err := seeder.CreateCommittee(ctx, "Documentation Committee", fixture.CommitteeSlug); err != nil {
		return appFixture{}, err
	}
	if err := seeder.CreateCommitteeUser(ctx, fixture.CommitteeSlug, fixture.ChairUsername, fixture.ChairPassword, "Chairperson Demo", true, "chairperson"); err != nil {
		return appFixture{}, err
	}
	if err := seeder.CreateCommitteeUser(ctx, fixture.CommitteeSlug, fixture.MemberUsername, fixture.MemberPassword, "Member Demo", false, "member"); err != nil {
		return appFixture{}, err
	}
	if err := seeder.CreateCommitteeUser(ctx, fixture.CommitteeSlug, fixture.JoinMemberUsername, fixture.JoinMemberPassword, "Join Member Demo", false, "member"); err != nil {
		return appFixture{}, err
	}

	const meetingName = "General Assembly"
	if err := seeder.CreateMeeting(ctx, fixture.CommitteeSlug, meetingName, "Quarterly planning and votes", "docs-meeting-secret", true); err != nil {
		return appFixture{}, err
	}
	meeting, err := seeder.SetActiveMeetingByName(ctx, fixture.CommitteeSlug, meetingName)
	if err != nil {
		return appFixture{}, err
	}
	fixture.MeetingID = meeting.ID

	currentAgenda, err := seeder.CreateAgendaPoint(ctx, fixture.CommitteeSlug, meetingName, "Budget Planning")
	if err != nil {
		return appFixture{}, err
	}
	fixture.AgendaPointID = currentAgenda.ID
	if _, err := seeder.CreateAgendaPoint(ctx, fixture.CommitteeSlug, meetingName, "Elections"); err != nil {
		return appFixture{}, err
	}
	if err := seeder.SetCurrentAgendaPointByName(ctx, fixture.CommitteeSlug, meetingName, currentAgenda.ID); err != nil {
		return appFixture{}, err
	}

	label := "Budget Draft PDF"
	attachment, err := seeder.CreateAttachment(ctx, currentAgenda.ID, "budget-draft.txt", "text/plain", "budget draft document for docs capture", &label)
	if err != nil {
		return appFixture{}, err
	}
	if err := seeder.SetCurrentAttachment(ctx, currentAgenda.ID, attachment.ID); err != nil {
		return appFixture{}, err
	}

	chairAttendee, err := seeder.CreateUserAttendee(ctx, fixture.CommitteeSlug, meetingName, fixture.ChairUsername)
	if err != nil {
		return appFixture{}, err
	}
	if err := seeder.SetAttendeeChair(ctx, chairAttendee.ID, true); err != nil {
		return appFixture{}, err
	}
	memberAttendee, err := seeder.CreateUserAttendee(ctx, fixture.CommitteeSlug, meetingName, fixture.MemberUsername)
	if err != nil {
		return appFixture{}, err
	}
	guestAttendee, err := seeder.CreateGuestAttendee(ctx, fixture.CommitteeSlug, meetingName, "Guest Participant", fixture.GuestSecret, true)
	if err != nil {
		return appFixture{}, err
	}

	memberSpeaker, err := seeder.AddSpeaker(ctx, currentAgenda.ID, memberAttendee.ID, "regular", false, true)
	if err != nil {
		return appFixture{}, err
	}
	if _, err := seeder.AddSpeaker(ctx, currentAgenda.ID, guestAttendee.ID, "ropm", true, false); err != nil {
		return appFixture{}, err
	}
	_ = memberSpeaker

	openVote, err := seeder.CreateVoteDefinition(
		ctx,
		meeting.ID,
		currentAgenda.ID,
		"Budget Approval",
		model.VoteVisibilityOpen,
		1,
		1,
		[]repository.VoteOptionInput{
			{Label: "Yes", Position: 1},
			{Label: "No", Position: 2},
		},
	)
	if err != nil {
		return appFixture{}, err
	}
	if _, err := seeder.OpenVoteForAttendees(ctx, openVote.ID, []int64{chairAttendee.ID, memberAttendee.ID, guestAttendee.ID}); err != nil {
		return appFixture{}, err
	}
	if _, err := seeder.CreateVoteDefinition(
		ctx,
		meeting.ID,
		currentAgenda.ID,
		"Confidential Election",
		model.VoteVisibilitySecret,
		1,
		1,
		[]repository.VoteOptionInput{
			{Label: "Candidate A", Position: 1},
			{Label: "Candidate B", Position: 2},
		},
	); err != nil {
		return appFixture{}, err
	}

	return fixture, nil
}

func loginUser(page playwright.Page, baseURL, username, password string) error {
	if err := gotoPath(page, baseURL, "/"); err != nil {
		return err
	}
	if err := page.Locator("input[name='username']").Fill(username); err != nil {
		return fmt.Errorf("fill username: %w", err)
	}
	if err := page.Locator("input[name='password']").Fill(password); err != nil {
		return fmt.Errorf("fill password: %w", err)
	}
	if err := page.Locator("input[name='password']").Press("Enter"); err != nil {
		return fmt.Errorf("submit login form: %w", err)
	}
	if err := page.WaitForURL(absoluteURL(baseURL, "/home")); err != nil {
		return fmt.Errorf("wait /home after login: %w", err)
	}
	return nil
}

func loginAttendee(page playwright.Page, baseURL, loginPath, livePath, secret string) error {
	if err := gotoPath(page, baseURL, loginPath); err != nil {
		return err
	}
	if err := page.Locator("input[name='secret']").Fill(secret); err != nil {
		return fmt.Errorf("fill attendee secret: %w", err)
	}
	if err := page.Locator("input[name='secret']").Press("Enter"); err != nil {
		return fmt.Errorf("submit attendee login: %w", err)
	}
	if err := page.WaitForURL(absoluteURL(baseURL, livePath)); err != nil {
		return fmt.Errorf("wait live attendee URL: %w", err)
	}
	return nil
}

func gotoPath(page playwright.Page, baseURL, p string) error {
	target := absoluteURL(baseURL, p)
	if _, err := page.Goto(target, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateLoad}); err != nil {
		return fmt.Errorf("navigate to %q: %w", target, err)
	}
	if err := page.Locator("body").WaitFor(); err != nil {
		return fmt.Errorf("wait for body on %q: %w", target, err)
	}
	return nil
}

func appOutputName(base string, variant docscapture.Variant, extension string) string {
	return fmt.Sprintf("%s.%s.%s", base, variantSuffix(variant), extension)
}

func variantSuffix(variant docscapture.Variant) string {
	parts := []string{string(variant.Language), string(variant.Theme), string(variant.Device)}
	return strings.Join(parts, ".")
}

func committeePath(f appFixture) string {
	return fmt.Sprintf("/committee/%s", f.CommitteeSlug)
}

func joinPath(f appFixture) string {
	return fmt.Sprintf("/committee/%s/meeting/%d/join", f.CommitteeSlug, f.MeetingID)
}

func moderatePath(f appFixture) string {
	return fmt.Sprintf("/committee/%s/meeting/%d/moderate", f.CommitteeSlug, f.MeetingID)
}

func attendeeLoginPath(f appFixture) string {
	return fmt.Sprintf("/committee/%s/meeting/%d/attendee-login", f.CommitteeSlug, f.MeetingID)
}

func livePath(f appFixture) string {
	return fmt.Sprintf("/committee/%s/meeting/%d", f.CommitteeSlug, f.MeetingID)
}

func agendaToolsPath(f appFixture) string {
	return fmt.Sprintf("/committee/%s/meeting/%d/agenda-point/%d/tools", f.CommitteeSlug, f.MeetingID, f.AgendaPointID)
}

func absoluteURL(baseURL, p string) string {
	base := strings.TrimRight(baseURL, "/")
	if p == "" {
		return base + "/"
	}
	if strings.HasPrefix(p, "/") {
		return base + p
	}
	return base + "/" + p
}

func highlight(page playwright.Page, selector string) error {
	return highlightMany(page, selector)
}

func highlightFirstAvailable(page playwright.Page, selectors ...string) error {
	var lastErr error
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		if err := highlight(page, selector); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	_ = lastErr
	return nil
}

func highlightMany(page playwright.Page, selectors ...string) error {
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		locator := page.Locator(selector).First()
		if err := locator.WaitFor(); err != nil {
			return fmt.Errorf("wait highlight target %q: %w", selector, err)
		}
		ok, err := locator.Evaluate(`(target) => {
			const styleId = "docs-capture-highlight-style";
			if (!document.getElementById(styleId)) {
				const style = document.createElement("style");
				style.id = styleId;
				style.textContent = '[data-docs-capture-highlight="1"]{' +
					'outline:3px solid #ef4444 !important;' +
					'outline-offset:2px !important;' +
					'box-shadow:0 0 0 5px rgba(239,68,68,0.22) !important;' +
					'border-radius:0.55rem !important;' +
				'}';
				document.head.appendChild(style);
			}
			if (!target) {
				return false;
			}
			target.setAttribute("data-docs-capture-highlight", "1");
			if (typeof target.scrollIntoView === "function") {
				target.scrollIntoView({ block: "center", inline: "nearest" });
			}
			return true;
		}`, nil)
		if err != nil {
			return fmt.Errorf("apply highlight for %q: %w", selector, err)
		}
		applied, _ := ok.(bool)
		if !applied {
			return fmt.Errorf("highlight target not found for selector %q", selector)
		}
	}
	return nil
}

func ensureGIFCursor(page playwright.Page) error {
	_, err := page.Evaluate(`() => {
		const styleId = "docs-capture-gif-cursor-style";
		if (!document.getElementById(styleId)) {
			const style = document.createElement("style");
			style.id = styleId;
			style.textContent =
				"* { cursor: none !important; }" +
				"#docs-capture-gif-cursor {" +
				"  position: fixed;" +
				"  left: 0;" +
				"  top: 0;" +
				"  width: 18px;" +
				"  height: 18px;" +
				"  border: 2px solid #ef4444;" +
				"  border-radius: 999px;" +
				"  background: rgba(239,68,68,0.18);" +
				"  box-shadow: 0 0 0 4px rgba(239,68,68,0.16);" +
				"  pointer-events: none;" +
				"  z-index: 2147483647;" +
				"  transform: translate(24px, 24px);" +
				"  transition: transform 70ms linear;" +
				"}";
			document.head.appendChild(style);
		}
		let cursor = document.getElementById("docs-capture-gif-cursor");
		if (!cursor) {
			cursor = document.createElement("div");
			cursor.id = "docs-capture-gif-cursor";
			document.body.appendChild(cursor);
		}
	}`, nil)
	if err != nil {
		return fmt.Errorf("inject gif cursor overlay: %w", err)
	}
	if err := page.Mouse().Move(24, 24); err != nil {
		return fmt.Errorf("position gif cursor: %w", err)
	}
	return nil
}

func clickLocatorWithCursor(page playwright.Page, locator playwright.Locator, target string) error {
	if err := locator.WaitFor(); err != nil {
		return fmt.Errorf("wait %s: %w", target, err)
	}
	if err := ensureGIFCursor(page); err != nil {
		return err
	}
	if err := locator.ScrollIntoViewIfNeeded(); err != nil {
		return fmt.Errorf("scroll %s into view: %w", target, err)
	}
	box, err := locator.BoundingBox()
	if err != nil {
		return fmt.Errorf("measure %s bounding box: %w", target, err)
	}
	if box == nil {
		return fmt.Errorf("measure %s bounding box: no box", target)
	}
	x := box.X + (box.Width / 2)
	y := box.Y + (box.Height / 2)
	if err := page.Mouse().Move(x, y, playwright.MouseMoveOptions{Steps: playwright.Int(gifCursorMoveSteps)}); err != nil {
		return fmt.Errorf("move cursor to %s: %w", target, err)
	}
	page.WaitForTimeout(gifPauseAfterMoveMS)
	if err := locator.Click(playwright.LocatorClickOptions{Delay: playwright.Float(gifClickDelayMS)}); err != nil {
		return fmt.Errorf("click %s: %w", target, err)
	}
	page.WaitForTimeout(gifPauseAfterClickMS)
	return nil
}
