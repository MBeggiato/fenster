<!-- Logo: to be added once the Fenster logo exists. -->

[![License: AGPL-3.0-or-later](https://img.shields.io/badge/License-AGPL--3.0--or--later-blue.svg)](LICENSE)

# Fenster

> A self-hosted to-do app with a liquid-glass interface.

## Based on Vikunja

Fenster is a soft fork of [Vikunja](https://github.com/go-vikunja/vikunja) ([vikunja.io](https://vikunja.io)).
Thanks to the upstream maintainers and all Vikunja contributors, whose work this project builds on.
Fenster is independent and not affiliated with or endorsed by the Vikunja project. See [NOTICE](NOTICE).

## Get the source and build

Source: https://github.com/MBeggiato/fenster

```bash
git clone https://github.com/MBeggiato/fenster.git
cd fenster
docker build -t fenster .
```

A prebuilt image is published at `ghcr.io/mbeggiato/fenster`.

For build and development commands see [CONTRIBUTING.md](CONTRIBUTING.md).
Upstream documentation (may differ from Fenster in places): [vikunja.io/docs](https://vikunja.io/docs/).

## Merging upstream fixes

```bash
git remote add upstream https://github.com/go-vikunja/vikunja.git
git fetch upstream
git merge upstream/main
```

## Security reports

Please report security issues privately through the
[GitHub security advisories](https://github.com/MBeggiato/fenster/security/advisories/new) of this repository.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Licensed under [AGPL-3.0-or-later](LICENSE). Original work Copyright 2018-present Vikunja and contributors;
modifications Copyright 2026 Marcel Beggiato. See [NOTICE](NOTICE).

### Unsplash Images

Background images from Unsplash are distributed under the [Unsplash License](https://unsplash.com/license). The license requires giving credit to the photographer and Unsplash. See [Unsplash’s terms](https://unsplash.com/terms). Other third-party notices are in [NOTICE](NOTICE).
