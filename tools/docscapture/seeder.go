package docscapture

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/storage"
)

// Seeder provides deterministic fixture helpers for docs-capture scripts.
type Seeder struct {
	repo    repository.Repository
	storage storage.Service
}

func NewSeeder(repo repository.Repository, storage storage.Service) *Seeder {
	return &Seeder{
		repo:    repo,
		storage: storage,
	}
}

func (s *Seeder) Storage() storage.Service {
	return s.storage
}

func (s *Seeder) CreateAccount(ctx context.Context, username, password, fullName string) (*model.Account, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	if fullName == "" {
		fullName = username
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return nil, fmt.Errorf("hash password for account %q: %w", username, err)
	}

	account, err := s.repo.CreateAccount(ctx, username, fullName, string(hash))
	if err != nil {
		return nil, fmt.Errorf("create account %q: %w", username, err)
	}
	return account, nil
}

func (s *Seeder) CreateAdminAccount(ctx context.Context, username, password, fullName string) (*model.Account, error) {
	account, err := s.CreateAccount(ctx, username, password, fullName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetAccountIsAdmin(ctx, account.ID, true); err != nil {
		return nil, fmt.Errorf("set account %q as admin: %w", username, err)
	}
	return account, nil
}

func (s *Seeder) CreateOAuthAccount(ctx context.Context, username, fullName string, admin bool) (*model.Account, error) {
	account, err := s.repo.CreateOAuthAccount(ctx, username, fullName)
	if err != nil {
		return nil, fmt.Errorf("create oauth account %q: %w", username, err)
	}
	if admin {
		if err := s.repo.SetAccountIsAdmin(ctx, account.ID, true); err != nil {
			return nil, fmt.Errorf("set oauth account %q as admin: %w", username, err)
		}
	}
	return account, nil
}

func (s *Seeder) CreateCommittee(ctx context.Context, name, slug string) error {
	if err := s.repo.CreateCommitteeWithSlug(ctx, name, slug); err != nil {
		return fmt.Errorf("create committee %q: %w", slug, err)
	}
	return nil
}

func (s *Seeder) CreateCommitteeUser(
	ctx context.Context,
	committeeSlug, username, password, fullName string,
	quoted bool,
	role string,
) error {
	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, committeeSlug)
	if err != nil {
		return fmt.Errorf("lookup committee %q: %w", committeeSlug, err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("hash password for user %q: %w", username, err)
	}
	if err := s.repo.CreateUser(ctx, committeeID, username, string(hash), fullName, quoted, role); err != nil {
		return fmt.Errorf("create committee user %q in %q: %w", username, committeeSlug, err)
	}
	return nil
}

func (s *Seeder) CreateMeeting(ctx context.Context, committeeSlug, name, description, secret string, signupOpen bool) error {
	committeeID, err := s.repo.GetCommitteeIDBySlug(ctx, committeeSlug)
	if err != nil {
		return fmt.Errorf("lookup committee %q: %w", committeeSlug, err)
	}
	if secret == "" {
		secret = "docs-capture-meeting-secret"
	}
	if err := s.repo.CreateMeeting(ctx, committeeID, name, description, secret, signupOpen); err != nil {
		return fmt.Errorf("create meeting %q in %q: %w", name, committeeSlug, err)
	}
	return nil
}

func (s *Seeder) CreateOAuthCommitteeRule(ctx context.Context, committeeSlug, groupName, role string) (*model.OAuthCommitteeGroupRule, error) {
	rule, err := s.repo.CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx, committeeSlug, groupName, role)
	if err != nil {
		return nil, fmt.Errorf("create oauth committee rule for %q: %w", committeeSlug, err)
	}
	return rule, nil
}
