# Third-Party Notices

NodePulse includes the following third-party software.

## Material Color Utilities

- Package: `@material/material-color-utilities` 0.3.0
- Copyright 2021 Google LLC
- License: Apache License 2.0
- File: `internal/uiassets/dist/material-color-utilities.js`
- Bundled from jsDelivr (`+esm` build) for offline use.
- License text: `internal/uiassets/dist/material-color-utilities.LICENSE.txt`

Used to generate a complete Material Design 3 light/dark color scheme from
the user-selected seed color.

## Preact

- Package: `preact` 10.26.9
- Copyright 2015-present Jason Miller
- License: MIT
- Files: `internal/uiassets/dist/vendor/preact.module.js`,
  `internal/uiassets/dist/vendor/hooks.module.js`
- Upstream: <https://github.com/preactjs/preact>
- Modification: the `hooks.module.js` bare import `preact` was rewritten to
  the relative `./preact.module.js` so the app runs without a bundler.
- License text: `internal/uiassets/dist/vendor/LICENSES/preact-LICENSE.txt`

Used for the component/state layer of the desktop and web UI.

## htm

- Package: `htm` 3.1.1
- Copyright 2018 Jason Miller
- License: Apache License 2.0
- File: `internal/uiassets/dist/vendor/htm.module.js`
- Upstream: <https://github.com/developit/htm>
- License text: `internal/uiassets/dist/vendor/LICENSES/htm-LICENSE.txt`

Used for the tagged-template view syntax.

## Floating UI

- Package: `@floating-ui/core` 1.7.3 and `@floating-ui/dom` 1.7.4
- Copyright 2021-present Floating UI contributors
- License: MIT
- Files: `internal/uiassets/dist/vendor/floating-ui.core.umd.min.js`,
  `internal/uiassets/dist/vendor/floating-ui.dom.umd.min.js`
- Upstream: <https://github.com/floating-ui/floating-ui>
- License text: `internal/uiassets/dist/vendor/LICENSES/floating-ui-LICENSE.txt`

Used to position the Material 3 select menu with viewport flipping and
shifting.

## Material Symbols

- Package: `@material-symbols/svg-400` 0.48.2 (Rounded, filled subset)
- Copyright Google LLC
- License: Apache License 2.0
- Files: `internal/uiassets/dist/icons.svg`,
  `internal/uiassets/dist/favicon.svg`,
  `internal/uiassets/icons/src/rounded/*.svg`
- Upstream: <https://github.com/google/material-design-icons>
- Modification: a pinned subset of 21 icons, normalised into a single SVG
  sprite by `internal/uiassets/genicons`; no runtime font or CDN is used.
- License text: `internal/uiassets/dist/icons.LICENSE.txt`

Used for all UI icons.

## NodePulse

NodePulse itself is licensed under Apache-2.0; see `LICENSE`.
