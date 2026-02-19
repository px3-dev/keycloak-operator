package chart

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type rawResource struct {
	Kind string
	Name string
	Raw  map[string]interface{}
}

// Parse reads a multi-document YAML manifest and extracts chart data.
func Parse(data []byte) (*Upstream, error) {
	resources, err := parseDocuments(data)
	if err != nil {
		return nil, err
	}
	return buildUpstream(resources)
}

func parseDocuments(data []byte) ([]rawResource, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var resources []rawResource
	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding YAML: %w", err)
		}
		if raw == nil {
			continue
		}

		kind, _ := raw["kind"].(string)
		metadata, _ := raw["metadata"].(map[string]interface{})
		name, _ := metadata["name"].(string)

		resources = append(resources, rawResource{Kind: kind, Name: name, Raw: raw})
	}
	return resources, nil
}

func buildUpstream(resources []rawResource) (*Upstream, error) {
	u := &Upstream{}
	managedRoles := make(map[string]string) // original name → suffix

	// First pass: collect roles, deployment, service
	for _, r := range resources {
		switch r.Kind {
		case "ClusterRole":
			role, err := parseRBACRole(r)
			if err != nil {
				return nil, fmt.Errorf("parsing ClusterRole %q: %w", r.Name, err)
			}
			u.RBAC.ClusterRoles = append(u.RBAC.ClusterRoles, role)
			managedRoles[r.Name] = role.Suffix

		case "Role":
			role, err := parseRBACRole(r)
			if err != nil {
				return nil, fmt.Errorf("parsing Role %q: %w", r.Name, err)
			}
			u.RBAC.Roles = append(u.RBAC.Roles, role)
			managedRoles[r.Name] = role.Suffix

		case "Deployment":
			if err := u.parseDeployment(r); err != nil {
				return nil, fmt.Errorf("parsing Deployment %q: %w", r.Name, err)
			}

		case "Service":
			u.parseService(r)
		}
	}

	// Second pass: process bindings (roles must be known first)
	for _, r := range resources {
		switch r.Kind {
		case "ClusterRoleBinding":
			u.RBAC.ClusterRoleBindings = append(u.RBAC.ClusterRoleBindings, parseRBACBinding(r, managedRoles))
		case "RoleBinding":
			u.RBAC.RoleBindings = append(u.RBAC.RoleBindings, parseRBACBinding(r, managedRoles))
		}
	}

	if u.AppVersion == "" {
		return nil, fmt.Errorf("no Deployment found or image tag missing")
	}

	return u, nil
}

// deriveSuffix maps upstream resource names to short Helm template suffixes.
func deriveSuffix(name string) string {
	replacements := []struct{ prefix, replacement string }{
		{"keycloak-operator-", ""},
		{"keycloakrealmimportcontroller-", "realmimport-"},
		{"keycloakcontroller-", "keycloak-"},
	}
	for _, r := range replacements {
		if strings.HasPrefix(name, r.prefix) {
			suffix := r.replacement + strings.TrimPrefix(name, r.prefix)
			return strings.TrimPrefix(suffix, "-")
		}
	}
	return name
}

func parseRBACRole(r rawResource) (RBACRole, error) {
	rules, ok := r.Raw["rules"]
	if !ok {
		return RBACRole{}, fmt.Errorf("no rules found")
	}

	rulesYAML, err := marshalYAML(rules)
	if err != nil {
		return RBACRole{}, fmt.Errorf("marshaling rules: %w", err)
	}

	return RBACRole{
		OriginalName: r.Name,
		Suffix:       deriveSuffix(r.Name),
		RulesYAML:    rulesYAML,
	}, nil
}

func parseRBACBinding(r rawResource, managedRoles map[string]string) RBACBinding {
	roleRef, _ := r.Raw["roleRef"].(map[string]interface{})
	roleRefKind, _ := roleRef["kind"].(string)
	roleRefName, _ := roleRef["name"].(string)

	roleSuffix, isManaged := managedRoles[roleRefName]

	return RBACBinding{
		OriginalName:  r.Name,
		Suffix:        deriveSuffix(r.Name),
		RoleRefKind:   roleRefKind,
		RoleRefName:   roleRefName,
		RoleSuffix:    roleSuffix,
		IsBuiltinRole: !isManaged,
	}
}

func (u *Upstream) parseDeployment(r rawResource) error {
	spec, _ := r.Raw["spec"].(map[string]interface{})
	u.Deployment.Replicas = intFromMap(spec, "replicas")

	tmpl, _ := spec["template"].(map[string]interface{})
	podSpec, _ := tmpl["spec"].(map[string]interface{})
	containers, _ := podSpec["containers"].([]interface{})
	if len(containers) == 0 {
		return fmt.Errorf("no containers found")
	}

	container, _ := containers[0].(map[string]interface{})

	// Image
	image, _ := container["image"].(string)
	u.OperatorImage, u.AppVersion = splitImage(image)

	// Container metadata
	u.Deployment.ContainerName, _ = container["name"].(string)

	// Ports
	ports, _ := container["ports"].([]interface{})
	if len(ports) > 0 {
		port, _ := ports[0].(map[string]interface{})
		u.Deployment.ContainerPort = intFromMap(port, "containerPort")
	}

	// Resources
	if resources, ok := container["resources"].(map[string]interface{}); ok {
		u.Deployment.Resources = parseResources(resources)
	}

	// Probes
	u.Deployment.Probes.Liveness = parseProbe(container, "livenessProbe")
	u.Deployment.Probes.Readiness = parseProbe(container, "readinessProbe")
	u.Deployment.Probes.Startup = parseProbe(container, "startupProbe")

	// Env vars
	envList, _ := container["env"].([]interface{})
	for _, e := range envList {
		env, _ := e.(map[string]interface{})
		name, _ := env["name"].(string)

		switch name {
		case "KUBERNETES_NAMESPACE":
			continue // handled by downward API in template
		case "RELATED_IMAGE_KEYCLOAK":
			value, _ := env["value"].(string)
			u.KeycloakImage, _ = splitImage(value)
		default:
			value, _ := env["value"].(string)
			u.Deployment.ExtraEnv = append(u.Deployment.ExtraEnv, StaticEnvVar{
				Name:  name,
				Value: value,
			})
		}
	}

	return nil
}

func (u *Upstream) parseService(r rawResource) {
	spec, _ := r.Raw["spec"].(map[string]interface{})
	u.Service.Type, _ = spec["type"].(string)

	ports, _ := spec["ports"].([]interface{})
	if len(ports) > 0 {
		port, _ := ports[0].(map[string]interface{})
		u.Service.Port = intFromMap(port, "port")
	}
}

func splitImage(ref string) (repo, tag string) {
	if i := strings.LastIndex(ref, ":"); i != -1 {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}

func intFromMap(m map[string]interface{}, key string) int {
	switch n := m[key].(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	}
	return 0
}

func stringFromMap(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

func parseResources(r map[string]interface{}) ResourceRequirements {
	rr := ResourceRequirements{}
	if requests, ok := r["requests"].(map[string]interface{}); ok {
		rr.Requests.CPU = stringFromMap(requests, "cpu")
		rr.Requests.Memory = stringFromMap(requests, "memory")
	}
	if limits, ok := r["limits"].(map[string]interface{}); ok {
		rr.Limits.CPU = stringFromMap(limits, "cpu")
		rr.Limits.Memory = stringFromMap(limits, "memory")
	}
	return rr
}

func parseProbe(container map[string]interface{}, key string) ProbeSpec {
	probe, ok := container[key].(map[string]interface{})
	if !ok {
		return ProbeSpec{}
	}

	httpGet, _ := probe["httpGet"].(map[string]interface{})
	path, _ := httpGet["path"].(string)

	return ProbeSpec{
		Path:                path,
		FailureThreshold:    intFromMap(probe, "failureThreshold"),
		InitialDelaySeconds: intFromMap(probe, "initialDelaySeconds"),
		PeriodSeconds:       intFromMap(probe, "periodSeconds"),
		SuccessThreshold:    intFromMap(probe, "successThreshold"),
		TimeoutSeconds:      intFromMap(probe, "timeoutSeconds"),
	}
}

func marshalYAML(v interface{}) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
