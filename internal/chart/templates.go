package chart

// Go templates use [[ ]] delimiters.
// Helm template expressions {{ }} pass through literally.

var chartYAMLTmpl = `apiVersion: v2
name: keycloak-operator
description: Keycloak operator for Kubernetes
type: application
version: 0.1.0
appVersion: "[[ .AppVersion ]]"
home: https://www.keycloak.org/operator/installation
sources:
  - https://github.com/keycloak/keycloak-k8s-resources
  - https://github.com/px3-dev/keycloak-operator
maintainers:
  - name: px3-dev
`

var valuesYAMLTmpl = `# Operator image
image:
  repository: [[ .OperatorImage ]]
  # Defaults to appVersion
  tag: ""
  pullPolicy: IfNotPresent

# Keycloak server image used by the operator when creating instances.
# The operator injects this as RELATED_IMAGE_KEYCLOAK.
keycloakImage:
  repository: [[ .KeycloakImage ]]
  # Defaults to appVersion
  tag: ""

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

replicas: [[ .Deployment.Replicas ]]

resources:
  requests:
    cpu: [[ .Deployment.Resources.Requests.CPU ]]
    memory: [[ .Deployment.Resources.Requests.Memory ]]
  limits:
    cpu: [[ .Deployment.Resources.Limits.CPU ]]
    memory: [[ .Deployment.Resources.Limits.Memory ]]

serviceAccount:
  create: true
  annotations: {}
  # If not set and create is true, a name is generated using the fullname template.
  name: ""

service:
  type: [[ .Service.Type ]]
  port: [[ .Service.Port ]]

nodeSelector: {}
tolerations: []
affinity: {}
podAnnotations: {}
podLabels: {}
`

var helmignoreContent = `# Patterns to ignore when packaging Helm charts.
.DS_Store
.git/
.gitignore
*.swp
*.bak
*.tmp
*.orig
*~
.idea/
.vscode/
`

var helpersContent = `{{/*
Expand the name of the chart.
*/}}
{{- define "keycloak-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "keycloak-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "keycloak-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "keycloak-operator.labels" -}}
helm.sh/chart: {{ include "keycloak-operator.chart" . }}
{{ include "keycloak-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "keycloak-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "keycloak-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "keycloak-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "keycloak-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
`

var notesContent = `Keycloak Operator {{ .Chart.AppVersion }} has been installed.

The operator is watching namespace {{ .Release.Namespace }} for Keycloak and KeycloakRealmImport resources.

To create a Keycloak instance, apply a Keycloak CR:

  kubectl apply -n {{ .Release.Namespace }} -f - <<EOF
  apiVersion: k8s.keycloak.org/v2alpha1
  kind: Keycloak
  metadata:
    name: my-keycloak
  spec:
    instances: 1
    hostname:
      hostname: my-keycloak.example.com
    http:
      tlsSecret: my-tls-secret
  EOF
`

var serviceAccountTmpl = `{{- if .Values.serviceAccount.create -}}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "keycloak-operator.serviceAccountName" . }}
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
`

var deploymentTmpl = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "keycloak-operator.fullname" . }}
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicas }}
  selector:
    matchLabels:
      {{- include "keycloak-operator.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      labels:
        {{- include "keycloak-operator.labels" . | nindent 8 }}
        {{- with .Values.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "keycloak-operator.serviceAccountName" . }}
      containers:
        - name: [[ .Deployment.ContainerName ]]
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          env:
            - name: KUBERNETES_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: RELATED_IMAGE_KEYCLOAK
              value: "{{ .Values.keycloakImage.repository }}:{{ .Values.keycloakImage.tag | default .Chart.AppVersion }}"
[[- range .Deployment.ExtraEnv ]]
            - name: [[ .Name ]]
              value: [[ .Value ]]
[[- end ]]
          ports:
            - name: http
              containerPort: [[ .Deployment.ContainerPort ]]
              protocol: TCP
[[- if .Deployment.Probes.Liveness.Path ]]
          livenessProbe:
            httpGet:
              path: [[ .Deployment.Probes.Liveness.Path ]]
              port: http
              scheme: HTTP
            failureThreshold: [[ .Deployment.Probes.Liveness.FailureThreshold ]]
            initialDelaySeconds: [[ .Deployment.Probes.Liveness.InitialDelaySeconds ]]
            periodSeconds: [[ .Deployment.Probes.Liveness.PeriodSeconds ]]
            successThreshold: [[ .Deployment.Probes.Liveness.SuccessThreshold ]]
            timeoutSeconds: [[ .Deployment.Probes.Liveness.TimeoutSeconds ]]
[[- end ]]
[[- if .Deployment.Probes.Readiness.Path ]]
          readinessProbe:
            httpGet:
              path: [[ .Deployment.Probes.Readiness.Path ]]
              port: http
              scheme: HTTP
            failureThreshold: [[ .Deployment.Probes.Readiness.FailureThreshold ]]
            initialDelaySeconds: [[ .Deployment.Probes.Readiness.InitialDelaySeconds ]]
            periodSeconds: [[ .Deployment.Probes.Readiness.PeriodSeconds ]]
            successThreshold: [[ .Deployment.Probes.Readiness.SuccessThreshold ]]
            timeoutSeconds: [[ .Deployment.Probes.Readiness.TimeoutSeconds ]]
[[- end ]]
[[- if .Deployment.Probes.Startup.Path ]]
          startupProbe:
            httpGet:
              path: [[ .Deployment.Probes.Startup.Path ]]
              port: http
              scheme: HTTP
            failureThreshold: [[ .Deployment.Probes.Startup.FailureThreshold ]]
            initialDelaySeconds: [[ .Deployment.Probes.Startup.InitialDelaySeconds ]]
            periodSeconds: [[ .Deployment.Probes.Startup.PeriodSeconds ]]
            successThreshold: [[ .Deployment.Probes.Startup.SuccessThreshold ]]
            timeoutSeconds: [[ .Deployment.Probes.Startup.TimeoutSeconds ]]
[[- end ]]
          {{- with .Values.resources }}
          resources:
            {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
`

var serviceContent = `apiVersion: v1
kind: Service
metadata:
  name: {{ include "keycloak-operator.fullname" . }}
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "keycloak-operator.selectorLabels" . | nindent 4 }}
`

var clusterRoleTmpl = `[[- range $i, $role := .RBAC.ClusterRoles ]]
[[- if $i ]]
---
[[- end ]]
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "keycloak-operator.fullname" . }}-[[ $role.Suffix ]]
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
rules:
[[ indent 2 $role.RulesYAML ]]
[[- end ]]
`

var clusterRoleBindingTmpl = `[[- range $i, $binding := .RBAC.ClusterRoleBindings ]]
[[- if $i ]]
---
[[- end ]]
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ include "keycloak-operator.fullname" . }}-[[ $binding.Suffix ]]
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
roleRef:
  kind: [[ $binding.RoleRefKind ]]
  apiGroup: rbac.authorization.k8s.io
[[- if $binding.IsBuiltinRole ]]
  name: [[ $binding.RoleRefName ]]
[[- else ]]
  name: {{ include "keycloak-operator.fullname" . }}-[[ $binding.RoleSuffix ]]
[[- end ]]
subjects:
  - kind: ServiceAccount
    name: {{ include "keycloak-operator.serviceAccountName" . }}
    namespace: {{ .Release.Namespace }}
[[- end ]]
`

var roleTmpl = `[[- range $i, $role := .RBAC.Roles ]]
[[- if $i ]]
---
[[- end ]]
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ include "keycloak-operator.fullname" . }}-[[ $role.Suffix ]]
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
rules:
[[ indent 2 $role.RulesYAML ]]
[[- end ]]
`

var roleBindingTmpl = `[[- range $i, $binding := .RBAC.RoleBindings ]]
[[- if $i ]]
---
[[- end ]]
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ include "keycloak-operator.fullname" . }}-[[ $binding.Suffix ]]
  labels:
    {{- include "keycloak-operator.labels" . | nindent 4 }}
roleRef:
[[- if $binding.IsBuiltinRole ]]
  kind: [[ $binding.RoleRefKind ]]
  apiGroup: rbac.authorization.k8s.io
  name: [[ $binding.RoleRefName ]]
[[- else if eq $binding.RoleRefKind "Role" ]]
  kind: Role
  apiGroup: rbac.authorization.k8s.io
  name: {{ include "keycloak-operator.fullname" . }}-[[ $binding.RoleSuffix ]]
[[- else ]]
  kind: ClusterRole
  apiGroup: rbac.authorization.k8s.io
  name: {{ include "keycloak-operator.fullname" . }}-[[ $binding.RoleSuffix ]]
[[- end ]]
subjects:
  - kind: ServiceAccount
    name: {{ include "keycloak-operator.serviceAccountName" . }}
[[- end ]]
`
