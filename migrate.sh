#!/usr/bin/env bash
set -e

show_usage() {
    echo "Usage: $0 -p <protocol> -u <database_url> [-d <direction>] [-t <auth_token>] [-s <steps>] [--cgo]"
    echo ""
    echo "Flags:"
    echo "  -p, --protocol     Database protocol (required: sqlite, sqlite3, libsql, postgres, etc.)"
    echo "  -u, --url          Database URL without protocol (required)"
    echo "  -d, --direction    Migration direction: up (default) or down"
    echo "  -t, --token        Authentication token for remote databases"
    echo "  -s, --steps        Number of steps for down migration (default: 1)"
    echo "  --cgo              Enable CGo (required for sqlite3/mattn and other CGo-based drivers)"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -p sqlite -u ./db.sqlite3"
    echo "  $0 -p sqlite -u ./db.sqlite3 -d up"
    echo "  $0 -p sqlite3 -u ./db.sqlite3 --cgo"
    echo "  $0 -p libsql -u mydb.aws-us-east-1.turso.io -d up -t your_token"
    echo "  $0 --protocol postgres --url localhost:5432/mydb --direction down --steps 3"
}

# Initialize variables
PROTOCOL=""
DB_URL=""
DIRECTION="up"
AUTH_TOKEN=""
STEPS="1"
CGO="0"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--protocol)
            PROTOCOL="$2"
            shift 2
            ;;
        -u|--url)
            DB_URL="$2"
            shift 2
            ;;
        -d|--direction)
            DIRECTION="$2"
            shift 2
            ;;
        -t|--token)
            AUTH_TOKEN="$2"
            shift 2
            ;;
        -s|--steps)
            STEPS="$2"
            shift 2
            ;;
        --cgo)
            CGO="1"
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Error: Unknown option $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [ -z "$PROTOCOL" ]; then
    echo "Error: Protocol is required (use -p or --protocol)"
    show_usage
    exit 1
fi

if [ -z "$DB_URL" ]; then
    echo "Error: Database URL is required (use -u or --url)"
    show_usage
    exit 1
fi

# Validate direction
if [[ "$DIRECTION" != "up" && "$DIRECTION" != "down" ]]; then
    echo "Error: Direction must be 'up' or 'down'"
    show_usage
    exit 1
fi

# Validate protocol-specific requirements
if [ "$PROTOCOL" == "libsql" ] && [ -z "$AUTH_TOKEN" ]; then
    echo "Error: Auth token is required for libsql protocol (use -t or --token)"
    show_usage
    exit 1
fi

dir="$(cd "$(dirname "$0")" && pwd)"
migrations_source="file://${dir}/internal/db/migrations"

run_migration() {
    echo "Running migration $DIRECTION with $PROTOCOL://$DB_URL"

    if [ "$PROTOCOL" == "libsql" ]; then
        # Uses a dedicated tool for libsql/Turso.
        # Add to go.mod with: go get -tool github.com/Piszmog/migrate-libsql
        if [ "$DIRECTION" == "down" ]; then
            go run github.com/Piszmog/migrate-libsql \
                -url "$PROTOCOL://$DB_URL" \
                -token "$AUTH_TOKEN" \
                -migrations "${dir}/internal/db/migrations" \
                -direction down \
                -steps "$STEPS"
        else
            go run github.com/Piszmog/migrate-libsql \
                -url "$PROTOCOL://$DB_URL" \
                -token "$AUTH_TOKEN" \
                -migrations "${dir}/internal/db/migrations" \
                -direction up
        fi
    else
        # Build the database URL.
        # sqlite (pure-Go modernc) requires an absolute path: sqlite://./rel fails
        # because '.' is parsed as the host component of the URL.
        local db_url
        if [ "$PROTOCOL" == "sqlite" ]; then
            raw="${DB_URL#file:}"
            raw_dir="$(dirname "$raw")"
            if [ ! -d "$raw_dir" ]; then
                echo "Error: directory '$raw_dir' does not exist (from database URL '$DB_URL')"
                exit 1
            fi
            abs="$(cd "$raw_dir" && pwd)/$(basename "$raw")"
            db_url="sqlite://${abs}"
        elif [ -n "$AUTH_TOKEN" ]; then
            db_url="${PROTOCOL}://${DB_URL}?authToken=${AUTH_TOKEN}"
        else
            db_url="${PROTOCOL}://${DB_URL}"
        fi

        # The migrate CLI registers its DB driver via build tag.
        # The tag name matches the protocol (sqlite, sqlite3, postgres, mysql, …).
        # CGO is disabled by default; pass --cgo for drivers that require it (e.g. sqlite3/mattn).
        # go tool does not accept -tags, so go run is used directly.
        if [ "$DIRECTION" == "down" ]; then
            CGO_ENABLED=$CGO go run -tags "$PROTOCOL" \
                github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
                -source "$migrations_source" \
                -database "$db_url" \
                down "$STEPS"
        else
            CGO_ENABLED=$CGO go run -tags "$PROTOCOL" \
                github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
                -source "$migrations_source" \
                -database "$db_url" \
                up
        fi
    fi

    echo "Migration completed successfully"
}

echo "Starting database migration..."
run_migration
