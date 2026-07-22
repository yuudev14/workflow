# Dev Keycloak (M3 — OIDC SSO)

`ytsoar-realm.json` is imported on boot (`start-dev --import-realm`). It creates
the `ytsoar` realm with a confidential client, a `groups` claim mapper, two
groups, and two test users.

| Realm admin console | http://localhost:8180 (admin / admin) |
|---|---|
| Realm | `ytsoar` |
| Client | `ytsoar` (confidential, PKCE S256), secret `ytsoar-dev-secret` |
| Test user | `kc-admin` / `admin123` — group `soc-admins` |
| Test user | `kc-analyst` / `analyst123` — group `soc-analysts` |

## Split-horizon issuer

The browser reaches Keycloak at `http://localhost:8180`; the api reaches it
in-network at `http://ytsoar_keycloak:8180`. The provider config carries both:
`issuer` (public — what the id_token `iss` is, and where the browser is
redirected) and `internal_issuer` (where the api runs discovery). The api pins
the public issuer via `oidc.InsecureIssuerURLContext`, and Keycloak's
`KC_HOSTNAME_BACKCHANNEL_DYNAMIC=true` makes the token endpoint resolve to the
internal host the api actually called. See `adapters/oidcclient`.

## Seed the provider row

The provider holds the client secret, so it is **not** a migration — create it
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
