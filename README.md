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

## Data Model

- Illnesses: diagnosis/status notes
- Examinations: result records and review status
- Documents: uploaded PDFs/scans/images, stored outside the web root
- Overview: counts and recent documents

Uploads are stored under `DOCTORINE_UPLOAD_DIR`; metadata lives in Postgres.

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

- Auth required for all `/api/*` routes except login/logout
- Session token stored in an HttpOnly cookie
- Browser API calls route through SvelteKit `/api` proxy
- Uploaded files are not served as static assets
- Personal app, not compliance-certified medical record software

## License

Private project
