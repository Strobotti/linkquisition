# Changelog

## [3.0.5](https://github.com/Strobotti/linkquisition/compare/v3.0.4...v3.0.5) (2026-07-17)


### Bug Fixes

* **ci:** commit generated Wayland headers to fix GoReleaser dirty state ([#162](https://github.com/Strobotti/linkquisition/issues/162)) ([e595103](https://github.com/Strobotti/linkquisition/commit/e5951030fe51c0f2330549803569f59ef85c9b8a))

## [3.0.4](https://github.com/Strobotti/linkquisition/compare/v3.0.3...v3.0.4) (2026-07-17)


### Bug Fixes

* **ci:** use /NOCD flag so makensis resolves paths from repo root ([#160](https://github.com/Strobotti/linkquisition/issues/160)) ([e697ac3](https://github.com/Strobotti/linkquisition/commit/e697ac3c7d74277b516bbf5a03dd0ba7a08c9a51))

## [3.0.3](https://github.com/Strobotti/linkquisition/compare/v3.0.2...v3.0.3) (2026-07-17)


### Bug Fixes

* **ci:** convert Icon.png to .ico for NSIS installer ([#158](https://github.com/Strobotti/linkquisition/issues/158)) ([1e19cbf](https://github.com/Strobotti/linkquisition/commit/1e19cbf2bfeef07ea0fb025d44745cecf7bdb02b))

## [3.0.2](https://github.com/Strobotti/linkquisition/compare/v3.0.1...v3.0.2) (2026-07-17)


### Bug Fixes

* **ci:** add NSIS to PATH after Chocolatey install ([#156](https://github.com/Strobotti/linkquisition/issues/156)) ([00209ba](https://github.com/Strobotti/linkquisition/commit/00209ba8f499b6a95c15ea2776755f34d23c8679))

## [3.0.1](https://github.com/Strobotti/linkquisition/compare/v3.0.0...v3.0.1) (2026-07-17)


### Bug Fixes

* safety report link and add bug reporting section to About tab ([#154](https://github.com/Strobotti/linkquisition/issues/154)) ([2b1b31a](https://github.com/Strobotti/linkquisition/commit/2b1b31a212ce18e242fd8c4c398b7424df82a5f9))

## [3.0.0](https://github.com/Strobotti/linkquisition/compare/v2.14.0...v3.0.0) (2026-07-14)


### ⚠ BREAKING CHANGES

* None — existing Linux and macOS behavior unchanged.

### Features

* add Windows support (portable + installer) ([#153](https://github.com/Strobotti/linkquisition/issues/153)) ([58b067e](https://github.com/Strobotti/linkquisition/commit/58b067e25b1ba91c11e55f821f350595ce97e40c))


### Bug Fixes

* **ci:** stage generated Wayland protocol headers before GoReleaser ([b28ba2f](https://github.com/Strobotti/linkquisition/commit/b28ba2f929d96876c76e4f2dea675a6da7161d5f))

## [2.14.0](https://github.com/Strobotti/linkquisition/compare/v2.13.0...v2.14.0) (2026-07-13)


### Features

* Add KeyValueList setting type and improve unwrap plugin settings UI ([#146](https://github.com/Strobotti/linkquisition/issues/146)) ([c7d420e](https://github.com/Strobotti/linkquisition/commit/c7d420e9f6958974e7664715440537c2e506fdeb))
* Add manual update check and document undocumented features ([#147](https://github.com/Strobotti/linkquisition/issues/147)) ([0bcbba9](https://github.com/Strobotti/linkquisition/commit/0bcbba95ac9ac799df43c367a99ce66ce92d45ec))
* Add report link to safety check popup & document security feature ([#140](https://github.com/Strobotti/linkquisition/issues/140)) ([5efc307](https://github.com/Strobotti/linkquisition/commit/5efc307cb6dd8840145d348cd2299b4846f62867))
* Add security check result caching and improve safety report UX ([#143](https://github.com/Strobotti/linkquisition/issues/143)) ([a401088](https://github.com/Strobotti/linkquisition/commit/a401088f2e48c3e25df91e2c991efc094c149bc3))
* add site/domain selection to remember checkbox and improve rule editor ([#148](https://github.com/Strobotti/linkquisition/issues/148)) ([5b0d9ec](https://github.com/Strobotti/linkquisition/commit/5b0d9ecdd3505727adad6d04d2117734718fd3a5))
* consistent link UI with clipboard copy button ([#145](https://github.com/Strobotti/linkquisition/issues/145)) ([10a7d1f](https://github.com/Strobotti/linkquisition/commit/10a7d1fb7ee725ab8deb48f4fb1f6f5356cda4ca))
* Improve UI uniformity and extract reusable components ([#149](https://github.com/Strobotti/linkquisition/issues/149)) ([0c5dd0e](https://github.com/Strobotti/linkquisition/commit/0c5dd0e866799c455857e1e7db0839e7c145e588))
* plugin settings UX improvements — defaults, metadata display, and reset button ([#151](https://github.com/Strobotti/linkquisition/issues/151)) ([bf7bb0f](https://github.com/Strobotti/linkquisition/commit/bf7bb0f9ffaea0a829d5eb40667e832a0542e3d2))
* Upgrade Fyne to 2.8.0 with full Wayland support ([#142](https://github.com/Strobotti/linkquisition/issues/142)) ([29086c4](https://github.com/Strobotti/linkquisition/commit/29086c4f5006c45dbed016cfae09979014ac8900))


### Bug Fixes

* hide "no browsers configured" warning on General tab after scanning ([#150](https://github.com/Strobotti/linkquisition/issues/150)) ([3d0b5dd](https://github.com/Strobotti/linkquisition/commit/3d0b5dd0af7593d5d9d47cf3fef7ed6537c8b5f3))
* improve whois error dialog layout and messages ([#144](https://github.com/Strobotti/linkquisition/issues/144)) ([0ce68e4](https://github.com/Strobotti/linkquisition/commit/0ce68e4db38ad2a381b165afa0d1e3472b2fb746))
* use valid Debian version number as default in packaging ([9e74163](https://github.com/Strobotti/linkquisition/commit/9e741636e7bb9a166f57ead8a7bc2b538b60eaf7))

## [2.13.0](https://github.com/Strobotti/linkquisition/compare/v2.12.0...v2.13.0) (2026-07-12)


### Features

* picker menu with QR code, Whois lookup, and URL safety checking ([#138](https://github.com/Strobotti/linkquisition/issues/138)) ([453efd3](https://github.com/Strobotti/linkquisition/commit/453efd3c8848b9ccae6f745fd5f2024fd3d071f2))

## [2.12.0](https://github.com/Strobotti/linkquisition/compare/v2.11.0...v2.12.0) (2026-07-11)


### Features

* favicon display in browser picker ([#133](https://github.com/Strobotti/linkquisition/issues/133)) ([f78c987](https://github.com/Strobotti/linkquisition/commit/f78c9877d7ef8fbad23897427ecc888849d9f07d))


### Bug Fixes

* **ci:** use explicit version for ossf/scorecard-action ([f19512f](https://github.com/Strobotti/linkquisition/commit/f19512fb6ad13252a45413d5d771d1bda927f690))

## [2.11.0](https://github.com/Strobotti/linkquisition/compare/v2.10.0...v2.11.0) (2026-07-10)


### Features

* add new shenanigans effects, --log-level flag, and split plugin into separate files ([#128](https://github.com/Strobotti/linkquisition/issues/128)) ([384f86d](https://github.com/Strobotti/linkquisition/commit/384f86d589a6548dff6a8a638e091ad2a6a98ebe))

## [2.10.0](https://github.com/Strobotti/linkquisition/compare/v2.9.1...v2.10.0) (2026-07-10)


### Features

* configurable UI theme and plugin UX improvements ([#125](https://github.com/Strobotti/linkquisition/issues/125)) ([46b63e0](https://github.com/Strobotti/linkquisition/commit/46b63e001a5a5f3e9e3c9e52d44502a6d84edc77))
* Redesign browsers tab UI and default to horizontal picker layout ([#126](https://github.com/Strobotti/linkquisition/issues/126)) ([8db11c1](https://github.com/Strobotti/linkquisition/commit/8db11c1c757d08f3967c5848514803b35a675506))


### Bug Fixes

* improve matrix rain effect rendering ([#123](https://github.com/Strobotti/linkquisition/issues/123)) ([ac8a1b8](https://github.com/Strobotti/linkquisition/commit/ac8a1b8981bab26ca83248c7a666eee47e7f0172))

## [2.9.1](https://github.com/Strobotti/linkquisition/compare/v2.9.0...v2.9.1) (2026-07-09)


### Bug Fixes

* **ci:** update coverage badge on every push to main ([#121](https://github.com/Strobotti/linkquisition/issues/121)) ([3569cff](https://github.com/Strobotti/linkquisition/commit/3569cff9da3d180d820770722396fa0bf2670d27))

## [2.9.0](https://github.com/Strobotti/linkquisition/compare/v2.8.0...v2.9.0) (2026-07-09)


### Features

* Add new visual effects to shenanigans plugin ([#120](https://github.com/Strobotti/linkquisition/issues/120)) ([8974336](https://github.com/Strobotti/linkquisition/commit/897433628b39b706e90bb8f6cd1a73ff57a26885))
* add translations for German, French, Hungarian, Ukrainian, and Portuguese ([#117](https://github.com/Strobotti/linkquisition/issues/117)) ([2733b1e](https://github.com/Strobotti/linkquisition/commit/2733b1eaea53f8ae7a05fbddc38a6e9d92d6b89c))

## [2.8.0](https://github.com/Strobotti/linkquisition/compare/v2.7.0...v2.8.0) (2026-07-08)


### Features

* Add copy button and improve URL readability in browser picker ([#115](https://github.com/Strobotti/linkquisition/issues/115)) ([1bc58d2](https://github.com/Strobotti/linkquisition/commit/1bc58d221cd246b7b25bbb6e2afa98c714254ece))

## [2.7.0](https://github.com/Strobotti/linkquisition/compare/v2.6.0...v2.7.0) (2026-07-08)


### Features

* add shenanigans plugin with visual effects for the browser picker ([#113](https://github.com/Strobotti/linkquisition/issues/113)) ([0379a13](https://github.com/Strobotti/linkquisition/commit/0379a1371dadc1e21e0fce8ee173c2e3012ac2ab))

## [2.6.0](https://github.com/Strobotti/linkquisition/compare/v2.5.5...v2.6.0) (2026-07-08)


### Features

* add configurable horizontal button layout for browser picker ([#111](https://github.com/Strobotti/linkquisition/issues/111)) ([8491760](https://github.com/Strobotti/linkquisition/commit/849176032640b32c02e0b5d8c8fc1cf6e26689ee))

## [2.5.5](https://github.com/Strobotti/linkquisition/compare/v2.5.4...v2.5.5) (2026-07-07)


### Bug Fixes

* **ci:** split asset uploads to prevent silent glob failures ([#109](https://github.com/Strobotti/linkquisition/issues/109)) ([cdece05](https://github.com/Strobotti/linkquisition/commit/cdece05ee08e7906692a44b4e261f9dd068807b2))

## [2.5.4](https://github.com/Strobotti/linkquisition/compare/v2.5.3...v2.5.4) (2026-07-07)


### Bug Fixes

* **ci:** remove explicit binary from nfpms contents (content collision) ([#106](https://github.com/Strobotti/linkquisition/issues/106)) ([ebcc468](https://github.com/Strobotti/linkquisition/commit/ebcc468e777931d6a5b735b9296e5dcde02623d7))
* **ci:** use gh CLI for asset uploads instead of GoReleaser publish ([#108](https://github.com/Strobotti/linkquisition/issues/108)) ([60da845](https://github.com/Strobotti/linkquisition/commit/60da845462086152fab3f0c7dd710e5cb5a7c9da))

## [2.5.3](https://github.com/Strobotti/linkquisition/compare/v2.5.2...v2.5.3) (2026-07-07)


### Bug Fixes

* **ci:** remove nfpms meta flag so arch resolves to amd64 ([#104](https://github.com/Strobotti/linkquisition/issues/104)) ([18c4e2e](https://github.com/Strobotti/linkquisition/commit/18c4e2e233c859012049097dbdc386667f7518b9))

## [2.5.2](https://github.com/Strobotti/linkquisition/compare/v2.5.1...v2.5.2) (2026-07-07)


### Bug Fixes

* **ci:** fix GoReleaser config version and dirty state error ([#102](https://github.com/Strobotti/linkquisition/issues/102)) ([b9947ce](https://github.com/Strobotti/linkquisition/commit/b9947ce540cdc2831ea010fc2d45e0b585d5ec56))

## [2.5.1](https://github.com/Strobotti/linkquisition/compare/v2.5.0...v2.5.1) (2026-07-07)


### Bug Fixes

* **ci:** Fix CI release workflow and move metainfo update to Release PR ([#100](https://github.com/Strobotti/linkquisition/issues/100)) ([5289d48](https://github.com/Strobotti/linkquisition/commit/5289d48d32b0c013fa869eec9ff98ca55811c22f))

## [2.5.0](https://github.com/Strobotti/linkquisition/compare/v2.4.0...v2.5.0) (2026-07-07)


### Features

* Fix browser icon loading for user-installed browsers ([#98](https://github.com/Strobotti/linkquisition/issues/98)) ([516bee2](https://github.com/Strobotti/linkquisition/commit/516bee26fe72d9dde43f791d113ee717103937ad))
