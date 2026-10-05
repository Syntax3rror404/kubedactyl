package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InviteSpec is a one-time link with which someone creates their own account. The object name is the public
// id in the link's token.
type InviteSpec struct {
	// Username is the name the account gets; empty lets the invited person choose it.
	Username string `json:"username,omitempty"`
	// +kubebuilder:default=user
	Role UserRole `json:"role"`
	// Note says whom the invite is for (shown to administrators only).
	Note string `json:"note,omitempty"`
	// Hash is a SHA-256 hash of the link's token; the token itself is never stored.
	Hash      string      `json:"hash"`
	ExpiresAt metav1.Time `json:"expiresAt"`
	// CreatedBy is the administrator who created the invite.
	CreatedBy string `json:"createdBy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=kinvite
// +kubebuilder:printcolumn:name="Username",type=string,JSONPath=`.spec.username`
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.role`
// +kubebuilder:printcolumn:name="Expires",type=date,JSONPath=`.spec.expiresAt`

// Invite lets someone create a panel account once, until it expires.
type Invite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InviteSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// InviteList contains a list of Invites.
type InviteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Invite `json:"items"`
}

func init() {
	register(&Invite{}, &InviteList{})
}
