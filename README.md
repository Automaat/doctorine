# Doctorine

Private health status and medical document manager.

## Stack

- Frontend: SvelteKit 2 + Svelte 5 + TypeScript 6
- Backend: Go + chi + pgx
- Database: PostgreSQL 18
- UI: Tailwind CSS + Skeleton
- Deployment: Docker Compose

Exact versions live in `package.json` and `backend-go/go.mod`.

## Development

```bash
npm install
cd backend-go
go build ./...
cd ..
cp .env.example .env
mise run dev
```

Dev URLs:

- Frontend: http://localhost:5175
- Backend: http://localhost:8010
- Postgres: localhost:5434

Dev login defaults:

- Username: `admin`
- Password: `admin`

`mise run dev` enables `DOCTORINE_SEED_DEMO_DATA=true` by default, which inserts
synthetic demo illnesses, lab results, and supplements into an empty or existing
development database without duplicating rows.

## Data Model

- Illnesses: diagnosis/status notes
- Examinations: result records and review status
- Supplements: current supplements, value, and frequency
- Documents: uploaded PDFs/scans/images, stored outside the web root
- Overview: counts and recent documents

Uploads are stored under `DOCTORINE_UPLOAD_DIR`; metadata lives in Postgres.

## Single Sign-On (OIDC)

Optional login through Authelia (or any OIDC provider) next to the
password login. Set all four backend variables to enable it; leave all
unset to disable it. A partial set fails startup.

| Variable                       | Example                                                 |
| ------------------------------ | ------------------------------------------------------- |
| `DOCTORINE_OIDC_ISSUER_URL`    | `https://auth.mskalski.dev`                             |
| `DOCTORINE_OIDC_CLIENT_ID`     | `doctorine`                                             |
| `DOCTORINE_OIDC_CLIENT_SECRET` | client secret registered with the provider              |
| `DOCTORINE_OIDC_REDIRECT_URL`  | `https://doctorine.mskalski.dev/api/auth/oidc/callback` |

- Flow: authorization code with PKCE (S256), `client_secret_basic`,
  scopes `openid profile email`.
- `preferred_username` (from the ID token, or from userinfo for the
  same subject when the ID token omits it, as Authelia does by default)
  must exactly match an existing Doctorine username. No users are
  created; unknown users are refused.
- A successful login creates the same session as the password login.
  Password login and personal access tokens keep working, so they stay
  available as a fallback while the provider is down.
- Provider discovery runs on first use and is retried until it
  succeeds, so the backend starts even when the provider is down.
- The short-lived state cookie is HMAC-signed with a key derived from
  `DOCTORINE_JWT_SECRET` (or the client secret when that is unset).
- The issuer URL must match the provider's `issuer` exactly (watch for
  trailing slashes).

Authelia client example:

```yaml
identity_providers:
  oidc:
    clients:
      - client_id: doctorine
        client_name: Doctorine
        client_secret: '$pbkdf2-sha512$...'
        redirect_uris:
          - https://doctorine.mskalski.dev/api/auth/oidc/callback
        scopes: [openid, profile, email]
        response_types: [code]
        grant_types: [authorization_code]
        require_pkce: true
        pkce_challenge_method: S256
        token_endpoint_auth_method: client_secret_basic
```

## Commands

- `npm run dev`
- `npm run check`
- `npm run lint`
- `npm run test`
- `npm run test:e2e`
- `cd backend-go && go test ./...`
- `mise run dev`
- `mise run down`

## Security Notes

- Auth required for all `/api/*` routes except login/logout and the
  `/api/auth/oidc/*` login flow
- Session token stored in an HttpOnly cookie
- Production compose defaults `DOCTORINE_COOKIE_SECURE=true`; serve behind
  HTTPS and set an `https://` `ORIGIN` so secure cookies are sent. Local
  dev compose keeps it `false` for plain HTTP.
- Browser API calls route through SvelteKit `/api` proxy
- Uploaded files are not served as static assets
- Personal app, not compliance-certified medical record software

### Required repository security settings

These GitHub settings are managed manually (not in code) and must stay enabled:

- Dependency graph — prerequisite; alerts are not generated without it
  (always on for public repos, enable explicitly for private ones)
- Dependabot alerts — surfaces vulnerable dependencies as security alerts
- Dependabot security updates — opens automated PRs for vulnerable deps

Renovate handles routine version bumps; Dependabot provides the vulnerability
signal. Verify with `gh api repos/Automaat/doctorine/dependabot/alerts`
(returns a JSON list when enabled, `403` when disabled). Re-enable via:

```bash
gh api -X PUT repos/Automaat/doctorine/vulnerability-alerts
gh api -X PUT repos/Automaat/doctorine/automated-security-fixes
```

## License

Private project
