package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserRole is the permission level of a panel user.
// +kubebuilder:validation:Enum=admin;user
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

// APIToken is a personal access token. Only a SHA-256 hash of the token is stored.
type APIToken struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Hash      string       `json:"hash"`
	CreatedAt metav1.Time  `json:"createdAt"`
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
}

// UserSpec defines a panel user. The object name is the username.
type UserSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
	// +kubebuilder:default=user
	Role UserRole `json:"role"`
	// PasswordHash is an Argon2id hash in PHC format; the password itself is never stored.
	PasswordHash string `json:"passwordHash"`
	// MustChangePassword makes the user replace the password (set by an administrator) after signing in.
	MustChangePassword bool `json:"mustChangePassword,omitempty"`
	Disabled           bool `json:"disabled,omitempty"`
	// SessionEpoch invalidates all sessions of the user when it is increased.
	SessionEpoch int64      `json:"sessionEpoch,omitempty"`
	Tokens       []APIToken `json:"tokens,omitempty"`
}

// UserStatus defines the observed state of a user.
type UserStatus struct {
	// Namespace holds the game servers of the user.
	Namespace   string       `json:"namespace,omitempty"`
	LastLoginAt *metav1.Time `json:"lastLoginAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kuser
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.role`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`
// +kubebuilder:printcolumn:name="Disabled",type=boolean,JSONPath=`.spec.disabled`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// User is a panel user; every user owns a namespace for game servers.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of Users.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	register(&User{}, &UserList{})
}
