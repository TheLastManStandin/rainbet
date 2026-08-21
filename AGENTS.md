# Agent Notes

## Commands

- Run the API from the repository root with `task run`; Task executes it from `backend`.
- Run `task check` before handoff; it runs `go vet ./...` followed by `go test ./...` inside `backend`.
- Run the HTTP adapter tests only with `go -C backend test ./internal/delivery/httpapi`.
- Run the HTTP E2E suite with `task e2e`; it starts an in-process TCP server with a temporary SQLite database.
- `task build` writes `bin/rainbet-api`; `bin/` is ignored.
- Run the complete application with `docker compose up --build`; the frontend is available at `http://localhost:8080` and proxies `/api` to the backend.
- `task compose` runs the full stack in the background. `task tunnel` starts it and prints a temporary public Cloudflare Quick Tunnel URL; press `Ctrl+C` to stop only the tunnel.
- Stop the Compose stack with `docker compose down`; its `rainbet-data` volume keeps the SQLite database between runs.
- Inspect or mutate the Compose database with `docker compose exec backend sqlite3 /data/rainbet.db`.

## Backend

- `backend` is the only Go module (`go 1.22`); `backend/cmd/api/main.go` is the executable entrypoint.
- `internal/domain` owns entities and business rules and must not import application, delivery, or infrastructure packages.
- `internal/application` owns use cases and ports. Keep HTTP, SQL, bcrypt, and driver details outside this layer.
- `internal/delivery/httpapi` registers routes, handles JSON/Basic Auth, and translates domain/application errors into HTTP responses.
- `internal/infrastructure/sqlite` creates and migrates the SQLite schema and implements application repositories plus the Unit of Work. It seeds `user` / `user` with `balanceDollars = 10000` ($100.00 in cents), a random private `serverSeed`, and `transactionNumber = 0` only when that username is absent.
- `internal/infrastructure/password` implements password verification with bcrypt; `internal/infrastructure/fairness` implements deterministic mine generation.
- `cmd/api/main.go` is the composition root. Construct and connect concrete adapters there rather than inside application or domain packages.
- `games` uses `userId` as a foreign key to `users`, `betAmount BIGINT` in cents, `gridSize` as the total cell count, `mines`, `demo`, JSON text in `openedCells`, statuses `inProcess`, `cachedOut`, or `failed`, game seed inputs/nonce, and `startedAt`.
- SQLite uses the CGO driver `github.com/mattn/go-sqlite3` with foreign key enforcement enabled. `DATABASE_PATH` overrides the file path; under `task run`, the default is the ignored `backend/rainbet.db`.
- `GET /api/user` returns the authenticated user's `balance` as a two-decimal dollar string.
- `POST /api/user/balance` is an unauthenticated test endpoint that accepts `username` and dollar `amount`, then updates the named user's balance without ownership or amount checks.
- `POST /api/mines/bets` accepts dollar `betAmount` values with at most two decimals and only permits `gridSize` 25, 36, 49, or 64. It creates an `inProcess` game, copies the user's seed and current nonce, and increments the user's transaction number in the same database transaction. Real games deduct a positive bet; demo games require `betAmount = 0` and do not change the balance.
- `POST /api/mines/bets/{gameID}/moves` opens a zero-based `cellIndex`; `POST /api/mines/bets/{gameID}/cashout` only succeeds for an active game with at least one diamond. Cashout payouts use the exact coefficient calculation, round down to cents, and return `payout` as a two-decimal dollar string.

## Frontend

- `frontend` is a static HTML/CSS/JavaScript Mines client served by Nginx. It uses Basic Auth only after the API returns `401`; credentials are stored in browser `localStorage` under `rainbet-mines-auth`.
- `frontend/nginx.conf` proxies `/api/` to the Compose service named `backend`; the frontend must not make direct backend-origin requests.
