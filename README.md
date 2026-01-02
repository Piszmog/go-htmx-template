# Go + HTMX Template

This is an _opinionated_ template repository that comes with everything you need to build a Web Application using Go (with templ) and HTMX. 

The template comes with a basic structure of using a SQL DB (`sqlc`), E2E testing (playwright), and styling (tailwindcss).

## Getting Started

In the top right, select the dropdown __Use this template__ and select __Create a new repository__.

Once cloned, run the `update_module.sh` script to change the module to your module name.

```shell
./update_module "github.com/me/my-new-module"
```

Once ran, delete the script.

Then you can upgrade your module dependencies with

```shell
go mod tidy
go get -u ./...
go get -u tool
# To upgrade test dependencies, run
go get -u all
```

Finally, you can proceed to generate sqlc and templ files

```shell
go tool sqlc generate
go tool templ generate -path ./internal/components
```

## Run

`air` is the primary way to run the applications for local development. It watches for file changes. When a file changes, it will rebuild and re-run the application.

When the application is running, go to http://localhost:8080/

### Prerequisites

- Install [air](https://github.com/air-verse/air#installation)

`templ`, `sqlc`, and `tailwindcss` (via [`go-tw`](https://github.com/Piszmog/go-tw)) are included as `go tool` directives. When running
the application for the first time, it may take a little time as `templ`, `sqlc` and `go-tw` are being downloaded and installed.

### air

`air` has been configured with the file `.air.toml` to allow live reloading of the application 
when a file changes.

To run, install `air`

```shell
go install github.com/air-verse/air@latest
```

Then simply run the command

```shell
air
```

## Technologies

A few different technologies are configured to help getting off the ground easier.

- [sqlc](https://sqlc.dev/) for database layer
  - Stubbed to use SQLite
  - This can be easily swapped with [sqlx](https://jmoiron.github.io/sqlx/)
- [Tailwind CSS](https://tailwindcss.com/) for styling
  - Output is generated with the [CLI](https://tailwindcss.com/docs/installation/tailwind-cli)
- [templ](https://templ.guide/) for creating HTML
- [HTMX](https://htmx.org/) for HTML interaction
  - The script `upgrade_htmx.sh` is available to make upgrading easier
  - Already included in this template
- [air](https://github.com/air-verse/air) for live reloading of the application.
- [golang migrate](https://github.com/golang-migrate/migrate) for DB migrations.
- [playwright-go](https://github.com/playwright-community/playwright-go) for E2E testing.

## Structure

```text
.
├── .air.toml
├── .github
│   └── workflows
│       ├── ci.yml
│       └── release.yml
├── .gitignore
├── .goreleaser.yaml
├── Dockerfile
├── cmd
│   └── server
│       └── main.go
├── internal
│   ├── components
│   │   ├── core
│   │   │   └── html.templ
│   │   └── home
│   │       └── home.templ
│   ├── db
│   │   ├── db.go
│   │   ├── local.go
│   │   ├── migrations
│   │   │   ├── 20240407203525_init.down.sql
│   │   │   └── 20240407203525_init.up.sql
│   │   └── queries
│   │       └── query.sql
│   ├── dist
│   │   ├── assets
│   │   │   └── js
│   │   │       └── htmx@2.0.4.min.js
│   │   └── dist.go
│   ├── log
│   │   └── log.go
│   ├── server
│   │   ├── handler
│   │   │   ├── handler.go
│   │   │   └── home.go
│   │   ├── middleware
│   │   │   ├── cache.go
│   │   │   ├── logging.go
│   │   │   └── middleware.go
│   │   ├── router
│   │   │   └── router.go
│   │   └── server.go
│   └── version
│       └── version.go
├── e2e
│   ├── e2e_test.go
│   ├── home_test.go
│   └── testdata
│       └── seed.sql
├── styles
│   └── input.css
├── go.mod
├── go.sum
├── sqlc.yml
├── update_module.sh
└── upgrade_htmx.sh
```

### Agents

At the root of the project is the file `AGENTS.md`. It is designed to help agents better understand the project and help you in your development.

### cmd/

This directory contains the application entrypoints. The `server/` subdirectory contains `main.go` which starts the HTTP server. This follows Go's standard project layout for applications.

### internal/

All application implementation code lives in the `internal/` directory. This prevents external packages from importing implementation details and follows the official Go project layout for server applications as documented at [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout).

### Components

This is where `templ` files live in `internal/components/`. Anything you want to render to the user goes here. Note, all `*.go` files will be ignored by `git` (configured in `.gitignore`).

### DB

This is the directory in `internal/db/` that `sqlc` generates to. Update `queries.sql` to build 
your database operations.

This project uses [golang migrate](https://github.com/golang-migrate/migrate) for DB 
migrations. `sqlc` uses the `internal/db/migrations` directory to generate DB tables. `cmd/server/main.go` calls `db.Migrate(..)` to automatically migrate the DB. To add migration
call the following command,

```shell
migrate create -ext sql -dir internal/db/migrations <name of migration>
```

#### Example Connection to Turso

If you want to connect to a remote Database, like [Turso](https://turso.tech/), you can create a struct that implements `Database`.

```golang
package db

import (
	"database/sql"
	"log/slog"

	"go-htmx-template/internal/db/queries"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type RemoteDB struct {
	logger  *slog.Logger
	db      *sql.DB
	queries *queries.Queries
}

var _ Database = (*RemoteDB)(nil)

func (d *RemoteDB) DB() *sql.DB {
	return d.db
}

func (d *RemoteDB) Queries() *queries.Queries {
	return d.queries
}

func (d *RemoteDB) Logger() *slog.Logger {
	return d.logger
}

func (d *RemoteDB) Close() error {
	return d.db.Close()
}

func newRemoteDB(logger *slog.Logger, name string, token string) (*RemoteDB, error) {
	db, err := sql.Open("libsql", "libsql://"+name+".turso.io?authToken="+token)
	if err != nil {
		return nil, err
	}
	return &RemoteDB{logger: logger, db: db, queries: queries.New(db)}, nil
}
```

### Dist

This is where your assets live in `internal/dist/`. Any Javascript, images, or styling needs to go in the 
`internal/dist/assets` directory. The directory will be embedded into the application.

Note, the `internal/dist/assets/css` will be ignored by `git` (configured in `.gitignore`) since the 
files that are written to this directory are done by the Tailwind CSS CLI. Custom styles should
go in the `styles/input.css` file at the root level.

### E2E

To test the UI, the `e2e` directory contains the Go tests for performing End to end testing. To
run the tests, run the command

```shell
go test -v ./... -tags=e2e
```

The end to end tests, will start up the app, on a random port, seeding the database using the 
`seed.sql` file. Once the tests are complete, the app will be stopped.

The E2E tests use Playwright (Go) for better integration into the Go tooling.

### Log

This contains helper function to create a `slog.Logger`. Log level and output type can be set
with then environment variables `LOG_LEVEL` and `LOG_OUTPUT`. The logger will write to 
`stdout`.

### Server

This contains everything related to the HTTP server in `internal/server/`. It comes with a graceful shutdown handler
that handles `SIGINT`.

#### Router

This package sets up the routing for the application, such as the `/assets/` path and `/` path.
It uses the standard library's mux for routing. You can easily swap out for other HTTP 
routers such as [gorilla/mux](https://github.com/gorilla/mux).

#### Middleware

This package contains any middleware to be configured with routes.

#### Handler

This package contains the handlers to handle the actual routes.

### Styles

This contains the `input.css` at the root level that the Tailwind CSS CLI uses to generate your output CSS. 
Update `input.css` with any custom CSS you need and it will be included in the output CSS.

### Version

This package in `internal/version/` allows you to set a version at build time. If not set, the version defaults to 
`dev`. To set the version run the following command,

```shell
go build -o ./app -ldflags="-X go-htmx-template/internal/version.Value=1.0.0" ./cmd/server
```

## Github Workflow

The repository comes with two Github workflows as well. One called `ci.yml` that lints and 
tests your code. The other called `release.yml` that creates a tag, GitHub Release, run [GoReleaser](https://goreleaser.com/) to build and 
attach all the binaries, and published the docker image. See release [v1.0.2](https://github.com/Piszmog/go-htmx-template/releases/tag/v1.0.2) as an example.

