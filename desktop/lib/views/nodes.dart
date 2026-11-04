import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/shape.dart';
import '../ui/widgets.dart';

enum _NodeFilter { all, ok, failed }

class NodesView extends StatefulWidget {
  const NodesView({super.key, required this.state});

  final AppState state;

  @override
  State<NodesView> createState() => _NodesViewState();
}

class _NodesViewState extends State<NodesView> {
  _NodeFilter _filter = _NodeFilter.all;

  AppState get state => widget.state;

  bool _failed(Map row) {
    final total = (row['total'] as num?)?.toInt() ?? 0;
    final success = (row['success'] as num?)?.toInt() ?? 0;
    return total > 0 && success / total < 0.8;
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final query = state.search.trim().toLowerCase();
    final base = state.results.whereType<Map>().where((row) {
      if (query.isEmpty) return true;
      return (row['name'] ?? '').toString().toLowerCase().contains(query);
    }).toList();
    final filtered = base.where((row) {
      return switch (_filter) {
        _NodeFilter.all => true,
        _NodeFilter.ok => !_failed(row),
        _NodeFilter.failed => _failed(row),
      };
    }).toList();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: AppSpace.lg,
      children: [
        LayoutBuilder(
          builder: (context, constraints) {
            final narrow = constraints.maxWidth < 760;
            final title = Row(
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              spacing: AppSpace.sm,
              children: [
                Text('节点', style: AppType.section(context)),
                Text(
                  '${filtered.length}',
                  style: AppType.caption(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              ],
            );
            final search = _SearchField(
              width: narrow ? double.infinity : 280,
              onChanged: state.setSearch,
            );
            final filter = SizedBox(
              width: narrow ? double.infinity : 300,
              child: SegmentedChoice<_NodeFilter>(
                value: _filter,
                options: const [
                  (value: _NodeFilter.all, label: '全部', icon: null),
                  (value: _NodeFilter.ok, label: '达标', icon: null),
                  (value: _NodeFilter.failed, label: '失败', icon: null),
                ],
                onChanged: (value) => setState(() => _filter = value),
              ),
            );
            final refresh = AppButton(
              label: '刷新状态',
              icon: Glyphs.refresh,
              variant: AppButtonVariant.outline,
              onPressed: state.refreshStatus,
            );
            if (narrow) {
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                spacing: AppSpace.md,
                children: [
                  Row(children: [title, const Spacer(), refresh]),
                  search,
                  filter,
                ],
              );
            }
            return Row(
              spacing: AppSpace.md,
              children: [title, const Spacer(), search, filter, refresh],
            );
          },
        ),
        if (state.selected != null)
          AppCard(
            padding: const EdgeInsets.symmetric(
              horizontal: AppSpace.lg,
              vertical: AppSpace.sm,
            ),
            child: Row(
              children: [
                Icon(Glyphs.checkCircle, size: 18, color: scheme.primary),
                const SizedBox(width: AppSpace.sm),
                Expanded(
                  child: Text(
                    '已选择 ${cleanName(state.selected)}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppType.secondary(
                      context,
                    ).copyWith(fontWeight: FontWeight.w600),
                  ),
                ),
                AppButton(
                  label: '取消选择',
                  variant: AppButtonVariant.ghost,
                  onPressed: () => state.selectNode(null),
                ),
                const SizedBox(width: AppSpace.sm),
                AppButton(
                  label: '切换节点',
                  icon: Glyphs.swap,
                  onPressed: state.busy ? null : state.switchSelected,
                ),
              ],
            ),
          ),
        if (state.busy && state.busyKind == 'test')
          const LoadingList(rows: 6)
        else if (state.results.isEmpty)
          AppCard(
            child: EmptyState(
              icon: Glyphs.nodes,
              title: '还没有测试结果',
              body: '回到仪表盘点「一键优化」开始。',
            ),
          )
        else if (filtered.isEmpty)
          AppCard(
            child: EmptyState(
              icon: Glyphs.search,
              title: '没有匹配的节点',
              body: query.isEmpty ? '换个筛选条件。' : '换个关键词。',
            ),
          )
        else
          LayoutBuilder(
            builder: (context, constraints) {
              final columns = constraints.maxWidth >= 1080
                  ? 3
                  : (constraints.maxWidth >= 700 ? 2 : 1);
              const gap = AppSpace.md;
              final width =
                  (constraints.maxWidth - gap * (columns - 1)) / columns;
              final best = state.best?['name']?.toString();
              return Wrap(
                spacing: gap,
                runSpacing: gap,
                children: [
                  for (var i = 0; i < filtered.length; i++)
                    SizedBox(
                      width: width,
                      child: NodeCard(
                        rank: i + 1,
                        name: cleanName((filtered[i]['name'] ?? '').toString()),
                        source: (filtered[i]['source'] ?? '').toString(),
                        metrics: NodeMetrics.of(filtered[i]),
                        successText: percent(filtered[i]),
                        selected: filtered[i]['name'] == state.selected,
                        best: filtered[i]['name'] == best,
                        failed: _failed(filtered[i]),
                        onTap: () => state.selectNode(
                          filtered[i]['name'] == state.selected
                              ? null
                              : filtered[i]['name'].toString(),
                        ),
                      ),
                    ),
                ],
              );
            },
          ),
      ],
    );
  }
}

class _SearchField extends StatelessWidget {
  const _SearchField({required this.width, required this.onChanged});

  final double width;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SizedBox(
      width: width,
      height: 44,
      child: TextField(
        onChanged: onChanged,
        style: AppType.secondary(context),
        decoration: InputDecoration(
          hintText: '搜索节点',
          prefixIcon: Icon(
            Glyphs.search,
            size: 18,
            color: scheme.onSurfaceVariant,
          ),
          prefixIconConstraints: const BoxConstraints(
            minWidth: 40,
            minHeight: 40,
          ),
          isDense: true,
          contentPadding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
          border: AppShape.input.copyWith(borderSide: BorderSide.none),
          enabledBorder: AppShape.input.copyWith(borderSide: BorderSide.none),
          focusedBorder: AppShape.input.copyWith(
            borderSide: BorderSide(color: scheme.primary, width: 2),
          ),
        ),
      ),
    );
  }
}
