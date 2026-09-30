# Migrating from Vikunja

Fenster started as a fork of Vikunja and has been developed independently since September 2026. You can move an existing Vikunja instance to Fenster in one of two ways.

| | Path A: swap the image | Path B: export and import |
|---|---|---|
| What moves | the whole instance: all users, teams, shares, settings, files | one user's projects, tasks, comments, attachments |
| Effort | change one line, restart | every user exports and imports on their own |
| Use it when | you run a single instance and want everything | Path A causes problems, or you only want your own data |

**There is no way back.** On first start Fenster runs its own database migrations, so Vikunja can't use the database afterwards. Make a backup first.

## Before you start: back up

Stop Vikunja cleanly (`docker compose stop` or `docker stop`, not `docker rm -f`) and copy:

- the database: for SQLite, the whole `/db` volume, including `vikunja.db-wal` and `vikunja.db-shm` if they exist; for MySQL or PostgreSQL, a regular dump
- the files volume (`/app/vikunja/files` in the Vikunja image)
- your `config.yml`, if you use one

## Path A: swap the image

Fenster still reads the Vikunja names until **2027-03-31** ([#15](https://github.com/MBeggiato/fenster/issues/15)): `VIKUNJA_*` env vars, `vikunja.db`, the `/app/vikunja/files` mount, and config files in `/etc/vikunja` or `~/.config/vikunja`. So the switch is:

```diff
 services:
   app:
-    image: vikunja/vikunja
+    image: ghcr.io/mbeggiato/fenster:latest
```

Start it and check the log. Expect:

- `Ran all migrations successfully.`
- one warning per old name still in use, for example `VIKUNJA_SERVICE_PUBLICURL is deprecated, use FENSTER_SERVICE_PUBLICURL`. Everything works; the warnings are a to-do list for the rename below.
- possibly `The database contains N migration(s) this Fenster version does not know`. Your Vikunja was newer than the point where Fenster forked. Usually this is harmless: these migrations have already run on your data and Fenster ignores them. If something doesn't work, restore the backup and use Path B.

Then log in and check that your projects, tasks and attachments are there.

### Rename to the Fenster names

Do this any time before 2027-03-31; after that date the old names stop working.

1. Env vars: in your compose file or `.env`, run `sed -i 's/VIKUNJA_/FENSTER_/g' docker-compose.yml .env`.
2. Files volume: mount it at `/app/fenster/files` instead of `/app/vikunja/files`.
3. SQLite database: stop the container cleanly, then rename all three files in the `/db` volume:
   ```bash
   for f in vikunja.db*; do mv "$f" "${f/vikunja.db/fenster.db}"; done
   ```
   Don't rename only `vikunja.db`: recent changes may still be in the `-wal` file.
4. Config file: move `/etc/vikunja/config.yml` to `/etc/fenster/config.yml`, or `~/.config/vikunja` to `~/.config/fenster`.
5. Webhooks: Fenster signs deliveries with `X-Fenster-Signature`. Until 2027-03-31 it also sends `X-Vikunja-Signature`, so update your receivers to the new header.
6. Restart. The log should show no more deprecation warnings.

## Path B: export and import

1. In Vikunja: user settings → data export. Download the zip once it's ready.
2. Start a fresh Fenster instance, see the [README](../README.md#quick-start-docker), and create your account.
3. In Fenster: Import from other services → "Fenster export (Vikunja format)", then upload the zip.

This moves your own projects, tasks, labels, comments and attachments. It does not move other users, teams, link shares, API tokens, webhooks or instance settings; set those up again.

## What's different in Fenster

- New interface (liquid glass) and a mobile app shell (PWA).
- Pomodoro timer and Eisenhower matrix.
- No license system: admin panel, time tracking, invite links and audit logs are always available.
- Configuration and headers use Fenster names (`FENSTER_*`, `X-Fenster-*`); see above.
- The Vikunja documentation at [vikunja.io/docs](https://vikunja.io/docs/) mostly applies. For config options, replace the `VIKUNJA_` prefix with `FENSTER_`.
