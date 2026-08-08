# Dev Keycloak (M3 - OIDC SSO)

`ytsoar-realm.json` is imported on boot (`start-dev --import-realm`). It creates
the `ytsoar` realm with a confidential client, a `groups` claim mapper, two
groups, and two test users.

| Realm admin console | http://localhost:8180 (admin / admin) |
|---|---|
| Realm | `ytsoar` |
| Client | `ytsoar` (confidential, PKCE S256), secret `ytsoar-dev-secret` |
| Test user | `kc-admin` / `admin123` - group `soc-admins` |
| Test user | `kc-analyst` / `analyst123` - group `soc-analysts` |

## Split-horizon issuer

The browser reaches Keycloak at `http://localhost:8180`; the api reaches it
in-network at `http://ytsoar_keycloak:8180`. The provider config carries both:
`issuer` (public - what the id_token `iss` is, and where the browser is
redirected) and `internal_issuer` (where the api runs discovery). The api pins
the public issuer via `oidc.InsecureIssuerURLContext`, and Keycloak's
`KC_HOSTNAME_BACKCHANNEL_DYNAMIC=true` makes the token endpoint resolve to the
internal host the api actually called. See `adapters/oidcclient`.

## Seed the provider row

The provider holds the client secret, so it is **not** a migration - create it
once against a running stack (or via the UI at **Settings → Providers**).

```bash
# 1. sign in as the local admin, saving the session cookies
curl -s -c /tmp/ytsoar.cookies \
  -X POST http://localhost:9999/auth-api/api/auth/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123!"}' > /dev/null

# 2. create the Keycloak OIDC provider
curl -s -b /tmp/ytsoar.cookies \
  -X POST http://localhost:9999/auth-api/api/auth-providers/v1 \
  -H 'Content-Type: application/json' \
  -d '{
        "type": "oidc",
        "name": "Keycloak",
        "enabled": true,
        "config": {
          "issuer": "http://localhost:8180/realms/ytsoar",
          "internal_issuer": "http://ytsoar_keycloak:8180/realms/ytsoar",
          "client_id": "ytsoar",
          "client_secret": "ytsoar-dev-secret",
          "scopes": ["openid", "profile", "email"],
          "groups_claim": "groups",
          "group_role_mapping": { "soc-admins": "admin", "soc-analysts": "analyst" },
          "default_role": "viewer",
          "allow_jit": true
        }
      }'
```

After seeding, the login screen shows **Continue with Keycloak**. Signing in as
`kc-analyst` JIT-provisions a YTSoar user with the `analyst` role; moving a user
between Keycloak groups re-syncs their role on their next login.

## Known limitation - one identity cannot span two providers

`users.email` carries a **global** `UNIQUE` constraint, while `external_id` is
scoped per provider (`<provider-uuid>|<sub>`). We deliberately never link an
incoming SSO identity to an existing account by email - an IdP that lets a user
set an unverified address could otherwise take over someone else's account - so
a person who already exists under provider A and signs in through provider B
cannot be provisioned. Postgres rejects the insert with
`users_email_key` (SQLSTATE 23505) and the browser lands on `/login?error=sso`
with nothing explaining why.

This bites when migrating between IdPs (Keycloak → Okta) or running two
providers side by side, and it is reproducible today.

**Deferred - needs a schema decision before implementing.** Options:

1. Scope the constraint per provider: `UNIQUE (auth_provider, email)`. Keeps
   email unique where it matters and lets the same human exist once per IdP.
2. Drop the global unique on `email` entirely and rely on `username` +
   `external_id` for identity. Most permissive; check nothing else assumes
   email is a key first.
3. Minimum viable: leave the schema alone and classify the collision into an
   actionable error, so an operator sees "an account with this email already
   exists under a different provider" instead of a generic SSO failure.
   (The `mapUniqueViolation` helper in `internal/adapters/repository/utils.go`
   already does exactly this for duplicate provider names → 409.)

Option 1 is the likely answer, but it is a migration plus a re-check of every
`GetUserByEmail` caller, so it is being handled as its own piece of work.

## Dev realm drift

`--import-realm` only imports when the realm does **not** already exist, so
editing `ytsoar-realm.json` has no effect on a container that already has the
realm. A long-lived dev container can therefore disagree with the file - in
particular the `ytsoar` client secret may be a Keycloak-generated value rather
than `ytsoar-dev-secret`. Read the live one with:

```sh
# realm admin: admin / admin
CID=$(curl -s -H "Authorization: Bearer $TOKEN" \
  'http://localhost:8180/admin/realms/ytsoar/clients?clientId=ytsoar' | jq -r '.[0].id')
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8180/admin/realms/ytsoar/clients/$CID/client-secret" | jq -r .value
```

Or recreate the container with a clean volume to pick the file up.
