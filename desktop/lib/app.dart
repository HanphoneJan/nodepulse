// ignore_for_file: deprecated_member_use

import 'package:material_ui/material_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'state/app_state.dart';
import 'ui/app_theme.dart';
import 'update/update_checker.dart';
import 'ui/flclash/color.dart';
import 'ui/flclash/metrics.dart';
import 'ui/flclash/shape.dart';
import 'ui/flclash/surface.dart';
import 'ui/flclash/typography.dart';
import 'ui/widgets.dart';
import 'ui/window_header.dart';
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
          home: _DesktopFrame(
            child: state.ready
                ? (state.error == null
                      ? _Shell(state: state)
                      : _StartupError(state: state))
                : _Splash(state: state),
          ),
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

/// Windows has no native caption, so the FlClash-style 32px header sits above
/// the whole app, including the splash and startup error screens.
class _DesktopFrame extends StatelessWidget {
  const _DesktopFrame({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (!usesCustomWindowHeader) return child;
    return Column(
      children: [
        const AppWindowHeader(),
        Expanded(child: child),
      ],
    );
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
            const SizedBox(height: AppSpace.lg),
            Text(
              '正在启动 NodePulse',
              style: AppType.secondary(
                context,
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
    final scheme = theme.colorScheme;
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 440),
          child: AppCard(
            radius: AppCorner.alert,
            padding: const EdgeInsets.all(AppSpace.xxl),
            shadow: true,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Glyphs.error, size: 36, color: scheme.error),
                const SizedBox(height: AppSpace.md),
                Text('无法启动后端', style: AppType.section(context)),
                const SizedBox(height: AppSpace.sm),
                Text(
                  state.error ?? '未知错误',
                  textAlign: TextAlign.center,
                  style: AppType.secondary(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
                const SizedBox(height: AppSpace.lg),
                AppButton(
                  label: '重试',
                  icon: Glyphs.refresh,
                  onPressed: () {
                    state.error = null;
                    state.ready = false;
                    state.init();
                  },
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _Shell extends StatefulWidget {
  const _Shell({required this.state});

  final AppState state;

  @override
  State<_Shell> createState() => _ShellState();
}

class _ShellState extends State<_Shell> {
  static const _items = [
    (
      view: 'dashboard',
      label: '仪表盘',
      icon: Glyphs.dashboard,
      selected: Glyphs.dashboardFill,
    ),
    (view: 'nodes', label: '节点', icon: Glyphs.nodes, selected: Glyphs.nodes),
    (
      view: 'settings',
      label: '设置',
      icon: Glyphs.settings,
      selected: Glyphs.settingsFill,
    ),
    (
      view: 'history',
      label: '历史',
      icon: Glyphs.history,
      selected: Glyphs.history,
    ),
  ];

  AppState get state => widget.state;
  bool _expanded = true;

  @override
  void initState() {
    super.initState();
    state.addListener(_handleState);
    WidgetsBinding.instance.addPostFrameCallback((_) => _maybeShowUpdate());
    SharedPreferences.getInstance().then((prefs) {
      final saved = prefs.getBool('nav_expanded');
      if (mounted && saved != null) setState(() => _expanded = saved);
    });
  }

  @override
  void dispose() {
    state.removeListener(_handleState);
    super.dispose();
  }

  void _handleState() {
    if (!mounted || !state.updateDialogPending) return;
    WidgetsBinding.instance.addPostFrameCallback((_) => _maybeShowUpdate());
  }

  void _maybeShowUpdate() {
    if (!mounted) return;
    final release = state.update;
    if (!state.updateDialogPending || release == null) return;
    state.consumeUpdateDialog();
    _showUpdateDialog(release);
  }

  Future<void> _showUpdateDialog(ReleaseInfo release) async {
    final go = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('发现新版本 v${release.version}'),
        content: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 460, maxHeight: 360),
          child: SingleChildScrollView(
            child: Text(
              release.notes.isEmpty ? '有新版本可用。' : release.notes,
              style: AppType.secondary(context),
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('稍后'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('前往下载'),
          ),
        ],
      ),
    );
    await state.dismissUpdate(release.version);
    if (go != true) return;
    final opened = await openExternalUrl(release.url);
    if (!opened && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('无法打开浏览器，请手动访问 Release 页面。')),
      );
    }
  }

  void _toggleExpanded() {
    setState(() => _expanded = !_expanded);
    SharedPreferences.getInstance().then(
      (prefs) => prefs.setBool('nav_expanded', _expanded),
    );
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final narrow = constraints.maxWidth < 820;
        if (narrow) {
          return Scaffold(
            body: SafeArea(
              bottom: false,
              child: Column(
                children: [
                  Expanded(
                    child: Column(
                      children: [
                        _TopBar(state: state, narrow: true),
                        Expanded(child: _Body(state: state, bottomInset: 88)),
                      ],
                    ),
                  ),
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
                _Sidebar(
                  state: state,
                  items: _items,
                  expanded: _expanded,
                  onToggle: _toggleExpanded,
                ),
                Expanded(
                  child: Column(
                    children: [
                      _TopBar(state: state, narrow: false),
                      Expanded(child: _Body(state: state, bottomInset: 0)),
                    ],
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _Sidebar extends StatelessWidget {
  const _Sidebar({
    required this.state,
    required this.items,
    required this.expanded,
    required this.onToggle,
  });

  final AppState state;
  final List<({String view, String label, IconData icon, IconData selected})>
  items;
  final bool expanded;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final width = expanded ? 220.0 : 48.0;
    final selectedSurface = SurfaceLevel.block.colorOf(scheme);
    return AnimatedContainer(
      duration: const Duration(milliseconds: 250),
      curve: Curves.easeOutCubic,
      width: width,
      color: scheme.page,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(
            height: 56,
            child: Align(
              alignment: expanded
                  ? AlignmentDirectional.centerEnd
                  : AlignmentDirectional.center,
              child: Padding(
                padding: EdgeInsets.symmetric(horizontal: expanded ? 8 : 0),
                child: IconButton(
                  tooltip: expanded ? '收起侧栏' : '展开侧栏',
                  onPressed: onToggle,
                  icon: Icon(
                    expanded ? Glyphs.chevronLeft : Glyphs.chevronRight,
                    size: 18,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: AppSpace.xs),
          for (final item in items)
            Padding(
              padding: EdgeInsets.symmetric(
                horizontal: expanded ? 8 : 4,
                vertical: 2,
              ),
              child: Tooltip(
                message: expanded ? '' : item.label,
                child: InkWell(
                  borderRadius: BorderRadius.circular(10),
                  onTap: () => state.setView(item.view),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 250),
                    curve: Curves.easeOutCubic,
                    height: 40,
                    decoration: ShapeDecoration(
                      color: state.view == item.view
                          ? selectedSurface
                          : Colors.transparent,
                      shape: AppShape.all(10),
                    ),
                    child: Row(
                      children: [
                        if (state.view == item.view)
                          Container(
                            width: 3,
                            height: 16,
                            decoration: ShapeDecoration(
                              color: scheme.primary,
                              shape: AppShape.full,
                            ),
                          )
                        else
                          const SizedBox(width: 3),
                        SizedBox(width: expanded ? 9 : 0),
                        Icon(
                          state.view == item.view ? item.selected : item.icon,
                          size: 24,
                          color: state.view == item.view
                              ? scheme.primary
                              : scheme.onSurfaceVariant,
                        ),
                        if (expanded) ...[
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              item.label,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: AppTextStyle.subheadline
                                  .of(
                                    context,
                                    emphasized: state.view == item.view,
                                  )
                                  .copyWith(
                                    color: state.view == item.view
                                        ? scheme.primary
                                        : scheme.onSurfaceVariant,
                                  ),
                            ),
                          ),
                          const SizedBox(width: 8),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          const Spacer(),
          if (expanded)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
              child: Text(
                'NodePulse',
                style: AppType.caption(
                  context,
                ).copyWith(color: scheme.onSurfaceVariant),
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
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 0, 8, 8),
      child: Container(
        height: 64,
        padding: const EdgeInsets.all(4),
        decoration: ShapeDecoration(
          color: scheme.card,
          shape: AppShape.all(AppCorner.xxl),
          shadows: AppShadows.e1(scheme),
        ),
        child: Row(
          children: [
            for (final item in items)
              Expanded(
                child: InkWell(
                  borderRadius: BorderRadius.circular(18),
                  onTap: () => state.setView(item.view),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    decoration: ShapeDecoration(
                      color: state.view == item.view
                          ? scheme.primaryContainer
                          : Colors.transparent,
                      shape: AppShape.all(18),
                    ),
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(
                          state.view == item.view ? item.selected : item.icon,
                          size: 22,
                          color: state.view == item.view
                              ? scheme.onPrimaryContainer
                              : scheme.onSurfaceVariant,
                        ),
                        const SizedBox(height: 2),
                        Text(
                          item.label,
                          style: AppTextStyle.caption2
                              .of(context)
                              .copyWith(
                                fontWeight: FontWeight.w500,
                                color: state.view == item.view
                                    ? scheme.onPrimaryContainer
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
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar({required this.state, required this.narrow});

  final AppState state;
  final bool narrow;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final title = switch (state.view) {
      'dashboard' => '仪表盘',
      'nodes' => '节点',
      'settings' => '设置',
      'history' => '历史',
      _ => '仪表盘',
    };
    final status = StatusChip(
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
      dot: true,
    );
    final bar = Row(
      children: [
        Text(title, style: AppType.pageTitle(context)),
        const Spacer(),
        if (!narrow) ...[
          _EnvironmentButton(state: state),
          const SizedBox(width: 8),
        ],
        status,
      ],
    );
    return Container(
      height: pageToolbarHeight,
      color: scheme.page,
      padding: EdgeInsets.symmetric(horizontal: narrow ? 16 : 24),
      child: bar,
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
                        style: AppType.secondary(context),
                      ),
                      Text(
                        (profile['api'] ?? '').toString(),
                        style: AppType.caption(
                          context,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ],
                  ),
                ),
                if (profile['id'] == state.settings['active_profile'])
                  Icon(Glyphs.check, size: 18, color: scheme.success),
              ],
            ),
          ),
      ],
      child: Container(
        height: 40,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        decoration: ShapeDecoration(color: scheme.inset, shape: AppShape.full),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          spacing: 6,
          children: [
            Icon(Glyphs.swap, size: 18, color: scheme.onSurfaceVariant),
            ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 140),
              child: Text(
                name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppType.secondary(
                  context,
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

  double get _maxWidth => switch (state.view) {
    'dashboard' => 1440,
    'nodes' => 1280,
    'settings' || 'history' => 960,
    _ => 1080,
  };

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
        key: PageStorageKey<String>(state.view),
        padding: EdgeInsets.fromLTRB(16, 0, 24, 24 + bottomInset),
        child: Center(
          child: ConstrainedBox(
            constraints: BoxConstraints(maxWidth: _maxWidth),
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 200),
              switchInCurve: Curves.easeOutCubic,
              switchOutCurve: Curves.easeInCubic,
              child: KeyedSubtree(key: ValueKey(state.view), child: child),
            ),
          ),
        ),
      ),
    );
  }
}
