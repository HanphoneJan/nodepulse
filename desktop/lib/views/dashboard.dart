import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/flclash/shape.dart';
import '../ui/flclash/typography.dart';
import '../ui/widgets.dart';
import 'result_table.dart';

class DashboardView extends StatelessWidget {
  const DashboardView({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final gap = width < 760 ? 8.0 : (width < 1040 ? 12.0 : 16.0);
        if (width < 760) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: gap,
            children: [
              _CurrentNodeCard(state: state),
              _ScoreCard(state: state),
              _ConnectionCard(state: state),
              _ControlBar(state: state),
              _EnvironmentCard(state: state),
              _LatestDecisionCard(state: state),
              _ResultsSummaryCard(state: state),
              _SubscribeCard(state: state),
              _DownloadCard(state: state),
              _ResultsCard(state: state),
            ],
          );
        }
        if (width < 900) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: gap,
            children: [
              _CurrentNodeCard(state: state),
              IntrinsicHeight(
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  spacing: gap,
                  children: [
                    Expanded(child: _ScoreCard(state: state)),
                    Expanded(child: _ConnectionCard(state: state)),
                  ],
                ),
              ),
              _ControlBar(state: state),
              IntrinsicHeight(
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  spacing: gap,
                  children: [
                    Expanded(child: _EnvironmentCard(state: state)),
                    Expanded(child: _LatestDecisionCard(state: state)),
                    Expanded(child: _ResultsSummaryCard(state: state)),
                  ],
                ),
              ),
              _SubscribeCard(state: state),
              _DownloadCard(state: state),
              _ResultsCard(state: state),
            ],
          );
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          spacing: gap,
          children: [
            SizedBox(
              height: 216,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: gap,
                children: [
                  Expanded(flex: 2, child: _CurrentNodeCard(state: state)),
                  Expanded(child: _ScoreCard(state: state)),
                  Expanded(child: _ConnectionCard(state: state)),
                ],
              ),
            ),
            _ControlBar(state: state),
            IntrinsicHeight(
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: gap,
                children: [
                  Expanded(child: _EnvironmentCard(state: state)),
                  Expanded(child: _LatestDecisionCard(state: state)),
                  Expanded(flex: 2, child: _ResultsSummaryCard(state: state)),
                ],
              ),
            ),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              spacing: gap,
              children: [
                Expanded(child: _SubscribeCard(state: state)),
                Expanded(child: _DownloadCard(state: state)),
              ],
            ),
            _ResultsCard(state: state),
          ],
        );
      },
    );
  }
}

class _CurrentNodeCard extends StatelessWidget {
  const _CurrentNodeCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final error = state.status['error'] != null;
    final busy = state.busy;
    return AppCard(
      radius: AppCorner.dashboard,
      shadow: true,
      padding: const EdgeInsets.all(AppSpace.xl),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            spacing: AppSpace.sm,
            children: [
              Text(
                '当前节点',
                style: AppType.caption(
                  context,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
              StatusChip(
                label: error
                    ? '连接失败'
                    : state.connected
                    ? '已连接'
                    : '未连接',
                tone: error
                    ? StatusTone.error
                    : state.connected
                    ? StatusTone.ok
                    : StatusTone.warn,
                dot: true,
              ),
            ],
          ),
          const SizedBox(height: AppSpace.xs),
          Text(
            state.connected ? cleanName(state.currentNode) : '—',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppType.nodeTitle(context),
          ),
          if (state.groupName.isNotEmpty)
            Text(
              '${state.groupName} · ${state.memberCount} 个成员',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          const SizedBox(height: AppSpace.lg),
          Wrap(
            spacing: AppSpace.sm,
            runSpacing: AppSpace.sm,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              AppButton(
                label: busy && state.busyKind == 'oneclick' ? '优化中…' : '一键优化',
                icon: Glyphs.play,
                loading: busy && state.busyKind == 'oneclick',
                onPressed: state.oneClick,
              ),
              if (busy)
                AppButton(
                  label: '取消',
                  variant: AppButtonVariant.ghost,
                  onPressed: state.cancel,
                ),
            ],
          ),
          if (busy) ...[
            const SizedBox(height: AppSpace.md),
            Row(
              spacing: AppSpace.sm,
              children: [
                SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    value: _progressRatio(state.progress),
                  ),
                ),
                Text(
                  '${state.busyLabel} ${state.progress['done'] ?? 0}/${state.progress['total'] ?? 0}',
                  style: AppType.caption(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              ],
            ),
          ] else ...[
            const SizedBox(height: AppSpace.sm),
            Row(
              spacing: 6,
              children: [
                Icon(Glyphs.timer, size: 15, color: scheme.onSurfaceVariant),
                Text(
                  '最近测速',
                  style: AppType.caption(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
                Text(
                  state.lastTestAt == null ? '尚未测速' : timeAgo(state.lastTestAt),
                  style: AppType.caption(
                    context,
                  ).copyWith(fontWeight: FontWeight.w600),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  double? _progressRatio(Map<String, dynamic> progress) {
    final total = (progress['total'] as num?)?.toInt() ?? 0;
    final done = (progress['done'] as num?)?.toInt() ?? 0;
    if (total <= 0) return null;
    return (done / total).clamp(0.0, 1.0);
  }
}

class _ScoreCard extends StatelessWidget {
  const _ScoreCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final best = state.best;
    return AppCard(
      radius: AppCorner.dashboard,
      shadow: true,
      padding: const EdgeInsets.all(AppSpace.xl),
      child: best == null
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Row(
                  spacing: 6,
                  children: [
                    Container(
                      width: 6,
                      height: 6,
                      decoration: BoxDecoration(
                        color: scheme.primary,
                        shape: BoxShape.circle,
                      ),
                    ),
                    Text(
                      '待测速',
                      style: AppType.caption(
                        context,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ],
                ),
                const SizedBox(height: AppSpace.xs),
                Text(
                  '${state.memberCount}',
                  style: AppType.figure(context, style: AppTextStyle.title1),
                ),
                Text(
                  '个节点',
                  style: AppType.caption(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
                const SizedBox(height: AppSpace.sm),
                Text(
                  '${state.settings['rounds'] ?? 5} 轮 · 并发 ${state.settings['concurrency'] ?? 8}',
                  style: AppType.caption(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              ],
            )
          : Center(
              child: ScoreRing(
                value: (best['score'] as num?)?.toDouble() ?? 0,
                bestName: cleanName((best['name'] ?? '').toString()),
              ),
            ),
    );
  }
}

class _ConnectionCard extends StatelessWidget {
  const _ConnectionCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final running = state.clients
        .whereType<Map>()
        .where((client) => client['running'] == true)
        .toList();
    return AppCard(
      radius: AppCorner.dashboard,
      shadow: true,
      padding: const EdgeInsets.all(AppSpace.xl),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '连接',
            glyph: Glyphs.link,
            padding: EdgeInsets.zero,
            actions: [
              AppButton(
                label: state.detecting ? '检测中…' : '重新检测',
                variant: AppButtonVariant.ghost,
                enabled: !state.detecting,
                onPressed: state.detect,
              ),
            ],
          ),
          const SizedBox(height: AppSpace.sm),
          if (running.isEmpty)
            Text(
              '未检测到客户端。',
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            )
          else
            for (final client in running)
              Padding(
                padding: const EdgeInsets.only(bottom: AppSpace.xs),
                child: Row(
                  children: [
                    Container(
                      width: 7,
                      height: 7,
                      decoration: BoxDecoration(
                        color: scheme.success,
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: AppSpace.sm),
                    Expanded(
                      child: Text(
                        (client['name'] ?? '').toString(),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppType.secondary(context),
                      ),
                    ),
                    Text(
                      '运行中',
                      style: AppType.caption(
                        context,
                      ).copyWith(color: scheme.success),
                    ),
                  ],
                ),
              ),
          const SizedBox(height: AppSpace.sm),
          Text(
            (state.settings['api'] ?? '未配置控制口').toString(),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppType.micro(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }
}

class _ControlBar extends StatelessWidget {
  const _ControlBar({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final busy = state.busy;
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Wrap(
        spacing: AppSpace.md,
        runSpacing: AppSpace.md,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          Text(
            '强度',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          SizedBox(
            width: 260,
            child: SegmentedChoice<String>(
              value: '${state.settings['rounds'] ?? 5}',
              options: const [
                (value: '3', label: '快速 3 轮', icon: null),
                (value: '5', label: '标准 5 轮', icon: null),
                (value: '10', label: '精确 10 轮', icon: null),
              ],
              onChanged: (value) =>
                  state.updateSetting('rounds', int.tryParse(value) ?? 5),
            ),
          ),
          AppButton(
            label: busy && state.busyKind == 'test' ? '测速中…' : '只测速',
            icon: Glyphs.refresh,
            variant: AppButtonVariant.tonal,
            loading: busy && state.busyKind == 'test',
            onPressed: state.testOnly,
          ),
          AppButton(
            label: busy && state.busyKind == 'all' ? '联合测速中…' : '全部订阅',
            icon: Glyphs.nodes,
            variant: AppButtonVariant.outline,
            loading: busy && state.busyKind == 'all',
            onPressed: state.testAll,
          ),
          Row(
            mainAxisSize: MainAxisSize.min,
            spacing: AppSpace.sm,
            children: [
              Text(
                '自动优化',
                style: AppType.caption(
                  context,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
              AppSwitch(value: state.daemon, onChanged: state.toggleDaemon),
            ],
          ),
          SizedBox(
            width: 116,
            child: AppSelect<String>(
              value: '${state.settings['daemon_interval'] ?? 300}',
              height: 40,
              items: const [
                (value: '300', label: '5 分钟'),
                (value: '600', label: '10 分钟'),
                (value: '1800', label: '30 分钟'),
              ],
              onChanged: (value) => state.updateSetting(
                'daemon_interval',
                int.tryParse(value) ?? 300,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EnvironmentCard extends StatelessWidget {
  const _EnvironmentCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final active = state.activeProfile;
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '当前环境',
            glyph: Glyphs.swap,
            padding: EdgeInsets.zero,
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            (active['name'] ?? '默认环境').toString(),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppType.secondary(
              context,
            ).copyWith(fontWeight: FontWeight.w600),
          ),
          Text(
            (active['api'] ?? '—').toString(),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppType.micro(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.sm),
          Align(
            alignment: AlignmentDirectional.centerStart,
            child: AppButton(
              label: '管理环境',
              variant: AppButtonVariant.ghost,
              onPressed: () => state.setView('settings'),
            ),
          ),
        ],
      ),
    );
  }
}

class _LatestDecisionCard extends StatelessWidget {
  const _LatestDecisionCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    Map? latest;
    for (final item in state.history.whereType<Map>()) {
      latest = item;
      break;
    }
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '最近自动选择',
            glyph: Glyphs.timer,
            padding: EdgeInsets.zero,
          ),
          const SizedBox(height: AppSpace.sm),
          if (latest == null)
            Text(
              '还没有决策记录。',
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            )
          else ...[
            Text(
              kindLabel(latest['kind']?.toString()),
              style: AppType.secondary(
                context,
              ).copyWith(fontWeight: FontWeight.w600),
            ),
            Text(
              '${latest['from'] ?? '—'}${(latest['to'] ?? '').toString().isEmpty ? '' : ' → ${latest['to']}'}',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: AppType.caption(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
            Text(
              '${clockText(latest['time']?.toString())} · ${reasonLabel(latest['reason_code']?.toString())}',
              style: AppType.micro(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
        ],
      ),
    );
  }
}

class _ResultsSummaryCard extends StatelessWidget {
  const _ResultsSummaryCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final best = state.best;
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '结果摘要',
            glyph: Glyphs.chart,
            padding: EdgeInsets.zero,
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            '${state.results.length}',
            style: AppType.figure(context, style: AppTextStyle.title2),
          ),
          Text(
            '个节点有结果',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            best == null
                ? '尚未测速'
                : '最高分 ${((best['score'] as num?)?.toDouble() ?? 0).toStringAsFixed(3)}',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          Text(
            state.lastTestAt == null ? '' : timeAgo(state.lastTestAt),
            style: AppType.micro(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }
}

class _SubscribeCard extends StatefulWidget {
  const _SubscribeCard({required this.state});

  final AppState state;

  @override
  State<_SubscribeCard> createState() => _SubscribeCardState();
}

class _SubscribeCardState extends State<_SubscribeCard> {
  final TextEditingController _subscription = TextEditingController();

  AppState get state => widget.state;

  @override
  void dispose() {
    _subscription.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final subscriptions = state.subscriptions.map((path) {
      final value = path.toString();
      return (value: value, label: baseName(value));
    }).toList();
    if (state.settings['config_path'] != null &&
        (state.settings['config_path'] as String).isNotEmpty &&
        !subscriptions.any(
          (item) => item.value == state.settings['config_path'],
        )) {
      subscriptions.insert(0, (
        value: state.settings['config_path'] as String,
        label: baseName(state.settings['config_path'] as String),
      ));
    }
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '订阅',
            glyph: Glyphs.download,
            padding: EdgeInsets.zero,
          ),
          const SizedBox(height: AppSpace.sm),
          AppSelect<String>(
            value: (state.settings['config_path'] ?? '').toString(),
            label: '订阅文件',
            hint: '选择订阅文件',
            items: subscriptions.isEmpty
                ? const [(value: '', label: '没有自动找到订阅文件')]
                : subscriptions,
            onChanged: (value) => state.updateSetting('config_path', value),
          ),
          const SizedBox(height: AppSpace.sm),
          Row(
            spacing: AppSpace.sm,
            children: [
              Expanded(
                child: AppTextField(
                  controller: _subscription,
                  hint: 'https://…/subscribe?token=…',
                ),
              ),
              AppButton(
                label: '导入',
                onPressed: state.busy
                    ? null
                    : () async {
                        await state.importSubscription(_subscription.text);
                        _subscription.clear();
                      },
              ),
            ],
          ),
          if (state.subscriptions.isEmpty)
            Padding(
              padding: const EdgeInsets.only(top: AppSpace.sm),
              child: Text(
                '也可以直接导入订阅链接。',
                style: AppType.micro(
                  context,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
        ],
      ),
    );
  }
}

class _DownloadCard extends StatefulWidget {
  const _DownloadCard({required this.state});

  final AppState state;

  @override
  State<_DownloadCard> createState() => _DownloadCardState();
}

class _DownloadCardState extends State<_DownloadCard> {
  late final TextEditingController _url = TextEditingController(
    text: widget.state.downloadUrl,
  );

  AppState get state => widget.state;

  @override
  void dispose() {
    _url.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '下载模式',
            glyph: Glyphs.upload,
            padding: EdgeInsets.zero,
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            '先测速选前 5 个，再小范围下载探测；只在开始前切换一次。',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.md),
          LayoutBuilder(
            builder: (context, constraints) {
              final narrow = constraints.maxWidth < 520;
              final field = AppTextField(
                controller: _url,
                onChanged: state.setDownloadUrl,
                label: '目标文件 URL',
                hint: 'https://example.com/file.apk',
              );
              final button = AppButton(
                label: state.busy && state.busyKind == 'download'
                    ? '探测中…'
                    : '测速并选节点',
                icon: Glyphs.download,
                loading: state.busy && state.busyKind == 'download',
                onPressed: state.busy ? null : state.downloadMode,
              );
              if (narrow) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  spacing: AppSpace.sm,
                  children: [field, button],
                );
              }
              return Row(
                spacing: AppSpace.sm,
                children: [
                  Expanded(child: field),
                  button,
                ],
              );
            },
          ),
          if (state.downloadResult != null) ...[
            const SizedBox(height: AppSpace.md),
            SurfaceBox(
              child: Text(
                (state.downloadResult!['message'] ?? '').toString(),
                style: AppType.caption(
                  context,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _ResultsCard extends StatelessWidget {
  const _ResultsCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      radius: AppCorner.dashboard,
      padding: const EdgeInsets.all(AppSpace.xl),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '测速结果',
            glyph: Glyphs.speed,
            padding: EdgeInsets.zero,
            actions: [
              if (state.results.isNotEmpty)
                AppButton(
                  label: '查看全部',
                  variant: AppButtonVariant.ghost,
                  onPressed: () => state.setView('nodes'),
                ),
            ],
          ),
          const SizedBox(height: AppSpace.sm),
          if (state.busy)
            const LoadingList(rows: 3)
          else if (state.results.isEmpty)
            const EmptyState(
              icon: Glyphs.speed,
              title: '还没有测速结果',
              body: '点「一键优化」开始。',
            )
          else
            ResultTable(
              rows: state.results,
              selected: state.selected,
              bestName: state.best?['name']?.toString(),
              maxRows: 6,
              onSelect: state.selectNode,
            ),
          if (state.results.isNotEmpty && state.selected != null) ...[
            const SizedBox(height: AppSpace.sm),
            Row(
              children: [
                Expanded(
                  child: Text(
                    '已选择 ${state.selected}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppType.caption(
                      context,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                ),
                AppButton(
                  label: '切换选中节点',
                  variant: AppButtonVariant.tonal,
                  onPressed: state.busy ? null : state.switchSelected,
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}
