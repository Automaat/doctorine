# Doctorine Instructions

- Keep stack aligned with `../finance-buddy`: SvelteKit, Go, Postgres, Docker Compose, mise.
- Treat medical documents as private data. Never commit uploads, exports, DB dumps, or screenshots with PHI.
- Use `DOCTORINE_*` env vars for backend-specific settings.
- Run `npm run check`, `npm run lint`, `npm run test`, and `cd backend-go && go test ./...` before commit.
- Use signed commits: `git commit -s -S -m "type(scope): description"`.
