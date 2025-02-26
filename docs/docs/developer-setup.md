# Developer Setup

Libredesk is a monorepo with a Go backend and a Vue.js frontend. The frontend uses Shadcn for UI components.

### Pre-requisites

- `go`
- `nodejs` (if you are working on the frontend) and `pnpm`
- Postgres database (>= 13)

### First time setup

Clone the repository:

```sh
git clone https://github.com/abhinavxd/libredesk.git
```

1. Copy `config.toml.sample` as `config.toml` and add your config.
2. Run `make` to build the libredesk binary. Once the binary is built, run `./libredesk --install` to run the DB setup and set the System user password.

### Running the Dev Environment

1. Run `make run-backend` to start the libredesk backend dev server on `:9000`.
2. Run `make run-frontend` to start the Vue frontend in dev mode using pnpm on `:8000`. Requests are proxied to the backend running on `:9000` check `vite.config.js` for the proxy config.

---

# Production Build

Run `make` to build the Go binary, build the Javascript frontend, and embed the static assets producing a single self-contained binary, `libredesk`.