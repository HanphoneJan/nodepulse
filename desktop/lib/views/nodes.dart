import 'package:material_ui/material_ui.dart';

import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/typography.dart';
import '../ui/widgets.dart';
import 'result_table.dart';

class NodesView extends StatelessWidget {
  const NodesView({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final query = state.search.trim().toLowerCase();
    final filtered = query.isEmpty
        ? state.results
        : state.results
              .where(
                (row) =>
                    row is Map &&
                    (row['name'] ?? '').toString().toLowerCase().contains(
                      query,
                    ),
              )
              .toList();
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          LayoutBuilder(
            builder: (context, constraints) {
              final narrow = constraints.maxWidth < 560;
              final search = SizedBox(
                width: narrow ? null : 280,
                child: TextField(
                  onChanged: state.setSearch,
                  decoration: const InputDecoration(
                    hintText: '搜索节点',
                    prefixIcon: Icon(Icons.search, size: 18),
                    isDense: true,
                  ),
                ),
              );
              final actions = Row(
                mainAxisSize: MainAxisSize.min,
                spacing: 8,
                children: [
                  GhostButton(
                    label: '刷新状态',
                    icon: Icons.refresh,
                    onPressed: state.refreshStatus,
                  ),
                ],
              );
              if (narrow) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  spacing: 8,
                  children: [
                    search,
                    Align(alignment: Alignment.centerRight, child: actions),
                  ],
                );
              }
              return Row(
                children: [
                  Text(
                    '节点排名',
                    style: appText(theme, AppTextStyle.headline).copyWith(
                      color: scheme.onSurfaceVariant.withValues(alpha: 0.8),
                    ),
                  ),
                  const Spacer(),
                  search,
                  const SizedBox(width: 8),
                  actions,
                ],
              );
            },
          ),
          const SizedBox(height: 12),
          if (state.busy && state.busyKind == 'test')
            const LoadingList()
          else if (state.results.isEmpty)
            const EmptyState(
              icon: Icons.speed,
              title: '还没有测试结果',
              body: '点「一键优化」开始。',
            )
          else if (filtered.isEmpty)
            const EmptyState(
              icon: Icons.search,
              title: '没有匹配的节点',
              body: '换个关键词。',
            )
          else
            ResultTable(
              rows: filtered,
              selected: state.selected,
              bestName: state.best?['name']?.toString(),
              onSelect: state.selectNode,
            ),
          if (filtered.isNotEmpty && state.selected != null) ...[
            const SizedBox(height: 12),
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
