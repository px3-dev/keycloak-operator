package chart

// Upstream holds all data extracted from the upstream Keycloak operator manifests.
type Upstream struct {
	AppVersion    string
	OperatorImage string
	KeycloakImage string
	Deployment    DeploymentData
	Service       ServiceData
	RBAC          RBACData
}

type DeploymentData struct {
	Replicas      int
	ContainerName string
	ContainerPort int
	Resources     ResourceRequirements
	Probes        ProbeConfig
	ExtraEnv      []StaticEnvVar
}

type ResourceRequirements struct {
	Requests ResourceList
	Limits   ResourceList
}

type ResourceList struct {
	CPU    string
	Memory string
}

type ProbeConfig struct {
	Liveness  ProbeSpec
	Readiness ProbeSpec
	Startup   ProbeSpec
}

type ProbeSpec struct {
	Path                string
	FailureThreshold    int
	InitialDelaySeconds int
	PeriodSeconds       int
	SuccessThreshold    int
	TimeoutSeconds      int
}

type StaticEnvVar struct {
	Name  string
	Value string
}

type ServiceData struct {
	Type string
	Port int
}

type RBACData struct {
	ClusterRoles        []RBACRole
	ClusterRoleBindings []RBACBinding
	Roles               []RBACRole
	RoleBindings        []RBACBinding
}

type RBACRole struct {
	OriginalName string
	Suffix       string
	RulesYAML    string
}

type RBACBinding struct {
	OriginalName  string
	Suffix        string
	RoleRefKind   string
	RoleRefName   string
	RoleSuffix    string
	IsBuiltinRole bool
}
