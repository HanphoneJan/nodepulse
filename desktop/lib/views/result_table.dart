import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../ui/app_theme.dart';
import '../ui/widgets.dart';

/// A compact result table for the dashboard. The node page uses [NodeCard].
class ResultTable extends StatelessWidget {
  const ResultTable({
    super.key,
    required this.rows,
    this.selected,
    this.onSelect,
    this.bestName,
    this.maxRows,
    this.compact = false,
  });

  final List<dynamic> rows;
  final String? selected;
  final ValueChanged<String>? onSelect;
  final String? bestName;
  final int? maxRows;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final visible = rows.whereType<Map>().take(maxRows ?? rows.length).toList();
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth > 620;
        return Column(
          children: [
            Container(
              height: 32,
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Row(
                children: [
                  SizedBox(
                    width: 28,
                    child: Text(
                      '#',
                      style: AppType.caption(
                        context,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ),
                  Expanded(
                    child: Text(
                      '节点',
                      style: AppType.caption(
                        context,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ),
                  if (wide) ...[
                    _HeaderCell(label: '成功率', width: 64),
                    _HeaderCell(label: 'P90', width: 64),
                    _HeaderCell(label: '抖动', width: 64),
                  ],
                  _HeaderCell(label: '评分', width: 56),
                ],
              ),
            ),
            for (var i = 0; i < visible.length; i++)
              NodeRow(
                rank: i + 1,
                name: cleanName((visible[i]['name'] ?? '').toString()),
                source: (visible[i]['source'] ?? '').toString(),
                metrics: NodeMetrics.of(visible[i]),
                successText: percent(visible[i]),
                wide: wide,
                selected: visible[i]['name'] == selected,
                best: visible[i]['name'] == bestName,
                failed:
                    ((visible[i]['success'] as num?)?.toInt() ?? 0) == 0 &&
                    ((visible[i]['total'] as num?)?.toInt() ?? 0) > 0,
                onTap: onSelect == null
                    ? null
                    : () => onSelect!(visible[i]['name'].toString()),
              ),
          ],
        );
      },
    );
  }
}

class _HeaderCell extends StatelessWidget {
  const _HeaderCell({required this.label, required this.width});

  final String label;
  final double width;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: width,
      child: Text(
        label,
        textAlign: TextAlign.right,
        style: AppType.caption(
          context,
        ).copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant),
      ),
    );
  }
}
