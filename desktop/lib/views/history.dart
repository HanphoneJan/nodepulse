import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/flclash/typography.dart';
import '../ui/widgets.dart';

class HistoryView extends StatelessWidget {
  const HistoryView({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '自动优化事件', glyph: Icons.history),
          if (state.history.isEmpty)
            const EmptyState(
              icon: Icons.history,
              title: '暂无事件',
              body: '开启自动优化后显示记录。',
            )
          else
            for (final event in state.history.whereType<Map>())
              _EventRow(event: event),
        ],
      ),
    );
  }
}

class _EventRow extends StatelessWidget {
  const _EventRow({required this.event});

  final Map event;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final bad =
        event['ok'] != true ||
        event['kind'] == 'rollback' ||
        event['kind'] == 'error';
    final from = cleanName((event['from'] ?? '—').toString());
    final to = cleanName((event['to'] ?? '').toString());
    final currentScore = (event['current_score'] as num?)?.toDouble() ?? 0;
    final bestScore = (event['best_score'] as num?)?.toDouble() ?? 0;
    return Container(
      constraints: const BoxConstraints(minHeight: 56),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: scheme.inset,
        borderRadius: BorderRadius.circular(8),
      ),
      margin: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: 12,
        children: [
          SizedBox(
            width: 92,
            child: Text(
              clockText(event['time']?.toString()),
              style: appText(
                theme,
                AppTextStyle.footnote,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      kindLabel(event['kind']?.toString()),
                      style: appText(theme, AppTextStyle.subheadline).copyWith(
                        fontWeight: FontWeight.w600,
                        color: bad ? scheme.error : scheme.onSurface,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        '$from${to.isEmpty ? '' : ' → $to'}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: appText(
                          theme,
                          AppTextStyle.footnote,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ),
                  ],
                ),
                if ((event['reason_code'] ?? '').toString().isNotEmpty)
                  Text(
                    reasonLabel(event['reason_code']?.toString()),
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                if (currentScore > 0 || bestScore > 0)
                  Text(
                    '当前 ${currentScore.toStringAsFixed(3)} · 候选 ${bestScore.toStringAsFixed(3)}',
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
                if (event['rollback_attempted'] == true)
                  Text(
                    event['rollback_ok'] == true ? '已回滚到原节点' : '回滚失败，请手动检查',
                    style: appText(theme, AppTextStyle.footnote).copyWith(
                      color: event['rollback_ok'] == true
                          ? scheme.warning
                          : scheme.error,
                    ),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
