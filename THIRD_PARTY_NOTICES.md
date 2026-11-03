# Third-Party Notices

NodePulse is distributed under GPL-3.0. The Go backend and Flutter desktop UI
are combined into one GPL-3.0 work. Individual files originally written for
NodePulse may still carry Apache-2.0 headers; Apache-2.0 is compatible with
GPL-3.0.

## FlClash

- Project: FlClash, <https://github.com/chen08209/FlClash>
- Copyright (c) FlClash contributors
- License: GNU General Public License v3.0
- Reused: design-system values and files under `desktop/lib/ui/flclash/`
  (color scheme, typography, metrics) and layout/component patterns adapted
  from FlClash's `lib/ui`.
- Modified for NodePulse: package names, imports, widget shell, state and data
  bindings.

FlClash is a separate project; NodePulse is not affiliated with it.

## Flutter

- Project: Flutter, <https://flutter.dev>
- Copyright 2014 The Flutter Authors
- License: BSD-3-Clause
- Used as the desktop UI toolkit for Windows, macOS and Linux.

## Flutter Material UI

- Package: `material_ui` (official Flutter material library)
- Copyright 2014 The Flutter Authors
- License: BSD-3-Clause
- Upstream: <https://github.com/flutter/packages>

## Material Color Utilities

- Package: `material_color_utilities`
- Copyright 2021 Google LLC
- License: Apache-2.0
- Used to generate the FlClash-style light and dark color schemes.

## Dart and Flutter packages

The desktop app uses the following packages through `desktop/pubspec.yaml`:

- `http` (BSD-3-Clause)
- `window_manager` (MIT)
- `tray_manager` (MIT)
- `launch_at_startup` (MIT)
- `file_picker` (MIT)
- `path` (BSD-3-Clause)
- `path_provider` (BSD-3-Clause)
- `shared_preferences` (BSD-3-Clause)

Their full license texts are available from pub.dev and in the Flutter
package cache used to build the app.
