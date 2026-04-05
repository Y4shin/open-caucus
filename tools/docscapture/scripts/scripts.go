package scripts

import (
	"context"
	"fmt"
	"path"
	"slices"

	"github.com/Y4shin/open-caucus/tools/docscapture"
)

type Script struct {
	Name        string
	Description string
	Environment EnvironmentOptions
	Run         func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error)
}

type EnvironmentOptions = docscapture.EnvironmentOptions

func All() []Script {
	return []Script{
		appScreenshotHomeCommitteesScript(),
		appScreenshotCommitteeDashboardChairScript(),
		appScreenshotCommitteeDashboardMemberActiveScript(),
		appScreenshotModerateOverviewScript(),
		appScreenshotAgendaToolsAttachmentsScript(),
		appScreenshotLiveViewWithSpeakersScript(),
		appScreenshotJoinPageMemberScript(),
		appScreenshotGuestSignupFormScript(),
		appScreenshotAttendeeLoginScript(),
		appScreenshotReceiptsVaultScript(),
		appScreenshotAdminLoginScript(),
		appScreenshotAdminDashboardScript(),
		appScreenshotMeetingWizardScript(),
		appScreenshotJoinQRDialogScript(),
		appGIFMemberJoinToLiveScript(),
		appGIFSpeakerLifecycleModerateToLiveScript(),
		appGIFVoteLifecycleOpenAndSecretScript(),
	}
}

func Match(globs []string) ([]Script, error) {
	if len(globs) == 0 {
		globs = []string{"*"}
	}

	matches := make([]Script, 0)
	seen := make(map[string]bool)
	for _, script := range All() {
		for _, pattern := range globs {
			ok, err := path.Match(pattern, script.Name)
			if err != nil {
				return nil, fmt.Errorf("invalid script glob %q: %w", pattern, err)
			}
			if ok {
				if !seen[script.Name] {
					matches = append(matches, script)
					seen[script.Name] = true
				}
				break
			}
		}
	}

	slices.SortFunc(matches, func(a, b Script) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		default:
			return 0
		}
	})
	return matches, nil
}
