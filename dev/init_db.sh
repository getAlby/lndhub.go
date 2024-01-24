#!/usr/bin/env bash
set -x
set -eo
# check key dependencies for postgres
if ! [ -x "$(command -v psql)" ]; then
    echo >&2 "Error: psql is not install."
    exit 1
fi
# TODO check on the dependency `bun` that will handle migrations

# check for environment based user, or attempt with defaults
PG_USER="${DB_USER:=postgres}"
PG_PASSWORD="${DB_PASSWORD:=password}"
PG_PORT="${DB_PORT:=5432}"
PG_HOST="${DB_HOST:=localhost}"
PG_NAME="${DB_NAME:=tahub}"
# launching postgres using docker...
# allow to skip docker if a dockerized 
# postgres container is already running
if [[ -z "${SKIP_DOCKER}" ]]
then 
    docker run \
        -e POSTGRES_USER=${PG_USER} \
        -e POSTGRES_PASSWORD=${PG_PASSWORD} \
        -e POSTGRES_DB=${PG_NAME} \
        -p "${PG_PORT}":5432 \
        -d postgres
fi
# ping postgres until it is ready to accept connections
export PGPASSWORD="${PG_PASSWORD}"
until psql -h "${PG_HOST}" -U "${PG_USER}" -p "${PG_PORT}" -d "postgres" -c '\q'; do
    >&2 echo "Postgres is still unavailable - Sleeping..."
    sleep 1
done
# postgres is ready
>&2 echo "Postgres is running on port ${PG_PORT}"
# create
psql -h "${PG_HOST}" -U "${PG_USER}" -tc "SELECT 1 FROM pg_database WHERE datname = '${PG_NAME}'" |
    grep -q 1 || \
    psql -U "${PG_USER}" -p "${PG_PORT}" -c "CREATE DATABASE '${PG_NAME}'"
# TODO run migrations in /db/models/migrations using bun
# run migrations
#go run ../db/