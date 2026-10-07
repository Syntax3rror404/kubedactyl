# Single sign-on (OIDC)

Users can sign in through an OpenID Connect identity provider (IdP). The IdP's groups decide who gets in and
with which role. Sign-in with a password keeps working next to it.

**Tested with Keycloak only.** The panel uses the standard authorization code flow, so other providers
(Authentik, Entra ID, Dex, ...) should work as long as they put the username and the groups into the ID token,
but they have not been tested.

## How a sign-in works

1. The sign-in page shows *Sign in with &lt;name&gt;* while single sign-on is on.
2. The panel redirects to the IdP (authorization code flow with PKCE S256, `state` and `nonce`). It keeps the
   flow in a signed cookie for 10 minutes (HttpOnly, SameSite=Lax; over HTTPS `__Host-kd_oidc`, which sites on
   other subdomains cannot set).
3. The IdP redirects back to the configured redirect URL (`/api/auth/oidc/callback`). The panel completes each
   flow once, exchanges the code (with the client secret and the PKCE verifier), verifies the ID token
   (signature, issuer, audience, expiry, nonce) and reads its claims.
4. The panel starts its own session (cookie `kd_session`, lifetime *Settings → Security*) and opens the page the
   sign-in started from.

The IdP is asked only during a sign-in: its discovery document and signing keys are kept in memory (keys are
fetched again when the IdP rotates them), its tokens are not stored. Removing someone from a group at the IdP
therefore takes effect at their next sign-in; to lock someone out at once, disable the account on the *Users*
page (this ends its sessions).

## Accounts

- **Groups:** members of the *admin group* sign in as administrators, members of the *user group* as users,
  everyone else is refused. The role is set again at every sign-in. The groups are compared exactly; with
  Keycloak's *Full group path* on they look like `/kubedactyl-admins`.
- **Matching:** the account linked to the IdP user (issuer and `sub`) is used, also when their username at the IdP
  changes. Without one, a new account is created (with its namespace) under the username from the *username
  claim* (default `preferred_username`). It must already follow the panel's rule (3-32 lowercase letters, digits
  or dashes, starting with a letter); it is not lower-cased, so `Admin` and `admin` of the IdP never meet.
- **Existing accounts:** when an account with that username exists and is not linked, the sign-in is refused,
  unless *Link existing accounts by username* is on: then the account is linked to the IdP user (like Nextcloud
  does). Turn it on to take over existing accounts and off again afterwards; while it is on, whoever can choose a
  username at the IdP gets the account with that name. Linking ends the account's sessions and API tokens. After
  the link only that IdP user can sign in to the account through single sign-on.
- **Profile:** display name (`name`), email (`email`) and role come from the IdP at every sign-in; the *Users*
  page shows linked accounts with an *OIDC* badge and does not let administrators change these fields. The edit
  dialog shows the IdP user ID (`sub`) and the issuer.
- **Passwords:** a linked account loses its password and signs in only through the IdP. With *Keep passwords of
  linked accounts* the password stays and works next to single sign-on; the role then changes only at a single
  sign-on, not at a sign-in with the password.
- **Disabled accounts** are refused. **The last administrator** is not demoted when the IdP removes them from the
  admin group: the sign-in is refused instead and the panel log says why.
- **Local accounts** keep working, and administrators can still create users and invites. Keep one local
  administrator that is not linked (for example the `admin` from the setup): it still signs in when the IdP is
  down.
- **API tokens** work as before; single sign-on users create them under *Account*.

## Keycloak

In the realm of your users:

1. **Client:** *Clients → Create client*, type *OpenID Connect*, client ID for example `kubedactyl`.
   *Client authentication* on (confidential client), *Standard flow* on, everything else off. Under
   *Advanced → Proof Key for Code Exchange*, set *S256*.
2. **Redirect URL:** *Valid redirect URIs* = exactly the redirect URL of the *OpenID Connect* card, for example
   `https://panel.example.com/api/auth/oidc/callback` (no wildcards). *Web origins* is not needed.
3. **Groups in the token:** in the client, tab *Client scopes → kubedactyl-dedicated → Add mapper → By
   configuration → Group Membership*, token claim name `groups`, *Full group path* off, *Add to ID token* on.
4. **Groups:** create for example `kubedactyl-admins` and `kubedactyl-users` and add the users.
5. **Client secret:** *Credentials → Client secret*.
6. **Usernames:** keep *Realm settings → Login → Edit username* off, and do not put self-registered users into
   the groups automatically: while *Link existing accounts by username* is on, the username decides which
   existing account someone gets.
7. **Panel:** enter issuer (`https://<keycloak host>/realms/<realm>`), client ID, client secret and both groups
   under [Panel settings](#panel-settings) and turn single sign-on on.

## Panel settings

*Settings → OpenID Connect* (stored in the `PanelSettings` resource, field `oidc`):

| Field | Meaning |
|---|---|
| Offer single sign-on | Shows the button on the sign-in page and enables the callback |
| Issuer URL | For Keycloak `https://<host>/realms/<realm>`; must use https (http only for localhost) |
| Client ID, client secret | From the IdP; an empty secret means a public client (PKCE only) |
| Redirect URL | `https://<panel>/api/auth/oidc/callback`, filled with the current address when single sign-on is turned on; taken from the settings, never from the request |
| Admin group, user group | Group names in the groups claim (required, must differ) |
| Groups claim, username claim | Default `groups` and `preferred_username` |
| Button name | *Sign in with &lt;name&gt;* (default *SSO*) |
| Link existing accounts by username | Off by default; see [Accounts](#accounts) |
| Keep passwords of linked accounts | See [Accounts](#accounts) |

Saving checks that the issuer answers (its discovery document). The client secret is stored in the Secret
`kubedactyl-oidc` in the panel namespace, never in the settings; the API only says whether one is stored
(`oidcClientSecretSet`). An empty field on the card keeps the stored secret. The sidebar health check
*Single sign-on* shows when the IdP cannot be reached.

## Troubleshooting

The sign-in page shows one of three messages; the panel log (`single sign-on failed`) has the details, with the
IdP username and `sub` when they are known.

| Message | Cause |
|---|---|
| Your account has no access to this panel | Not in the admin or user group (check the group mapper and the claim name), or the account is disabled |
| This account cannot sign in here | An account with the username exists and *Link existing accounts by username* is off, the account is linked to another IdP user, the username is no valid panel username, or it is the last administrator and the IdP removed it from the admin group |
| Single sign-on failed | Wrong client secret or redirect URL, the IdP is unreachable, the sign-in took longer than 10 minutes or was started in another browser |

To link an account to another IdP user (for example after moving to a new realm), remove its link; the next
single sign-on links it again by its username:

```sh
kubectl -n kubedactyl patch kuser alice --type merge -p '{"spec":{"oidc":null}}'
```
