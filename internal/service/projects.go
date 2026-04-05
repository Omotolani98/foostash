package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/Omotolani98/foostash/internal/store"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

var planLimits = map[string]struct {
	MaxProjects     int
	MaxEnvs         int
	MaxSecretsPerEnv int
}{
	"free":  {MaxProjects: 3, MaxEnvs: 3, MaxSecretsPerEnv: 50},
	"team":  {MaxProjects: 0, MaxEnvs: 0, MaxSecretsPerEnv: 0},
	"cloud": {MaxProjects: 0, MaxEnvs: 0, MaxSecretsPerEnv: 0},
}

type ProjectsService struct {
	projects *store.ProjectsStore
	envs     *store.EnvironmentsStore
	orgs     *store.OrganizationsStore
}

func NewProjectsService(p *store.ProjectsStore, e *store.EnvironmentsStore, o *store.OrganizationsStore) *ProjectsService {
	return &ProjectsService{projects: p, envs: e, orgs: o}
}

func ValidateSlug(slug string) error {
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("%w: slug must be lowercase alphanumeric with hyphens", ErrValidation)
	}
	return nil
}

func (s *ProjectsService) Create(ctx context.Context, orgID, slug, name string) (*store.ProjectRow, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}
	if name == "" {
		name = slug
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	limits := planLimits[org.Plan]
	if limits.MaxProjects > 0 {
		count, err := s.projects.CountByOrg(ctx, orgID)
		if err != nil {
			return nil, err
		}
		if count >= limits.MaxProjects {
			return nil, ErrPlanLimitReached
		}
	}

	proj, err := s.projects.Create(ctx, orgID, slug, name)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return proj, nil
}

func (s *ProjectsService) Get(ctx context.Context, orgID, slug string) (*store.ProjectRow, error) {
	p, err := s.projects.GetBySlug(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *ProjectsService) List(ctx context.Context, orgID string) ([]store.ProjectRow, error) {
	return s.projects.ListByOrg(ctx, orgID)
}

func (s *ProjectsService) Delete(ctx context.Context, orgID, slug string) error {
	if err := s.projects.Delete(ctx, orgID, slug); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *ProjectsService) CreateEnv(ctx context.Context, orgID, projectSlug, envSlug string) (*store.EnvironmentRow, error) {
	if err := ValidateSlug(envSlug); err != nil {
		return nil, err
	}
	proj, err := s.Get(ctx, orgID, projectSlug)
	if err != nil {
		return nil, err
	}
	env, err := s.envs.Create(ctx, proj.ID, envSlug)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return env, nil
}

func (s *ProjectsService) ListEnvs(ctx context.Context, orgID, projectSlug string) ([]store.EnvironmentRow, error) {
	proj, err := s.Get(ctx, orgID, projectSlug)
	if err != nil {
		return nil, err
	}
	return s.envs.ListByProject(ctx, proj.ID)
}

func (s *ProjectsService) DeleteEnv(ctx context.Context, orgID, projectSlug, envSlug string) error {
	proj, err := s.Get(ctx, orgID, projectSlug)
	if err != nil {
		return err
	}
	if err := s.envs.Delete(ctx, proj.ID, envSlug); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
