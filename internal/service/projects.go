package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Omotolani98/foostash/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Projects struct {
	repos *repo.Repos
	envs  *Environments
}

type Environments struct {
	repos *repo.Repos
}

func NewProjects(r *repo.Repos) *Projects {
	return &Projects{repos: r, envs: &Environments{repos: r}}
}

func (p *Projects) Envs() *Environments { return p.envs }

// ProjectView is what handlers return for a single project.
type ProjectView struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	CreatedAt    string
	UpdatedAt    string
	Environments []EnvView
}

// ProjectListEntry is what handlers return for the list endpoint.
type ProjectListEntry struct {
	Slug      string
	Name      string
	EnvCount  int
	UpdatedAt string
}

type EnvView struct {
	Slug      string
	CreatedAt string
}

// Create creates a project under the caller's org. A default "dev" env is
// created in the same transaction so that newly-initialized projects have
// somewhere to put secrets.
func (p *Projects) Create(ctx context.Context, actx *AuthContext, name string) (*repo.Project, error) {
	if !isAdmin(actx) {
		return nil, ErrForbidden
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidArgument
	}
	slug := Slugify(name)
	if slug == "" {
		return nil, ErrInvalidArgument
	}

	if existing, err := p.repos.Projects.GetBySlug(ctx, actx.OrgID, slug); err == nil && existing != nil {
		return nil, ErrProjectExists
	} else if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}

	var out *repo.Project
	err := repo.InTx(ctx, p.repos.Pool, func(tx pgx.Tx) error {
		proj, err := p.repos.Projects.Insert(ctx, tx, actx.OrgID, slug, name, actx.UserID)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrProjectExists
			}
			return err
		}
		if _, err := p.repos.Environments.Insert(ctx, tx, proj.ID, "dev"); err != nil {
			return err
		}
		out = proj
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (p *Projects) List(ctx context.Context, actx *AuthContext) ([]ProjectListEntry, error) {
	items, err := p.repos.Projects.List(ctx, actx.OrgID)
	if err != nil {
		return nil, err
	}
	out := make([]ProjectListEntry, 0, len(items))
	for _, it := range items {
		out = append(out, ProjectListEntry{
			Slug:      it.Slug,
			Name:      it.Name,
			EnvCount:  it.EnvCount,
			UpdatedAt: it.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out, nil
}

func (p *Projects) Get(ctx context.Context, actx *AuthContext, slug string) (*ProjectView, error) {
	proj, err := p.requireProject(ctx, actx, slug)
	if err != nil {
		return nil, err
	}
	envs, err := p.repos.Environments.ListByProject(ctx, proj.ID)
	if err != nil {
		return nil, err
	}
	view := &ProjectView{
		ID:        proj.ID,
		Slug:      proj.Slug,
		Name:      proj.Name,
		CreatedAt: proj.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: proj.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	for _, e := range envs {
		view.Environments = append(view.Environments, EnvView{
			Slug:      e.Slug,
			CreatedAt: e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return view, nil
}

func (p *Projects) Delete(ctx context.Context, actx *AuthContext, slug string) error {
	if !isAdmin(actx) {
		return ErrForbidden
	}
	if _, err := p.requireProject(ctx, actx, slug); err != nil {
		return err
	}
	if err := p.repos.Projects.Delete(ctx, actx.OrgID, slug); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrProjectNotFound
		}
		return err
	}
	return nil
}

// requireProject returns the project iff it exists in the caller's org.
// Returns ErrProjectNotFound for both "no row" and "wrong org" — never leak.
func (p *Projects) requireProject(ctx context.Context, actx *AuthContext, slug string) (*repo.Project, error) {
	proj, err := p.repos.Projects.GetBySlug(ctx, actx.OrgID, slug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return proj, nil
}

// --- Environments ---

func (e *Environments) Create(ctx context.Context, actx *AuthContext, projectSlug, envSlug string) (*repo.Environment, error) {
	if !isAdmin(actx) {
		return nil, ErrForbidden
	}
	envSlug = Slugify(envSlug)
	if envSlug == "" {
		return nil, ErrInvalidArgument
	}
	proj, err := (&Projects{repos: e.repos}).requireProject(ctx, actx, projectSlug)
	if err != nil {
		return nil, err
	}
	env, err := e.repos.Environments.Insert(ctx, e.repos.Pool, proj.ID, envSlug)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEnvExists
		}
		return nil, err
	}
	return env, nil
}

func (e *Environments) List(ctx context.Context, actx *AuthContext, projectSlug string) ([]EnvView, error) {
	proj, err := (&Projects{repos: e.repos}).requireProject(ctx, actx, projectSlug)
	if err != nil {
		return nil, err
	}
	envs, err := e.repos.Environments.ListByProject(ctx, proj.ID)
	if err != nil {
		return nil, err
	}
	out := make([]EnvView, 0, len(envs))
	for _, env := range envs {
		out = append(out, EnvView{
			Slug:      env.Slug,
			CreatedAt: env.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out, nil
}

// Clone copies an env's metadata to a new env. Secrets are not yet stored
// server-side, so this is just an empty env created with the dest slug.
func (e *Environments) Clone(ctx context.Context, actx *AuthContext, projectSlug, srcSlug, destSlug string) (*repo.Environment, error) {
	if !isAdmin(actx) {
		return nil, ErrForbidden
	}
	destSlug = Slugify(destSlug)
	if destSlug == "" {
		return nil, ErrInvalidArgument
	}
	proj, err := (&Projects{repos: e.repos}).requireProject(ctx, actx, projectSlug)
	if err != nil {
		return nil, err
	}

	var out *repo.Environment
	err = repo.InTx(ctx, e.repos.Pool, func(tx pgx.Tx) error {
		if _, err := e.repos.Environments.GetBySlug(ctx, tx, proj.ID, srcSlug); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				return ErrEnvNotFound
			}
			return err
		}
		env, err := e.repos.Environments.Insert(ctx, tx, proj.ID, destSlug)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrEnvExists
			}
			return err
		}
		out = env
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (e *Environments) Delete(ctx context.Context, actx *AuthContext, projectSlug, envSlug string) error {
	if !isAdmin(actx) {
		return ErrForbidden
	}
	proj, err := (&Projects{repos: e.repos}).requireProject(ctx, actx, projectSlug)
	if err != nil {
		return err
	}
	if err := e.repos.Environments.Delete(ctx, proj.ID, envSlug); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrEnvNotFound
		}
		return err
	}
	return nil
}

func isAdmin(actx *AuthContext) bool {
	return actx != nil && actx.Role == "admin"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
