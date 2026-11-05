// ignore_for_file: deprecated_member_use

// Ported from FlClash (https://github.com/chen08209/FlClash).
// Copyright (c) FlClash contributors, licensed under GPL-3.0.
// Modified for NodePulse.

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:material_ui/material_ui.dart';
import 'package:window_manager/legacy.dart';

import 'app_theme.dart';
import 'flclash/shape.dart';
import 'widgets.dart';

/// FlClash draws a 32px window header on Windows and Linux. NodePulse keeps
/// the native title bar on Linux and macOS, so only Windows uses this one.
const windowHeaderHeight = 32.0;

bool get usesCustomWindowHeader =>
    !kIsWeb && defaultTargetPlatform == TargetPlatform.windows;

/// The drag area plus the pin, minimize, maximize and close buttons.
class AppWindowHeader extends StatefulWidget {
  const AppWindowHeader({super.key});

  @override
  State<AppWindowHeader> createState() => _AppWindowHeaderState();
}

class _AppWindowHeaderState extends State<AppWindowHeader> with WindowListener {
  bool _maximized = false;
  bool _pinned = false;

  @override
  void initState() {
    super.initState();
    windowManager.addListener(this);
    unawaited(_syncFromWindow());
  }

  Future<void> _syncFromWindow() async {
    try {
      final states = await Future.wait<bool>([
        windowManager.isMaximized(),
        windowManager.isAlwaysOnTop(),
      ]);
      if (!mounted) return;
      setState(() {
        _maximized = states[0];
        _pinned = states[1];
      });
    } catch (_) {
      // The header is also built in widget tests, where the native window
      // channel is missing. The buttons stay usable and simply start unpinned.
    }
  }

  @override
  void dispose() {
    windowManager.removeListener(this);
    super.dispose();
  }

  @override
  void onWindowMaximize() {
    if (mounted) setState(() => _maximized = true);
  }

  @override
  void onWindowUnmaximize() {
    if (mounted) setState(() => _maximized = false);
  }

  Future<void> _toggleMaximize() async {
    try {
      if (_maximized) {
        await windowManager.unmaximize();
      } else {
        await windowManager.maximize();
      }
    } catch (_) {}
  }

  Future<void> _togglePin() async {
    final next = !_pinned;
    try {
      await windowManager.setAlwaysOnTop(next);
      if (mounted) setState(() => _pinned = next);
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final brightness = theme.brightness;
    return Container(
      height: windowHeaderHeight,
      decoration: BoxDecoration(
        color: scheme.chrome,
        border: Border(
          bottom: BorderSide(
            color: scheme.outlineVariant.withValues(alpha: 0.5),
          ),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(child: DragToMoveArea(child: SizedBox.expand())),
          _PinButton(pinned: _pinned, onPressed: _togglePin),
          Tooltip(
            message: '最小化',
            child: WindowCaptionButton.minimize(
              brightness: brightness,
              onPressed: () => unawaited(windowManager.minimize()),
            ),
          ),
          if (_maximized)
            Tooltip(
              message: '还原',
              child: WindowCaptionButton.unmaximize(
                brightness: brightness,
                onPressed: () => unawaited(windowManager.unmaximize()),
              ),
            )
          else
            Tooltip(
              message: '最大化',
              child: WindowCaptionButton.maximize(
                brightness: brightness,
                onPressed: _toggleMaximize,
              ),
            ),
          Tooltip(
            message: '关闭',
            child: WindowCaptionButton.close(
              brightness: brightness,
              onPressed: () => unawaited(windowManager.close()),
            ),
          ),
        ],
      ),
    );
  }
}

class _PinButton extends StatelessWidget {
  const _PinButton({required this.pinned, required this.onPressed});

  final bool pinned;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SizedBox(
      width: 32,
      height: windowHeaderHeight,
      child: IconButton(
        tooltip: pinned ? '取消置顶' : '置顶窗口',
        onPressed: onPressed,
        style: ButtonStyle(
          minimumSize: const WidgetStatePropertyAll(
            Size(32, windowHeaderHeight),
          ),
          maximumSize: const WidgetStatePropertyAll(
            Size(32, windowHeaderHeight),
          ),
          padding: const WidgetStatePropertyAll(EdgeInsets.zero),
          iconSize: const WidgetStatePropertyAll(16),
          shape: const WidgetStatePropertyAll(AppShape.full),
          foregroundColor: WidgetStatePropertyAll(
            pinned ? scheme.primary : scheme.onSurfaceVariant,
          ),
          overlayColor: WidgetStatePropertyAll(
            scheme.onSurfaceVariant.withValues(alpha: 0.08),
          ),
        ),
        icon: Icon(pinned ? Glyphs.pinFill : Glyphs.pin),
      ),
    );
  }
}
