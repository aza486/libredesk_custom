# Upgrade

!!! warning "Warning"
    Always take a backup of the Postgres database before upgrading Libredesk.

## Binary
- Stop running libredesk binary.
- Download the [latest release](https://github.com/abhinavxd/libredesk/releases) and extract the libredesk binary and overwrite the previous version.
- `./libredesk --upgrade` to upgrade an existing database schema. Upgrades are idempotent and running them multiple times have no side effects.
- Run `./libredesk` again.

## Docker

```shell
docker compose down app
docker compose pull
docker compose up app -d
```
