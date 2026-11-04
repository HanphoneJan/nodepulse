// ignore_for_file: deprecated_member_use

import 'dart:io';

import 'package:flutter/services.dart';
import 'package:material_ui/material_ui.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:tray_manager/legacy.dart';
import 'package:tray_manager/tray_manager.dart' as tray_native;
import 'package:window_manager/legacy.dart';

import 'app.dart';
import 'state/app_state.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final isDesktop = Platform.isLinux || Platform.isMacOS || Platform.isWindows;
  if (isDesktop) {
    await windowManager.ensureInitialized();
    await windowManager.setPreventClose(true);
    if (!await _acquireSingleInstance()) {
      exit(0);
    }
  }

  final windowOptions = WindowOptions(
    size: const Size(1180, 780),
    minimumSize: const Size(720, 560),
    center: true,
    title: 'NodePulse',
    backgroundColor: Colors.transparent,
    // Windows draws its own title bar in the app shell, with the window
    // controls at the right end of the top bar.
    titleBarStyle: Platform.isWindows ? TitleBarStyle.hidden : null,
  );

  final state = AppState();
  runApp(NodePulseApp(state: state));
  if (isDesktop) {
    await windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
    });

    final trayReady = await _TrayController(state).install();
    final controller = _WindowController(state, trayReady);
    windowManager.addListener(controller);
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
  }
  await state.init();
}

Future<bool> _acquireSingleInstance() async {
  const port = 47654;
  try {
    final server = await ServerSocket.bind(InternetAddress.loopbackIPv4, port);
    server.listen((socket) {
      socket.destroy();
      windowManager.show();
      windowManager.focus();
    });
    return true;
  } on SocketException {
    try {
      final socket = await Socket.connect(
        InternetAddress.loopbackIPv4,
        port,
        timeout: const Duration(milliseconds: 500),
      );
      socket.destroy();
    } catch (_) {}
    return false;
  }
}

class _WindowController with WindowListener {
  _WindowController(this.state, this.trayReady);

  final AppState state;
  final bool trayReady;

  @override
  void onWindowClose() async {
    final closeToTray = state.settings['close_to_tray'] != false;
    if (closeToTray && trayReady) {
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

  Future<bool> install() async {
    if (!(Platform.isWindows || Platform.isLinux || Platform.isMacOS)) {
      return false;
    }
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
      // The legacy bridge only enables this on Linux. Windows and macOS keep
      // the native default (none), so a right click showed no menu there.
      trayManager.trayIcon?.setContextMenuTrigger(
        tray_native.ContextMenuTrigger.rightClicked,
      );
      trayManager.addListener(this);
      return true;
    } catch (error) {
      // Without a tray icon there is no way back once the window is hidden,
      // so the window controller quits instead of hiding in this case.
      debugPrint('NodePulse: tray icon unavailable: $error');
      try {
        await trayManager.destroy();
      } catch (_) {}
      return false;
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
      await _quit();
    }
  }

  Future<void> _quit() async {
    try {
      await state.disposeSidecar();
    } finally {
      try {
        await trayManager.destroy();
        await windowManager.destroy();
      } catch (error) {
        debugPrint('NodePulse: quit failed: $error');
        exit(0);
      }
    }
  }
}
