import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/widgets.dart';

enum _HistoryFilter { all, switch_, rollback, error }

class HistoryView extends StatefulWidget {
  const HistoryView({super.key, required this.state});

  final AppState state;

  @override
  State<HistoryView> createState() => _HistoryViewState();
}

class _HistoryViewState extends State<HistoryView> {
  _HistoryFilter _filter = _HistoryFilter.all;

  AppState get state => widget.state;

  bool _matches(Map event) {
    final kind = event['kind']?.toString();
    return switch (_filter) {
      _HistoryFilter.all => true,
      _HistoryFilter.switch_ => kind == 'switch',
      _HistoryFilter.rollback => kind == 'rollback',
      _HistoryFilter.error => kind == 'error',
    };
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final events = state.history.whereType<Map>().where(_matches).toList();
    final groups = <String, List<Map>>{};
    for (final event in events) {
      final clock = clockText(event['time']?.toString());
      final day = clock.length >= 5 ? clock.substring(0, 5) : '未知日期';
      groups.putIfAbsent(day, () => []).add(event);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: AppSpace.lg,
      children: [
        LayoutBuilder(
          builder: (context, constraints) {
            final narrow = constraints.maxWidth < 760;
            final count = Text(
              '${events.length} 条记录',
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            );
            final filter = SizedBox(
              width: narrow ? double.infinity : 360,
              child: SegmentedChoice<_HistoryFilter>(
                value: _filter,
                options: const [
                  (value: _HistoryFilter.all, label: '全部', icon: null),
                  (value: _HistoryFilter.switch_, label: '切换', icon: null),
                  (value: _HistoryFilter.rollback, label: '回滚', icon: null),
                  (value: _HistoryFilter.error, label: '错误', icon: null),
                ],
                onChanged: (value) => setState(() => _filter = value),
              ),
            );
            if (narrow) {
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: AppSpace.md,
                children: [count, filter],
              );
            }
            return Row(
              spacing: AppSpace.md,
              children: [count, const Spacer(), filter],
            );
          },
        ),
        AppCard(
          padding: const EdgeInsets.fromLTRB(
            AppSpace.xl,
            AppSpace.lg,
            AppSpace.xl,
            AppSpace.xl,
          ),
          child: events.isEmpty
              ? const EmptyState(
                  icon: Glyphs.history,
                  title: '暂无事件',
                  body: '开启自动优化后显示记录。',
                )
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    for (final entry in groups.entries) ...[
                      Padding(
                        padding: const EdgeInsets.only(
                          top: AppSpace.md,
                          bottom: AppSpace.xs,
                        ),
                        child: Text(
                          _dayLabel(entry.key),
                          style: AppType.caption(
                            context,
                          ).copyWith(color: scheme.onSurfaceVariant),
                        ),
                      ),
                      for (var i = 0; i < entry.value.length; i++)
                        _EventRow(
                          event: entry.value[i],
                          last: i == entry.value.length - 1,
                        ),
                    ],
                  ],
                ),
        ),
      ],
    );
  }

  String _dayLabel(String day) =>
      day == '未知日期' ? day : '${day.replaceFirst('-', ' 月 ')} 日';
}

class _EventRow extends StatelessWidget {
  const _EventRow({required this.event, required this.last});

  final Map event;
  final bool last;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final kind = event['kind']?.toString();
    final bad = event['ok'] != true || kind == 'rollback' || kind == 'error';
    final tone = switch (kind) {
      'error' => StatusTone.error,
      'rollback' => StatusTone.warn,
      'switch' => StatusTone.ok,
      _ => StatusTone.neutral,
    };
    final from = cleanName((event['from'] ?? '—').toString());
    final to = cleanName((event['to'] ?? '').toString());
    final currentScore = (event['current_score'] as num?)?.toDouble() ?? 0;
    final bestScore = (event['best_score'] as num?)?.toDouble() ?? 0;
    final time = clockText(event['time']?.toString());
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(
            width: 72,
            child: Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                time.length > 5 ? time.substring(6) : time,
                style: AppType.caption(context).copyWith(
                  color: scheme.onSurfaceVariant,
                  fontFeatures: const [FontFeature.tabularFigures()],
                ),
              ),
            ),
          ),
          SizedBox(
            width: 20,
            child: Column(
              children: [
                Container(
                  width: 9,
                  height: 9,
                  margin: const EdgeInsets.only(top: 4),
                  decoration: BoxDecoration(
                    color: bad ? scheme.error : scheme.success,
                    shape: BoxShape.circle,
                  ),
                ),
                if (!last)
                  Expanded(
                    child: Container(
                      width: 1,
                      color: scheme.outlineVariant.withValues(alpha: 0.5),
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(width: AppSpace.md),
          Expanded(
            child: Padding(
              padding: EdgeInsets.only(bottom: last ? 0 : AppSpace.lg),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    spacing: AppSpace.sm,
                    children: [
                      StatusChip(label: kindLabel(kind), tone: tone),
                      Expanded(
                        child: Text(
                          '$from${to.isEmpty ? '' : ' → $to'}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: AppType.secondary(context).copyWith(
                            fontWeight: FontWeight.w600,
                            color: bad ? scheme.error : scheme.onSurface,
                          ),
                        ),
                      ),
                    ],
                  ),
                  if ((event['reason_code'] ?? '').toString().isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        reasonLabel(event['reason_code']?.toString()),
                        style: AppType.caption(
                          context,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ),
                  if (currentScore > 0 || bestScore > 0)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        '当前 ${currentScore.toStringAsFixed(3)} · 候选 ${bestScore.toStringAsFixed(3)}',
                        style: AppType.caption(
                          context,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ),
                  if (event['rollback_attempted'] == true)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        event['rollback_ok'] == true ? '已回滚到原节点' : '回滚失败，请手动检查',
                        style: AppType.caption(context).copyWith(
                          color: event['rollback_ok'] == true
                              ? scheme.warning
                              : scheme.error,
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
