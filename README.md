<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="frontend/originalMedia/brand/lockup-white.svg">
    <img alt="Fenster" src="frontend/originalMedia/brand/lockup-static.svg" width="320">
  </picture>
</p>

<p align="center">
  A calm, self-hosted to-do app with a liquid-glass interface and a real mobile app feel.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: AGPL-3.0-or-later" src="https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg"></a>
  <a href="https://github.com/MBeggiato/fenster/pkgs/container/fenster"><img alt="Docker" src="https://img.shields.io/badge/docker-ghcr.io%2Fmbeggiato%2Ffenster-1973ff?logo=docker&logoColor=white"></a>
</p>

<img alt="Fenster desktop home screen" src="docs/screenshots/desktop-home.png" width="100%">

## Features

- **Multiple project views**: organize tasks in list, table, kanban, or Gantt views
- **Smart task creation**: parse dates, labels, and priority from natural language in quick-add
- **Eisenhower matrix**: categorize tasks by urgency and importance
- **Pomodoro focus timer**: focus sessions with breaks, per-task estimates and statistics
- **Installable PWA**: install on any device for a native app experience with bottom tab navigation on mobile
- **CalDAV sync**: sync tasks with calendar apps and other CalDAV clients
- **Dark mode**: switch themes automatically or manually
- **Import from other services**: migrate tasks from Todoist, Trello, Microsoft To Do, Planka, Wekan, TickTick, CSV and Vikunja exports
- **REST API**: build custom integrations with full API access

## Screenshots

### Desktop

| | |
|:---:|:---:|
| ![Project list](docs/screenshots/desktop-list.png "Project list") | ![Kanban board](docs/screenshots/desktop-kanban.png "Kanban board") |
| ![Task details](docs/screenshots/desktop-task.png "Task details") | ![Dark mode](docs/screenshots/desktop-dark.png "Dark mode") |

### Mobile

<table>
  <tr>
    <td align="center"><img alt="Home" src="docs/screenshots/mobile-home.png" width="160"><br><sub>Home</sub></td>
    <td align="center"><img alt="Upcoming" src="docs/screenshots/mobile-upcoming.png" width="160"><br><sub>Upcoming</sub></td>
    <td align="center"><img alt="Eisenhower matrix" src="docs/screenshots/mobile-eisenhower.png" width="160"><br><sub>Eisenhower matrix</sub></td>
    <td align="center"><img alt="Task" src="docs/screenshots/mobile-task.png" width="160"><br><sub>Task</sub></td>
    <td align="center"><img alt="Focus" src="docs/screenshots/mobile-focus.png" width="160"><br><sub>Focus</sub></td>
  </tr>
</table>

## Quick Start (Docker)

The container runs as user `1000`, so create the data folders first and hand them to that user:

```bash
mkdir -p fenster/db fenster/files && cd fenster
sudo chown 1000 db files
docker run -p 3456:3456 -v $PWD/db:/db -v $PWD/files:/app/vikunja/files \
  -e VIKUNJA_SERVICE_PUBLICURL=http://localhost:3456 \
  -e VIKUNJA_SERVICE_JWTSECRET=change-me \
  ghcr.io/mbeggiato/fenster:latest
```

Then open http://localhost:3456 and register the first account.

**Docker Compose** (same folders):

```yaml
services:
  fenster:
    image: ghcr.io/mbeggiato/fenster:latest
    ports:
      - "3456:3456"
    volumes:
      - ./db:/db
      - ./files:/app/vikunja/files
    environment:
      VIKUNJA_SERVICE_PUBLICURL: https://tasks.example.com
      VIKUNJA_SERVICE_JWTSECRET: change-me
    restart: unless-stopped
```

Configuration uses the upstream `VIKUNJA_*` variables, see [vikunja.io/docs/config-options](https://vikunja.io/docs/config-options/).

## Build from Source

Source: https://github.com/MBeggiato/fenster

```bash
git clone https://github.com/MBeggiato/fenster.git
cd fenster
docker build -t fenster .
```

For build and development commands, see [CONTRIBUTING.md](CONTRIBUTING.md).
Upstream documentation (may differ from Fenster in places): [vikunja.io/docs](https://vikunja.io/docs/).

## Regenerating the screenshots

Screenshots are generated with demo data by `frontend/scripts/readme-screenshots/run.sh`.

## Based on Vikunja

Fenster is a soft fork of [Vikunja](https://github.com/go-vikunja/vikunja) ([vikunja.io](https://vikunja.io)).
Thanks to the upstream maintainers and all Vikunja contributors, whose work this project builds on.
Fenster is independent and not affiliated with or endorsed by the Vikunja project. See [NOTICE](NOTICE).

## Merging Upstream Fixes

```bash
git remote add upstream https://github.com/go-vikunja/vikunja.git
git fetch upstream
git merge upstream/main
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security Reports

Please report security issues privately through the
[GitHub security advisories](https://github.com/MBeggiato/fenster/security/advisories/new) of this repository.

## License

Licensed under [AGPL-3.0-or-later](LICENSE). Original work Copyright 2018-present Vikunja and contributors;
modifications Copyright 2026 Marcel Beggiato. See [NOTICE](NOTICE).

### Unsplash Images

Background images from Unsplash are distributed under the [Unsplash License](https://unsplash.com/license). The license requires giving credit to the photographer and Unsplash. See [Unsplash's terms](https://unsplash.com/terms). Other third-party notices are in [NOTICE](NOTICE).
