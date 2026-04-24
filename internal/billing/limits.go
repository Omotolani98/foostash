package billing

// PlanLimits encodes the published tiers from ARCHITECTURE.md §13. Keep in
// sync with the public pricing page (foostash-web/components/Pricing.tsx).
var PlanLimits = map[string]Limits{
	PlanSelfHosted: Unlimited(),
	PlanFree: {
		Projects: 2, EnvsPerProject: 3, SecretsPerEnv: 30, Users: 2,
		VaultSecrets: 5, AuditRetentionDays: 7,
	},
	PlanPro: {
		Projects: -1, EnvsPerProject: -1, SecretsPerEnv: -1, Users: 10,
		VaultSecrets: -1, AuditRetentionDays: 90,
	},
	PlanTeam: {
		Projects: -1, EnvsPerProject: -1, SecretsPerEnv: -1, Users: -1,
		VaultSecrets: -1, AuditRetentionDays: 365,
	},
	PlanLifetime: {
		Projects: -1, EnvsPerProject: -1, SecretsPerEnv: -1, Users: 10,
		VaultSecrets: -1, AuditRetentionDays: 90,
	},
}

// ResourceKind names the resource being checked for quota. Callers pass one of
// these to Enforcer.Check.
type ResourceKind string

const (
	ResourceProjects     ResourceKind = "projects"
	ResourceEnvsInProject ResourceKind = "envs_in_project"
	ResourceSecretsInEnv ResourceKind = "secrets_in_env"
	ResourceUsers        ResourceKind = "users"
	ResourceVaultSecrets ResourceKind = "vault_secrets"
)

// LimitFor returns the numeric cap for a resource on the given plan, or -1 if
// unlimited. Unknown plans are treated as free (defensive: rather deny than
// unexpectedly grant unlimited on a typo).
func LimitFor(plan string, kind ResourceKind) int {
	l, ok := PlanLimits[plan]
	if !ok {
		l = PlanLimits[PlanFree]
	}
	switch kind {
	case ResourceProjects:
		return l.Projects
	case ResourceEnvsInProject:
		return l.EnvsPerProject
	case ResourceSecretsInEnv:
		return l.SecretsPerEnv
	case ResourceUsers:
		return l.Users
	case ResourceVaultSecrets:
		return l.VaultSecrets
	default:
		return -1
	}
}

// ErrLimitExceeded is returned by Enforcer.Check when the caller would exceed
// their plan's quota. Handlers should translate this to HTTP 402.
type LimitExceededError struct {
	Plan    string
	Kind    ResourceKind
	Current int
	Limit   int
}

func (e *LimitExceededError) Error() string {
	return "plan limit exceeded"
}
