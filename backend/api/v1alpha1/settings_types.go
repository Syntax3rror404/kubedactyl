package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SettingsName is the name of the single PanelSettings object in the panel namespace.
const SettingsName = "panel"

// PanelSettingsSpec holds the settings an administrator manages in the panel.
type PanelSettingsSpec struct {
	// ExternalDomain is shown to users as the server address (domain:port) instead of the
	// load balancer IP. Empty shows the IP.
	// +optional
	ExternalDomain string `json:"externalDomain,omitempty"`
	// StorageClasses can be selected for server volumes.
	// +optional
	StorageClasses []string `json:"storageClasses,omitempty"`
	// DefaultStorageClass is preselected for new servers (one of StorageClasses).
	// +optional
	DefaultStorageClass string `json:"defaultStorageClass,omitempty"`
	// LoadBalancerPools are the Cilium LB IPAM pools that can be selected for servers.
	// +optional
	LoadBalancerPools []string `json:"loadBalancerPools,omitempty"`
	// DefaultLoadBalancerPool is preselected for new servers (one of LoadBalancerPools).
	// +optional
	DefaultLoadBalancerPool string `json:"defaultLoadBalancerPool,omitempty"`
	// AllowPrivateNetworks lets game servers reach private networks (other namespaces,
	// nodes, the Kubernetes API, the LAN). By default every user namespace gets a network
	// policy that only allows the internet, the cluster DNS and the user's own servers.
	// +optional
	AllowPrivateNetworks bool `json:"allowPrivateNetworks,omitempty"`
	// DisableAPIDocs turns the API documentation (Swagger UI at /swagger/) off. The API itself
	// keeps working.
	// +optional
	DisableAPIDocs bool `json:"disableApiDocs,omitempty"`
	// ServerNotice is shown to users every time they open one of their servers (plain text).
	// +optional
	// +kubebuilder:validation:MaxLength=2000
	ServerNotice string `json:"serverNotice,omitempty"`
	// LegalNotice (imprint) and PrivacyPolicy are Markdown texts linked in the footer of every
	// page, also before sign-in.
	// +optional
	// +kubebuilder:validation:MaxLength=20000
	LegalNotice string `json:"legalNotice,omitempty"`
	// +optional
	// +kubebuilder:validation:MaxLength=20000
	PrivacyPolicy string `json:"privacyPolicy,omitempty"`
	// BrandName and BrandTagline replace "Kubedactyl" and "Game servers on Kubernetes" in the
	// sidebar, on the sign-in page and in the browser title (the footer keeps the software name).
	// +optional
	// +kubebuilder:validation:MaxLength=40
	BrandName string `json:"brandName,omitempty"`
	// +optional
	// +kubebuilder:validation:MaxLength=80
	BrandTagline string `json:"brandTagline,omitempty"`
	// BrandLogo and Favicon are images as data URLs (PNG, JPEG, GIF, WebP, SVG or ICO, at most
	// 128 KiB); without a logo the built-in one is shown, without a favicon the browser's default.
	// +optional
	// +kubebuilder:validation:MaxLength=180000
	BrandLogo string `json:"brandLogo,omitempty"`
	// +optional
	// +kubebuilder:validation:MaxLength=180000
	Favicon string `json:"favicon,omitempty"`
	// SessionHours is how long a sign-in lasts (default 12). It applies to every session, so
	// shortening it also ends older sessions.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=720
	SessionHours int32 `json:"sessionHours,omitempty"`
	// APITokenMaxDays is the longest lifetime of an API token (default 90). It also limits the
	// tokens created before, counted from their creation.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3650
	APITokenMaxDays int32 `json:"apiTokenMaxDays,omitempty"`
	// KubeAPIQPS is how many requests per second the panel sends to the Kubernetes API at most
	// (default 50, bursts of twice that). It applies at once.
	// +optional
	// +kubebuilder:validation:Minimum=5
	// +kubebuilder:validation:Maximum=1000
	KubeAPIQPS int32 `json:"kubeApiQps,omitempty"`
	// KubeAPIUserQPS is how many requests per second one user may send to the panel (default 10,
	// bursts of twice that); each may lead to Kubernetes API calls. More are refused with 429, so
	// one user cannot use up KubeAPIQPS for everybody. It applies at once.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=200
	KubeAPIUserQPS int32 `json:"kubeApiUserQps,omitempty"`
	// EggLibraries are git repositories (https://<host>/<owner>/<repo>) whose eggs the egg
	// library on the eggs page lists. The panel reads them when the library is opened and keeps
	// nothing of them in the cluster.
	// +optional
	// +kubebuilder:validation:MaxItems=20
	EggLibraries []string `json:"eggLibraries,omitempty"`
	// OIDC signs users in through an OpenID Connect identity provider (single sign-on). The client
	// secret is kept in a Secret, not here.
	// +optional
	OIDC OIDCSettings `json:"oidc,omitempty"`
}

// OIDCSettings configure the sign-in through an OpenID Connect identity provider. Its groups decide
// who may sign in and with which role.
type OIDCSettings struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// Name is shown on the sign-in button: "Sign in with <name>".
	// +optional
	// +kubebuilder:validation:MaxLength=40
	Name string `json:"name,omitempty"`
	// IssuerURL is the issuer of the identity provider (its discovery document is at
	// <issuerUrl>/.well-known/openid-configuration).
	// +optional
	IssuerURL string `json:"issuerUrl,omitempty"`
	// +optional
	ClientID string `json:"clientId,omitempty"`
	// RedirectURL is the callback of the panel registered at the identity provider
	// (https://<panel>/api/auth/oidc/callback). It is not taken from the request: a forged Host
	// header must not send the code elsewhere.
	// +optional
	RedirectURL string `json:"redirectUrl,omitempty"`
	// UsernameClaim holds the panel username (default preferred_username).
	// +optional
	UsernameClaim string `json:"usernameClaim,omitempty"`
	// GroupsClaim holds the groups of the user (default groups).
	// +optional
	GroupsClaim string `json:"groupsClaim,omitempty"`
	// Members of AdminGroup sign in as administrators, members of UserGroup as users; nobody
	// else may sign in.
	// +optional
	AdminGroup string `json:"adminGroup,omitempty"`
	// +optional
	UserGroup string `json:"userGroup,omitempty"`
	// LinkByUsername links an existing account that is not linked yet to the user of the identity
	// provider with the same username (to bootstrap). Off: such a sign-in is refused, so nobody
	// can take over an account by choosing its name at the identity provider.
	// +optional
	LinkByUsername bool `json:"linkByUsername,omitempty"`
	// KeepPasswords keeps the password of an existing account when it is linked to the identity
	// provider; otherwise the account then signs in only through the identity provider.
	// +optional
	KeepPasswords bool `json:"keepPasswords,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=kdsettings
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.externalDomain`
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.defaultStorageClass`
// +kubebuilder:printcolumn:name="Pool",type=string,JSONPath=`.spec.defaultLoadBalancerPool`

// PanelSettings holds the panel wide settings (one object named "panel").
type PanelSettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PanelSettingsSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// PanelSettingsList contains a list of PanelSettings.
type PanelSettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PanelSettings `json:"items"`
}

func init() {
	register(&PanelSettings{}, &PanelSettingsList{})
}
