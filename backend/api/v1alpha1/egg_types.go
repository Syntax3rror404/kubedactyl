package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DockerImage is one selectable runtime image of an egg (display name -> image).
type DockerImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// ConfigReplace is a single key replacement inside a config file (one entry of the egg's
// config.files "find" map).
type ConfigReplace struct {
	// Match is the key (or line prefix for the "file" parser) to look for.
	Match string `json:"match"`
	// IfValue only replaces when the current value equals it; a "regex:" prefix
	// turns it into a regular expression replacement on the current value.
	// +optional
	IfValue string `json:"ifValue,omitempty"`
	// ReplaceWith is the new value; placeholders like {{server.build.default.port}} are supported.
	ReplaceWith string `json:"replaceWith"`
	// ValueType is the JSON type of the value in the egg (string, number or boolean).
	// +kubebuilder:validation:Enum=string;number;boolean
	// +optional
	ValueType string `json:"valueType,omitempty"`
}

// ConfigFile describes how a config file is modified before every server start.
type ConfigFile struct {
	// File is the path relative to the server root (/home/container).
	File string `json:"file"`
	// +kubebuilder:validation:Enum=file;yaml;yml;properties;ini;json;xml
	Parser  string          `json:"parser"`
	Replace []ConfigReplace `json:"replace,omitempty"`
}

// InstallScript is the egg installation script, executed once per (re)install.
type InstallScript struct {
	Script     string `json:"script,omitempty"`
	Container  string `json:"container,omitempty"`
	Entrypoint string `json:"entrypoint,omitempty"`
}

// EggVariable is a user configurable environment variable of an egg.
type EggVariable struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	EnvVariable  string `json:"envVariable"`
	DefaultValue string `json:"defaultValue,omitempty"`
	UserViewable bool   `json:"userViewable,omitempty"`
	UserEditable bool   `json:"userEditable,omitempty"`
	// Rules are Laravel style validation rules, e.g. "required|string|max:20".
	Rules     string `json:"rules,omitempty"`
	FieldType string `json:"fieldType,omitempty"`
}

// EggSource records where an egg was imported from.
type EggSource struct {
	// Format of the imported file, e.g. PTDL_v2 or PLCN_v3.
	Format string `json:"format,omitempty"`
	// ExportedAt is the exported_at of the imported file: when its panel exported the egg.
	// +optional
	ExportedAt *metav1.Time `json:"exportedAt,omitempty"`
	UpdateURL  string       `json:"updateUrl,omitempty"`
	// ImportedFrom is the URL the egg was fetched from, if any.
	ImportedFrom string      `json:"importedFrom,omitempty"`
	ImportedAt   metav1.Time `json:"importedAt,omitempty"`
	// UUID identifies the egg across panels (Pelican overwrites an egg with the same UUID on import).
	// +optional
	UUID string `json:"uuid,omitempty"`
	// EditedAt is set when the egg was changed in the panel after its import.
	// +optional
	EditedAt *metav1.Time `json:"editedAt,omitempty"`
	// AutoUpdate downloads the update URL every hour and replaces the egg when the file changed
	// (changes made in the panel are replaced, like "Update from URL" does).
	// +optional
	AutoUpdate bool `json:"autoUpdate,omitempty"`
}

// EggStatus is what the automatic update last observed.
type EggStatus struct {
	// UpdateCheckedAt is the last automatic check of the update URL.
	// +optional
	UpdateCheckedAt *metav1.Time `json:"updateCheckedAt,omitempty"`
	// UpdatedAt is when the last automatic check found a newer file and applied it.
	// +optional
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
	// UpdateError is the problem of the last automatic check ("" when it worked).
	// +optional
	UpdateError string `json:"updateError,omitempty"`
}

// EggSpec is a normalized Pterodactyl/Pelican egg.
type EggSpec struct {
	DisplayName string `json:"displayName"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	// Icon is an optional data URI (Pelican eggs).
	Icon     string   `json:"icon,omitempty"`
	Features []string `json:"features,omitempty"`
	// Tags group eggs (Pelican).
	// +optional
	Tags         []string      `json:"tags,omitempty"`
	DockerImages []DockerImage `json:"dockerImages"`
	// Startup is the default startup command (with {{VAR}} placeholders).
	Startup string `json:"startup"`
	// Stop is the stop command; a leading "^" means a signal (^C = SIGINT, ^^C = SIGKILL).
	Stop string `json:"stop,omitempty"`
	// StartupDone lists console output snippets that mark the server as running.
	// A "regex:" prefix makes an entry a regular expression.
	StartupDone  []string      `json:"startupDone,omitempty"`
	StripAnsi    bool          `json:"stripAnsi,omitempty"`
	ConfigFiles  []ConfigFile  `json:"configFiles,omitempty"`
	FileDenylist []string      `json:"fileDenylist,omitempty"`
	Install      InstallScript `json:"install,omitempty"`
	Variables    []EggVariable `json:"variables,omitempty"`
	Source       EggSource     `json:"source,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=egg
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Format",type=string,JSONPath=`.spec.source.format`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Egg is a game server template (Pterodactyl egg).
type Egg struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EggSpec `json:"spec"`
	// +optional
	Status EggStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EggList contains a list of Eggs.
type EggList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Egg `json:"items"`
}

func init() {
	register(&Egg{}, &EggList{})
}
