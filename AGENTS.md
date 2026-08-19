# Agent Notes

## Commands

- Run the API from the repository root with `task run`; Task executes it from `backend`.
- Run `task check` before handoff; it runs `go vet ./...` followed by `go test ./...` inside `backend`.
- Run the route tests only with `go -C backend test ./internal/router -run TestCreateMinesBet`.
- `task build` writes `bin/rainbet-api`; `bin/` is ignored.

## Backend

- `backend` is the only Go module (`go 1.22`); `backend/cmd/api/main.go` is the executable entrypoint.
- `internal/router` registers routes and wraps them with `middleware.BasicAuth`; provide a `middleware.Authenticator` implementation rather than raw credentials.
- `internal/database` creates the SQLite `users` table and seeds `user` / `user` with `balanceDollars = 100` only when that username is absent; passwords are bcrypt hashes. `internal/user.Store` performs the database-backed credential check.
- SQLite uses the CGO driver `github.com/mattn/go-sqlite3`. `DATABASE_PATH` overrides the file path; under `task run`, the default is the ignored `backend/rainbet.db`.
- The current API is `POST /api/mines/bets`; it requires Basic Auth and intentionally returns `501` after strict JSON decoding. Router tests create a temporary SQLite database.
