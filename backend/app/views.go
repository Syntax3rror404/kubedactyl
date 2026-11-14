package app

import "app/api/v1alpha1"

// The app's parts of requests and answers of the base API (backend/base/api): embedded there, so their fields
// sit next to the base fields in the JSON.

// InviteRequest is what a new invite adds: a subuser and the servers it sees.
type InviteRequest struct {
	// Subuser invites a subuser of the caller (role user). Users always invite subusers.
	Subuser bool `json:"subuser,omitempty"`
	// Shares are servers of the caller the subuser sees once the account exists.
	Shares []v1alpha1.InviteShare `json:"shares,omitempty"`
}

// UserView is what an account adds in the answers: its owner (for a subuser), namespace and servers.
type UserView struct {
	// Owner is set for a subuser: the user whose shared servers it sees. A subuser has no namespace.
	Owner     string `json:"owner,omitempty"     example:"bob"`
	Namespace string `json:"namespace,omitempty" example:"kubedactyl-user-alice"`
	// Servers counts the servers of the user, for a subuser the servers shared with it.
	Servers int `json:"servers" example:"2"`
}

// InviteView is what an invite adds: its owner (an invite for a subuser) and the servers the subuser sees.
type InviteView struct {
	// Owner is set for an invite of a subuser: the account will see the servers this user shares.
	Owner string `json:"owner,omitempty" example:"bob"`
	// Shares are the servers the subuser sees once the account exists.
	Shares []v1alpha1.InviteShare `json:"shares"`
}

// RequestRates are the request rates the footer shows besides the user's and the panel's: none in this app.
type RequestRates struct{}
