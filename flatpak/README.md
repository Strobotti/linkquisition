# Flatpak / Flathub Packaging

This directory contains the files needed to package Linkquisition as a Flatpak
and submit it to [Flathub](https://flathub.org).

## Files

| File                                             | Purpose                                                 |
|--------------------------------------------------|---------------------------------------------------------|
| `io.github.strobotti.linkquisition.yml`          | Flatpak manifest (build recipe)                         |
| `io.github.strobotti.linkquisition.metainfo.xml` | AppStream metadata (description, screenshots, releases) |
| `flathub.json`                                   | Flathub build configuration (architecture limits)       |

## App ID

The Flatpak app ID is `io.github.strobotti.linkquisition` (following the
`io.github.<owner>.<repo>` convention for GitHub-hosted projects).

## Building locally

### Prerequisites

```bash
# Install Flatpak and flatpak-builder
sudo apt install flatpak flatpak-builder

# Add Flathub remote
flatpak remote-add --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo

# Install the SDK and Go extension
flatpak install flathub org.freedesktop.Platform//24.08
flatpak install flathub org.freedesktop.Sdk//24.08
flatpak install flathub org.freedesktop.Sdk.Extension.golang//24.08
```

### Vendor Go dependencies

Flathub builds have **no network access**, so Go modules must be vendored:

```bash
go mod vendor
git add vendor/
git commit -m "chore: vendor Go dependencies for Flatpak build"
```

### Build and test

```bash
# Build using org.flatpak.Builder (recommended)
flatpak install -y flathub org.flatpak.Builder
flatpak run --command=flathub-build org.flatpak.Builder --install \
    flatpak/io.github.strobotti.linkquisition.yml

# Or build with flatpak-builder directly
flatpak-builder --force-clean --user --install \
    build-dir flatpak/io.github.strobotti.linkquisition.yml

# Run
flatpak run io.github.strobotti.linkquisition
```

## Submitting to Flathub

The submission goes to a separate repository under the Flathub GitHub org.
The process is:

1. Ensure the latest release tag exists on GitHub
2. Update the `tag` and `commit` in the manifest's source
3. Update the `<releases>` section in the metainfo XML
4. Fork [flathub/flathub](https://github.com/flathub/flathub)
5. Create a branch from `new-pr` with:
    - `io.github.strobotti.linkquisition.yml` (manifest — at top level)
    - `flathub.json` (at top level)
6. Open a PR against the `new-pr` base branch

**Important:** The metainfo XML and desktop file must be in the upstream repo
(this repo), not in the Flathub submission. They are picked up from the git
source during the build.

## Updating releases

When publishing a new version:

1. Tag and push the release (handled by CI)
2. Update the Flathub manifest repo with the new tag + commit SHA
3. Add a `<release>` entry to the metainfo XML in this repo

## Permissions

The app requests:

- `--share=network` — needed to open URLs in browsers and for the unwrap plugin
- `--socket=fallback-x11` / `--socket=wayland` — GUI display
- `--device=dri` — OpenGL rendering (Fyne toolkit)
- `--share=ipc` — X11 shared memory
- D-Bus access for desktop integration and notifications
