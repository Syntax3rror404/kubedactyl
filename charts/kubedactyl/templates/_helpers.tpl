{{- define "kubedactyl.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Release name, or "<release>-kubedactyl" when the release name does not contain it. */}}
{{- define "kubedactyl.fullname" -}}
{{- if contains .Chart.Name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "kubedactyl.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubedactyl.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "kubedactyl.labels" -}}
{{ include "kubedactyl.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "kubedactyl.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kubedactyl.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Secret with the admin password / setup token (chart-managed or existing). */}}
{{- define "kubedactyl.secretName" -}}
{{- default (printf "%s-admin" (include "kubedactyl.fullname" .)) .Values.admin.existingSecret }}
{{- end }}

{{- define "kubedactyl.hasSecret" -}}
{{- if or .Values.admin.existingSecret .Values.admin.password .Values.admin.setupToken }}true{{ end }}
{{- end }}

{{/* Service account and roles of the self-upgrade job. */}}
{{- define "kubedactyl.upgraderName" -}}
{{- printf "%s-upgrader" (include "kubedactyl.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "kubedactyl.image" -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
{{- end }}

{{/*
Self-upgrades need the admission policy (Kubernetes 1.30+): the upgrader's service account may
change the release's cluster roles, and only the policy keeps the panel from starting anything
but the exact upgrade job with it. "true" or empty.
*/}}
{{- define "kubedactyl.selfUpgrade" -}}
{{- $policy := .Capabilities.APIVersions.Has "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy" }}
{{- if and .Values.selfUpgrade.enabled .Values.rbac.create $policy }}true{{ end }}
{{- end }}
