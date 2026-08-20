# Agent Notes

## Commands

- Run the API from the repository root with `task run`; Task executes it from `backend`.
- Run `task check` before handoff; it runs `go vet ./...` followed by `go test ./...` inside `backend`.
- Run the route tests only with `go -C backend test ./internal/router -run 'Test(Mines|CreateMinesBet)'`.
- Run the HTTP E2E suite with `task e2e`; it starts an in-process TCP server with a temporary SQLite database.
- `task build` writes `bin/rainbet-api`; `bin/` is ignored.
- Run the complete application with `docker compose up --build`; the frontend is available at `http://localhost:8080` and proxies `/api` to the backend.
- `task compose` runs the full stack in the background. `task tunnel` starts it and prints a temporary public Cloudflare Quick Tunnel URL; press `Ctrl+C` to stop only the tunnel.
- Stop the Compose stack with `docker compose down`; its `rainbet-data` volume keeps the SQLite database between runs.
- Inspect or mutate the Compose database with `docker compose exec backend sqlite3 /data/rainbet.db`.

## Backend

- `backend` is the only Go module (`go 1.22`); `backend/cmd/api/main.go` is the executable entrypoint.
- `internal/router` registers routes and wraps them with `middleware.BasicAuth`; provide a `middleware.Authenticator` implementation rather than raw credentials.
- `middleware.BasicAuth` stores the authenticated user ID in the request context; handlers should use `middleware.UserIDFromContext` instead of parsing Basic Auth again.
- `internal/database` creates the SQLite `users` table and seeds `user` / `user` with `balanceDollars = 10000` ($100.00 in cents), a random private `serverSeed`, and `transactionNumber = 0` only when that username is absent; passwords are bcrypt hashes. `internal/user.Store` performs the database-backed credential check.
- `games` uses `userId` as a foreign key to `users`, `betAmount BIGINT` in cents, `gridSize` as the total cell count, `mines`, `demo`, JSON text in `openedCells`, statuses `inProcess`, `cachedOut`, or `failed`, game seed inputs/nonce, and `startedAt`.
- SQLite uses the CGO driver `github.com/mattn/go-sqlite3` with foreign key enforcement enabled. `DATABASE_PATH` overrides the file path; under `task run`, the default is the ignored `backend/rainbet.db`.
- `GET /api/user` returns the authenticated user's `balance` as a two-decimal dollar string.
- `POST /api/user/balance` is an unauthenticated test endpoint that accepts `username` and dollar `amount`, then updates the named user's balance without ownership or amount checks.
- `POST /api/mines/bets` accepts dollar `betAmount` values with at most two decimals and only permits `gridSize` 25, 36, 49, or 64. It creates an `inProcess` game, copies the user's seed and current nonce, and increments the user's transaction number in the same database transaction. Real games deduct a positive bet; demo games require `betAmount = 0` and do not change the balance.
- `POST /api/mines/bets/{gameID}/moves` opens a zero-based `cellIndex`; `POST /api/mines/bets/{gameID}/cashout` only succeeds for an active game with at least one diamond. Cashout payouts use the exact coefficient calculation, round down to cents, and return `payout` as a two-decimal dollar string.

## Frontend

- `frontend` is a static HTML/CSS/JavaScript Mines client served by Nginx. It uses Basic Auth only after the API returns `401`; credentials are stored in browser `localStorage` under `rainbet-mines-auth`.
- `frontend/nginx.conf` proxies `/api/` to the Compose service named `backend`; the frontend must not make direct backend-origin requests.
