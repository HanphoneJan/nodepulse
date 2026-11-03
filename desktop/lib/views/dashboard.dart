import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/flclash/typography.dart';
import '../ui/widgets.dart';
import 'result_table.dart';

class DashboardView extends StatelessWidget {
  const DashboardView({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: 16,
      children: [
        _Hero(state: state),
        _TwoColumn(
          left: _ConnectCard(state: state),
          right: _DaemonCard(state: state),
        ),
        _DownloadCard(state: state),
        _ResultsCard(state: state),
      ],
    );
  }
}

class _TwoColumn extends StatelessWidget {
  const _TwoColumn({required this.left, required this.right});

  final Widget left;
  final Widget right;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 760) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [left, right],
          );
        }
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 16,
          children: [
            Expanded(child: left),
            Expanded(child: right),
          ],
        );
      },
    );
  }
}

class _Hero extends StatelessWidget {
  const _Hero({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final busy = state.busy;
    return CommonCard(
      child: LayoutBuilder(
        builder: (context, constraints) {
          final narrow = constraints.maxWidth < 620;
          final main = Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                spacing: 8,
                children: [
                  Text(
                    '当前节点',
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                  StatusChip(
                    label: state.status['error'] != null
                        ? '连接失败'
                        : state.connected
                        ? '已连接'
                        : '未连接',
                    tone: state.status['error'] != null
                        ? StatusTone.error
                        : state.connected
                        ? StatusTone.ok
                        : StatusTone.warn,
                  ),
                ],
              ),
              const SizedBox(height: 4),
              Text(
                state.connected ? cleanName(state.currentNode) : '—',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: appText(
                  theme,
                  AppTextStyle.title1,
                ).copyWith(fontWeight: FontWeight.w600),
              ),
              if (state.groupName.isNotEmpty)
                Text(
                  '${state.groupName} · ${state.memberCount} 个成员',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: appText(
                    theme,
                    AppTextStyle.subheadline,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              const SizedBox(height: 16),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  ActionPill(
                    label: busy && state.busyKind == 'oneclick'
                        ? '优化中…'
                        : '一键优化',
                    icon: Icons.play_arrow_rounded,
                    filled: true,
                    loading: busy && state.busyKind == 'oneclick',
                    onPressed: state.oneClick,
                  ),
                  ActionPill(
                    label: busy && state.busyKind == 'test' ? '测速中…' : '只测速',
                    icon: Icons.refresh,
                    loading: busy && state.busyKind == 'test',
                    onPressed: state.testOnly,
                  ),
                  ActionPill(
                    label: busy && state.busyKind == 'all' ? '联合测速中…' : '全部订阅',
                    icon: Icons.format_list_bulleted,
                    loading: busy && state.busyKind == 'all',
                    onPressed: state.testAll,
                  ),
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    spacing: 8,
                    children: [
                      Text(
                        '强度',
                        style: appText(
                          theme,
                          AppTextStyle.footnote,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                      SizedBox(
                        width: 124,
                        child: AppSelect<String>(
                          value: '${state.settings['rounds'] ?? 5}',
                          height: 40,
                          items: const [
                            (value: '3', label: '快速 3 轮'),
                            (value: '5', label: '标准 5 轮'),
                            (value: '10', label: '精确 10 轮'),
                          ],
                          onChanged: (value) => state.updateSetting(
                            'rounds',
                            int.tryParse(value) ?? 5,
                          ),
                        ),
                      ),
                    ],
                  ),
                  if (busy) GhostButton(label: '取消', onPressed: state.cancel),
                ],
              ),
              if (busy) ...[
                const SizedBox(height: 12),
                Row(
                  spacing: 8,
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
                      style: appText(
                        theme,
                        AppTextStyle.footnote,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ],
                ),
              ],
            ],
          );
          final side = state.best != null
              ? ScoreRing(
                  value: (state.best!['score'] as num?)?.toDouble() ?? 0,
                  bestName: cleanName((state.best!['name'] ?? '').toString()),
                )
              : _Pending(
                  members: state.memberCount,
                  rounds: state.settings['rounds'] ?? 5,
                  concurrency: state.settings['concurrency'] ?? 8,
                );
          if (narrow) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              spacing: 16,
              children: [
                main,
                Align(alignment: Alignment.centerLeft, child: side),
              ],
            );
          }
          return Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            spacing: 16,
            children: [
              Expanded(child: main),
              SizedBox(width: 180, child: side),
            ],
          );
        },
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

class _Pending extends StatelessWidget {
  const _Pending({
    required this.members,
    required this.rounds,
    required this.concurrency,
  });

  final int members;
  final dynamic rounds;
  final dynamic concurrency;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return SurfaceBox(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
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
                style: appText(
                  theme,
                  AppTextStyle.footnote,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Row(
            crossAxisAlignment: CrossAxisAlignment.baseline,
            textBaseline: TextBaseline.alphabetic,
            spacing: 6,
            children: [
              Text(
                '$members',
                style: appText(
                  theme,
                  AppTextStyle.title1,
                ).copyWith(fontWeight: FontWeight.w700),
              ),
              Text(
                '个节点',
                style: appText(
                  theme,
                  AppTextStyle.footnote,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ],
          ),
          Text(
            '$rounds 轮 · 并发 $concurrency',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          Text(
            '尚未测速',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }
}

class _ConnectCard extends StatefulWidget {
  const _ConnectCard({required this.state});

  final AppState state;

  @override
  State<_ConnectCard> createState() => _ConnectCardState();
}

class _ConnectCardState extends State<_ConnectCard> {
  final TextEditingController _subscription = TextEditingController();

  AppState get state => widget.state;

  @override
  void dispose() {
    _subscription.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final running = state.clients
        .whereType<Map>()
        .where((client) => client['running'] == true)
        .toList();
    final showControllers = state.controllers.length > 1 || running.isEmpty;
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
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          InfoHeader(
            label: '连接',
            glyph: Icons.link,
            actions: [
              GhostButton(
                label: state.detecting ? '检测中…' : '重新检测',
                onPressed: state.detecting ? null : state.detect,
              ),
            ],
          ),
          if (state.controllers.isEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                running.isEmpty ? '未检测到客户端。' : '控制口未开启，请到客户端打开外部控制。',
                style: appText(
                  theme,
                  AppTextStyle.subheadline,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
          for (final client in running)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: SurfaceBox(
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
                    const SizedBox(width: 8),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            (client['name'] ?? '').toString(),
                            style: appText(theme, AppTextStyle.subheadline),
                          ),
                          Text(
                            (client['hint'] ?? '控制口已就绪').toString(),
                            style: appText(
                              theme,
                              AppTextStyle.footnote,
                            ).copyWith(color: scheme.onSurfaceVariant),
                          ),
                        ],
                      ),
                    ),
                    Text(
                      '运行中',
                      style: appText(
                        theme,
                        AppTextStyle.footnote,
                      ).copyWith(color: scheme.success),
                    ),
                  ],
                ),
              ),
            ),
          if (showControllers)
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final controller in state.controllers.whereType<Map>())
                  InkWell(
                    borderRadius: BorderRadius.circular(8),
                    onTap: () {
                      state.updateSetting('api', controller['url']);
                      if ((controller['secret'] ?? '').toString().isNotEmpty) {
                        state.updateSetting('secret', controller['secret']);
                      }
                      state.refreshStatus();
                    },
                    child: Container(
                      height: 32,
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      decoration: BoxDecoration(
                        color: state.settings['api'] == controller['url']
                            ? scheme.secondaryContainer
                            : null,
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                          color: state.settings['api'] == controller['url']
                              ? scheme.secondaryContainer
                              : scheme.outlineVariant.withValues(alpha: 0.6),
                        ),
                      ),
                      alignment: Alignment.center,
                      child: Text(
                        '${controller['url']}${(controller['version'] ?? '').toString().isEmpty ? '' : ' · ${controller['version']}'}',
                        style: appText(theme, AppTextStyle.footnote).copyWith(
                          color: state.settings['api'] == controller['url']
                              ? scheme.onSecondaryContainer
                              : scheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          const SizedBox(height: 16),
          Divider(
            height: 1,
            color: scheme.outlineVariant.withValues(alpha: 0.4),
          ),
          const SizedBox(height: 12),
          Row(
            spacing: 8,
            children: [
              Icon(
                Icons.download_outlined,
                size: 20,
                color: scheme.onSurfaceVariant,
              ),
              Text(
                '订阅',
                style: appText(theme, AppTextStyle.headline).copyWith(
                  color: scheme.onSurfaceVariant.withValues(alpha: 0.8),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          AppSelect<String>(
            value: (state.settings['config_path'] ?? '').toString(),
            label: '选择订阅文件',
            items: subscriptions.isEmpty
                ? const [(value: '', label: '没有自动找到订阅文件')]
                : subscriptions,
            onChanged: (value) => state.updateSetting('config_path', value),
          ),
          const SizedBox(height: 8),
          Row(
            spacing: 8,
            children: [
              Expanded(
                child: TextField(
                  controller: _subscription,
                  decoration: const InputDecoration(
                    hintText: 'https://…/subscribe?token=…',
                    isDense: true,
                  ),
                  style: appText(theme, AppTextStyle.subheadline),
                ),
              ),
              FilledButton(
                onPressed: state.busy
                    ? null
                    : () async {
                        await state.importSubscription(_subscription.text);
                        _subscription.clear();
                      },
                child: const Text('导入'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _DaemonCard extends StatelessWidget {
  const _DaemonCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    Map? latest;
    for (final item in state.history.whereType<Map>()) {
      latest = item;
      break;
    }
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '后台自动优化', glyph: Icons.settings),
          Text(
            '变差时切换，失败回滚。${state.settings['auto_confirm'] ?? 2} 次确认，余量 ${(((state.settings['auto_margin'] as num?) ?? 0.2) * 100).round()}%。',
            style: appText(
              theme,
              AppTextStyle.subheadline,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(
                child: Text(
                  '自动优化',
                  style: appText(theme, AppTextStyle.subheadline),
                ),
              ),
              Switch(value: state.daemon, onChanged: state.toggleDaemon),
            ],
          ),
          const SizedBox(height: 4),
          Row(
            children: [
              Expanded(
                child: Text(
                  '检查间隔',
                  style: appText(
                    theme,
                    AppTextStyle.footnote,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              ),
              SizedBox(
                width: 124,
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
          const SizedBox(height: 12),
          SurfaceBox(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  spacing: 6,
                  children: [
                    Icon(
                      Icons.monitor_heart_outlined,
                      size: 16,
                      color: scheme.onSurfaceVariant,
                    ),
                    Text(
                      '最近自动选择',
                      style: appText(
                        theme,
                        AppTextStyle.footnote,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                if (latest == null)
                  Text(
                    '还没有决策记录。',
                    style: appText(
                      theme,
                      AppTextStyle.subheadline,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  )
                else ...[
                  Text(
                    kindLabel(latest['kind']?.toString()),
                    style: appText(
                      theme,
                      AppTextStyle.subheadline,
                    ).copyWith(fontWeight: FontWeight.w600),
                  ),
                  Text(
                    '${latest['from'] ?? '—'}${(latest['to'] ?? '').toString().isEmpty ? '' : ' → ${latest['to']}'}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                  Text(
                    '${clockText(latest['time']?.toString())} · ${reasonLabel(latest['reason_code']?.toString())}',
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                ],
              ],
            ),
          ),
          if (state.daemon) ...[
            const SizedBox(height: 8),
            Text(
              '每 ${((state.settings['daemon_interval'] as num?) ?? 300) < 60 ? '${state.settings['daemon_interval']} 秒' : '${(((state.settings['daemon_interval'] as num?) ?? 300) / 60).round()} 分钟'}检查',
              style: appText(
                theme,
                AppTextStyle.footnote,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
        ],
      ),
    );
  }
}

class _DownloadCard extends StatelessWidget {
  const _DownloadCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '下载模式', glyph: Icons.download_outlined),
          Text(
            '先测速选前 5 个，再小范围下载探测；只在开始前切换一次。',
            style: appText(
              theme,
              AppTextStyle.subheadline,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          LayoutBuilder(
            builder: (context, constraints) {
              final narrow = constraints.maxWidth < 560;
              final field = _DownloadField(state: state);
              final button = FilledButton.icon(
                onPressed: state.busy ? null : state.downloadMode,
                icon: const Icon(Icons.download, size: 18),
                label: Text(
                  state.busy && state.busyKind == 'download'
                      ? '探测中…'
                      : '测速并选节点',
                ),
              );
              if (narrow) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  spacing: 8,
                  children: [field, button],
                );
              }
              return Row(
                spacing: 8,
                children: [
                  Expanded(child: field),
                  button,
                ],
              );
            },
          ),
          if (state.downloadResult != null) ...[
            const SizedBox(height: 12),
            SurfaceBox(
              child: Text(
                (state.downloadResult!['message'] ?? '').toString(),
                style: appText(
                  theme,
                  AppTextStyle.footnote,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _DownloadField extends StatefulWidget {
  const _DownloadField({required this.state});

  final AppState state;

  @override
  State<_DownloadField> createState() => _DownloadFieldState();
}

class _DownloadFieldState extends State<_DownloadField> {
  late final TextEditingController controller = TextEditingController(
    text: widget.state.downloadUrl,
  );

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      onChanged: widget.state.setDownloadUrl,
      decoration: const InputDecoration(
        labelText: '目标文件 URL',
        hintText: 'https://example.com/file.apk',
      ),
    );
  }
}

class _ResultsCard extends StatelessWidget {
  const _ResultsCard({required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          InfoHeader(
            label: '测速结果',
            glyph: Icons.speed,
            actions: [
              if (state.results.isNotEmpty)
                GhostButton(
                  label: '查看全部',
                  onPressed: () => state.setView('nodes'),
                ),
            ],
          ),
          if (state.busy)
            LoadingList(rows: 3)
          else if (state.results.isEmpty)
            EmptyState(icon: Icons.speed, title: '还没有测速结果', body: '点「一键优化」开始。')
          else
            ResultTable(
              rows: state.results,
              selected: state.selected,
              bestName: state.best?['name']?.toString(),
              maxRows: 6,
              onSelect: state.selectNode,
            ),
          if (state.results.isNotEmpty && state.selected != null) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: Text(
                    '已选择 ${state.selected}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                ),
                FilledButton.tonal(
                  onPressed: state.busy ? null : state.switchSelected,
                  child: const Text('切换选中节点'),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}
