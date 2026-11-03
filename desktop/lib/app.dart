import 'package:material_ui/material_ui.dart';

import 'state/app_state.dart';
import 'ui/app_theme.dart';
import 'ui/flclash/color.dart';
import 'ui/flclash/typography.dart';
import 'ui/widgets.dart';
import 'views/dashboard.dart';
import 'views/history.dart';
import 'views/nodes.dart';
import 'views/settings.dart';

class NodePulseApp extends StatelessWidget {
  const NodePulseApp({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: state,
      builder: (context, _) {
        final seed = _parseSeed(state.seed);
        return MaterialApp(
          title: 'NodePulse',
          debugShowCheckedModeBanner: false,
          scaffoldMessengerKey: state.messengerKey,
          theme: buildAppTheme(Brightness.light, seed),
          darkTheme: buildAppTheme(Brightness.dark, seed),
          themeMode: state.themeMode,
          home: state.ready
              ? (state.error == null
                    ? _Shell(state: state)
                    : _StartupError(state: state))
              : _Splash(state: state),
        );
      },
    );
  }

  static Color _parseSeed(String value) {
    final hex = value.replaceFirst('#', '');
    final parsed = int.tryParse(hex, radix: 16);
    return Color(0xff000000 | (parsed ?? 0x007aff));
  }
}

class _Splash extends StatelessWidget {
  const _Splash({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(
              width: 28,
              height: 28,
              child: CircularProgressIndicator(strokeWidth: 2.5),
            ),
            const SizedBox(height: 16),
            Text(
              '正在启动 NodePulse',
              style: appText(
                theme,
                AppTextStyle.subheadline,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
        ),
      ),
    );
  }
}

class _StartupError extends StatelessWidget {
  const _StartupError({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.error_outline,
                  size: 36,
                  color: theme.colorScheme.error,
                ),
                const SizedBox(height: 12),
                Text('无法启动后端', style: appText(theme, AppTextStyle.headline)),
                const SizedBox(height: 8),
                Text(
                  state.error ?? '未知错误',
                  textAlign: TextAlign.center,
                  style: appText(
                    theme,
                    AppTextStyle.subheadline,
                  ).copyWith(color: theme.colorScheme.onSurfaceVariant),
                ),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () {
                    state.error = null;
                    state.ready = false;
                    state.init();
                  },
                  child: const Text('重试'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _Shell extends StatelessWidget {
  const _Shell({required this.state});

  final AppState state;

  static const _items = [
    (
      view: 'dashboard',
      label: '仪表盘',
      icon: Icons.dashboard_outlined,
      selected: Icons.dashboard,
    ),
    (
      view: 'nodes',
      label: '节点',
      icon: Icons.format_list_bulleted,
      selected: Icons.format_list_bulleted,
    ),
    (
      view: 'settings',
      label: '设置',
      icon: Icons.settings_outlined,
      selected: Icons.settings,
    ),
    (
      view: 'history',
      label: '历史',
      icon: Icons.history,
      selected: Icons.history,
    ),
  ];

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final narrow = constraints.maxWidth < 760;
        final content = Column(
          children: [
            _Toolbar(state: state, narrow: narrow),
            Expanded(
              child: narrow
                  ? _Body(state: state, bottomInset: 64)
                  : _Body(state: state, bottomInset: 0),
            ),
          ],
        );
        if (narrow) {
          return Scaffold(
            body: SafeArea(
              bottom: false,
              child: Column(
                children: [
                  Expanded(child: content),
                  _BottomDock(state: state, items: _items),
                ],
              ),
            ),
          );
        }
        return Scaffold(
          body: SafeArea(
            child: Row(
              children: [
                _Rail(state: state, items: _items),
                Expanded(child: content),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _Rail extends StatelessWidget {
  const _Rail({required this.state, required this.items});

  final AppState state;
  final List<({String view, String label, IconData icon, IconData selected})>
  items;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Container(
      width: 72,
      color: scheme.page,
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 12),
      child: Column(
        children: [
          for (final item in items)
            Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: Tooltip(
                message: item.label,
                child: InkWell(
                  borderRadius: BorderRadius.circular(16),
                  onTap: () => state.setView(item.view),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    width: 56,
                    height: 56,
                    decoration: BoxDecoration(
                      color: state.view == item.view ? scheme.inset : null,
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Icon(
                      state.view == item.view ? item.selected : item.icon,
                      size: 24,
                      color: state.view == item.view
                          ? scheme.primary
                          : scheme.onSurfaceVariant,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _BottomDock extends StatelessWidget {
  const _BottomDock({required this.state, required this.items});

  final AppState state;
  final List<({String view, String label, IconData icon, IconData selected})>
  items;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      height: 64,
      color: scheme.page,
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        children: [
          for (final item in items)
            Expanded(
              child: InkWell(
                borderRadius: BorderRadius.circular(12),
                onTap: () => state.setView(item.view),
                child: Container(
                  decoration: BoxDecoration(
                    color: state.view == item.view ? scheme.inset : null,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        state.view == item.view ? item.selected : item.icon,
                        size: 22,
                        color: state.view == item.view
                            ? scheme.primary
                            : scheme.onSurfaceVariant,
                      ),
                      const SizedBox(height: 2),
                      Text(
                        item.label,
                        style: appText(Theme.of(context), AppTextStyle.caption2)
                            .copyWith(
                              color: state.view == item.view
                                  ? scheme.primary
                                  : scheme.onSurfaceVariant,
                            ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _Toolbar extends StatelessWidget {
  const _Toolbar({required this.state, required this.narrow});

  final AppState state;
  final bool narrow;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final title = switch (state.view) {
      'dashboard' => '仪表盘',
      'nodes' => '节点',
      'settings' => '设置',
      'history' => '历史',
      _ => '仪表盘',
    };
    return Container(
      height: 72,
      color: scheme.page,
      padding: EdgeInsets.symmetric(horizontal: narrow ? 16 : 20),
      child: Row(
        children: [
          Text(
            title,
            style: appText(
              theme,
              AppTextStyle.pageTitle,
            ).copyWith(fontWeight: FontWeight.w600),
          ),
          const Spacer(),
          if (!narrow) ...[
            _EnvironmentButton(state: state),
            const SizedBox(width: 4),
          ],
          StatusChip(
            label: state.connected
                ? '已连接'
                : state.clients.any((c) => c is Map && c['running'] == true)
                ? '检测到客户端'
                : '未连接',
            tone: state.connected
                ? StatusTone.ok
                : state.clients.any((c) => c is Map && c['running'] == true)
                ? StatusTone.warn
                : StatusTone.neutral,
          ),
          if (!narrow && state.version.isNotEmpty) ...[
            const SizedBox(width: 8),
            Text(
              'v${state.version}',
              style: appText(
                theme,
                AppTextStyle.caption2,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
        ],
      ),
    );
  }
}

class _EnvironmentButton extends StatelessWidget {
  const _EnvironmentButton({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final active = state.activeProfile;
    final name = (active['name'] ?? '默认环境').toString();
    return PopupMenuButton<String>(
      tooltip: '快速切换环境',
      color: scheme.floating,
      position: PopupMenuPosition.under,
      onSelected: state.activateProfile,
      itemBuilder: (context) => [
        for (final profile in state.profiles.whereType<Map>())
          PopupMenuItem<String>(
            value: profile['id'].toString(),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        (profile['name'] ?? '').toString(),
                        style: appText(theme, AppTextStyle.subheadline),
                      ),
                      Text(
                        (profile['api'] ?? '').toString(),
                        style: appText(
                          theme,
                          AppTextStyle.footnote,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ],
                  ),
                ),
                if (profile['id'] == state.settings['active_profile'])
                  Icon(Icons.check, size: 18, color: scheme.success),
              ],
            ),
          ),
      ],
      child: Container(
        height: 40,
        padding: const EdgeInsets.symmetric(horizontal: 8),
        decoration: BoxDecoration(borderRadius: BorderRadius.circular(999)),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          spacing: 6,
          children: [
            Icon(Icons.swap_horiz, size: 18, color: scheme.onSurfaceVariant),
            ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 140),
              child: Text(
                name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: appText(
                  theme,
                  AppTextStyle.subheadline,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({required this.state, required this.bottomInset});

  final AppState state;
  final double bottomInset;

  @override
  Widget build(BuildContext context) {
    final child = switch (state.view) {
      'nodes' => NodesView(state: state),
      'settings' => SettingsView(state: state),
      'history' => HistoryView(state: state),
      _ => DashboardView(state: state),
    };
    return Scrollbar(
      child: SingleChildScrollView(
        padding: EdgeInsets.fromLTRB(16, 0, 16, 24 + bottomInset),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 1080),
            child: child,
          ),
        ),
      ),
    );
  }
}
