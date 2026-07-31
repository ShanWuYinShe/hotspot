  # HotSpot Frontend

  This is the SvelteKit web frontend for HotSpot. It is a pure UI layer that must be served alongside the backend API.

  ## Running the code

  This project runs inside Docker Compose together with the Go backend, PostgreSQL, Redis, SeaweedFS, and an nginx gateway.

  ```bash
  # From the repository root:
  docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

  # Then open: http://localhost:8888
  ```

  **Do not** run `bun run dev` directly on the host and attempt to access the Vite dev server via its port — API calls will fail because they must go through the nginx gateway. See `AGENTS.md` in this directory for details.
  