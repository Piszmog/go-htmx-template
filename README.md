# Go HTMX Template

This is a template repository for stiches together a number of technologies that you may use to server a website using Go and HTMX.

## Getting Started

TODO

## Technologies

A few different technologies are configured to help getting off the ground easier.

- [sqlc](https://sqlc.dev/) for database layer
  - Stubbed to use SQLite
- [Tailwind CSS](https://tailwindcss.com/) for styling
  - Output is generated with the [CLI](https://tailwindcss.com/docs/installation)
- [templ](https://templ.guide/) for creating HTML
- [HTMX](https://htmx.org/) for HTML interaction

Everything else uses the standard library.

## Structure

```text
.
├── Makefile
├── components
│   ├── core
│   │   └── html.templ
│   └── home
│       └── home.templ
├── db
│   ├── db.go
│   ├── models.go
│   ├── query.sql
│   ├── query.sql.go
│   ├── schema.go
│   ├── schema.sql
│   └── sqlite.go
├── db.sqlite3
├── dist
│   ├── assets
│   │   └── js
│   │       └── htmx@1.9.10.min.js
│   └── dist.go
├── go.mod
├── go.sum
├── logger
│   └── logger.go
├── main.go
├── server
│   ├── handler
│   │   ├── handler.go
│   │   └── home.go
│   ├── middleware
│   │   ├── cache.go
│   │   └── logging.go
│   ├── router
│   │   └── router.go
│   └── server.go
├── sqlc.yml
├── styles
│   └── input.css
├── tailwind.config.js
└── version
    └── version.go
```

### Components

This is where `templ` files live. Anything you want to render to the user goes here. Note, all
`*.go` files will be ignored by `git` (configured in `.gitignore`).

### DB

This is the directory that `sqlc` generates to. Update `schema.sql` and `queries.sql` to build 
your database.

### Dist

This is where your assets live. Any Javascript, images, or styling needs to go in the 
`dist/assets` directory. The directory will be embedded into the application.

Note, the `dist/assets/css` will be ignored by `git` (configured in `.gitignore`) since the 
files that are written to this directory are done by the Tailwind CSS CLI. Custom styles should
go in the `styles/input.css` file.

### Logger

This contains helper function to create a `slog.Logger`.

### Server

This contains everything related to the HTTP server.

#### Router

This package sets up the routing for the application, such as the `/assets/` path and `/` path.

#### Middleware

This package contains any middleware to configured with routes.

#### Handler

This package contains the handler to handle the actual routes.

#### Version

This package allows you to set a version at build time. If not set, the version defaults to `dev`. To set the version run the following command,

```shell
go build -o ./app -ldflags="-X version.Value=1.0.0"
```

See the `Makefile` for building the application.

## Run

To run, you will need to compile tailwind and templ first prior to running the application. 
See the `Makefile` for helper commands to build everything.

You may also want to look into [air](https://github.com/cosmtrek/air) for performing live 
reloads of the application.

