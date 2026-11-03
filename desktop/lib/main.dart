// ignore_for_file: deprecated_member_use

import 'dart:io';

import 'package:flutter/services.dart';
import 'package:material_ui/material_ui.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:tray_manager/legacy.dart';
import 'package:window_manager/legacy.dart';

import 'app.dart';
import 'state/app_state.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await windowManager.ensureInitialized();
  await windowManager.setPreventClose(true);

  const windowOptions = WindowOptions(
    size: Size(1180, 780),
    minimumSize: Size(860, 600),
    center: true,
    title: 'NodePulse',
    backgroundColor: Colors.transparent,
  );

  final state = AppState();
  runApp(NodePulseApp(state: state));
  await windowManager.waitUntilReadyToShow(windowOptions, () async {
    await windowManager.show();
    await windowManager.focus();
  });

  final controller = _WindowController(state);
  windowManager.addListener(controller);
  await _TrayController(state).install();
  void applyTitleBar() {
    final bright = switch (state.themeMode) {
      ThemeMode.light => Brightness.light,
      ThemeMode.dark => Brightness.dark,
      _ => WidgetsBinding.instance.platformDispatcher.platformBrightness,
    };
    windowManager.setBrightness(bright).catchError((_) {});
  }

  state.addListener(applyTitleBar);
  applyTitleBar();
  await state.init();
}

class _WindowController with WindowListener {
  _WindowController(this.state);

  final AppState state;

  @override
  void onWindowClose() async {
    final closeToTray = state.settings['close_to_tray'] != false;
    if (closeToTray) {
      await windowManager.hide();
      return;
    }
    await state.disposeSidecar();
    await windowManager.destroy();
  }
}

class _TrayController with TrayListener {
  _TrayController(this.state);

  final AppState state;

  Future<void> install() async {
    if (!(Platform.isWindows || Platform.isLinux || Platform.isMacOS)) return;
    try {
      final iconPath = await _trayIconPath();
      await windowManager.setIcon(iconPath).catchError((_) {});
      await trayManager.setIcon(iconPath);
      await trayManager.setToolTip('NodePulse');
      await trayManager.setContextMenu(
        Menu(
          items: [
            MenuItem(key: 'show', label: '显示 NodePulse'),
            MenuItem.separator(),
            MenuItem(key: 'quit', label: '退出'),
          ],
        ),
      );
      trayManager.addListener(this);
    } catch (_) {
      // Tray support is optional; the app still works without it.
    }
  }

  Future<String> _trayIconPath() async {
    final dir = await getTemporaryDirectory();
    final path = p.join(dir.path, 'nodepulse-tray.png');
    final file = File(path);
    if (!await file.exists()) {
      final data = await rootBundle.load('assets/icon.png');
      await file.writeAsBytes(data.buffer.asUint8List());
    }
    return path;
  }

  @override
  void onTrayIconMouseDown() {
    windowManager.show();
    windowManager.focus();
  }

  @override
  void onTrayMenuItemClick(MenuItem menuItem) async {
    if (menuItem.key == 'show') {
      await windowManager.show();
      await windowManager.focus();
    } else if (menuItem.key == 'quit') {
      await state.disposeSidecar();
      await trayManager.destroy();
      await windowManager.destroy();
    }
  }
}
