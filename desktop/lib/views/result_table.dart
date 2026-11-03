import 'package:material_ui/material_ui.dart';

import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/flclash/typography.dart';
import '../format.dart';

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
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final visible = rows.whereType<Map>().take(maxRows ?? rows.length).toList();
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth > 620;
        return Column(
          children: [
            Container(
              height: 32,
              padding: const EdgeInsets.symmetric(horizontal: 10),
              child: Row(
                children: [
                  SizedBox(
                    width: 32,
                    child: Text(
                      '#',
                      style: appText(
                        theme,
                        AppTextStyle.footnote,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ),
                  Expanded(
                    child: Text(
                      '节点',
                      style: appText(
                        theme,
                        AppTextStyle.footnote,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ),
                  if (wide) ...[
                    SizedBox(
                      width: 64,
                      child: Text(
                        '成功率',
                        textAlign: TextAlign.right,
                        style: appText(
                          theme,
                          AppTextStyle.footnote,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ),
                    const SizedBox(width: 12),
                    SizedBox(
                      width: 64,
                      child: Text(
                        'P90',
                        textAlign: TextAlign.right,
                        style: appText(
                          theme,
                          AppTextStyle.footnote,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ),
                    const SizedBox(width: 12),
                    SizedBox(
                      width: 64,
                      child: Text(
                        '抖动',
                        textAlign: TextAlign.right,
                        style: appText(
                          theme,
                          AppTextStyle.footnote,
                        ).copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ),
                    const SizedBox(width: 12),
                  ],
                  SizedBox(
                    width: 56,
                    child: Text(
                      '评分',
                      textAlign: TextAlign.right,
                      style: appText(
                        theme,
                        AppTextStyle.footnote,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ),
                ],
              ),
            ),
            for (var i = 0; i < visible.length; i++)
              _ResultRow(
                index: i + 1,
                row: visible[i],
                wide: wide,
                compact: compact,
                selected: visible[i]['name'] == selected,
                best: visible[i]['name'] == bestName,
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

class _ResultRow extends StatelessWidget {
  const _ResultRow({
    required this.index,
    required this.row,
    required this.wide,
    required this.compact,
    required this.selected,
    required this.best,
    this.onTap,
  });

  final int index;
  final Map row;
  final bool wide;
  final bool compact;
  final bool selected;
  final bool best;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final source = (row['source'] ?? '').toString();
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        constraints: BoxConstraints(minHeight: compact ? 48 : 56),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: BoxDecoration(
          color: selected ? scheme.secondaryContainer : null,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            SizedBox(
              width: 32,
              child: Text(
                '$index',
                style: appText(
                  theme,
                  AppTextStyle.footnote,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Row(
                    children: [
                      if (best)
                        Padding(
                          padding: const EdgeInsets.only(right: 4),
                          child: Icon(
                            Icons.check_circle,
                            size: 14,
                            color: scheme.success,
                          ),
                        ),
                      Expanded(
                        child: Text(
                          cleanName((row['name'] ?? '').toString()),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: appText(theme, AppTextStyle.subheadline)
                              .copyWith(
                                color: selected
                                    ? scheme.onSecondaryContainer
                                    : scheme.onSurface,
                                fontWeight: FontWeight.w600,
                              ),
                        ),
                      ),
                    ],
                  ),
                  if (source.isNotEmpty ||
                      (row['type'] ?? '').toString().isNotEmpty)
                    Text(
                      source.isNotEmpty
                          ? source
                          : (row['type'] ?? '').toString(),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: appText(theme, AppTextStyle.footnote).copyWith(
                        color: selected
                            ? scheme.onSecondaryContainer.withValues(
                                alpha: 0.75,
                              )
                            : scheme.onSurfaceVariant,
                      ),
                    ),
                ],
              ),
            ),
            if (wide) ...[
              _Number(width: 64, text: percent(row), selected: selected),
              const SizedBox(width: 12),
              _Number(width: 64, text: ms(row['p90_ms']), selected: selected),
              const SizedBox(width: 12),
              _Number(
                width: 64,
                text: ms(row['jitter_ms']),
                selected: selected,
              ),
              const SizedBox(width: 12),
            ],
            _Number(
              width: 56,
              text: scoreText(row['score']),
              selected: selected,
              bold: true,
            ),
          ],
        ),
      ),
    );
  }
}

class _Number extends StatelessWidget {
  const _Number({
    required this.width,
    required this.text,
    required this.selected,
    this.bold = false,
  });

  final double width;
  final String text;
  final bool selected;
  final bool bold;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return SizedBox(
      width: width,
      child: Text(
        text,
        textAlign: TextAlign.right,
        style: appText(theme, AppTextStyle.footnote).copyWith(
          color: selected
              ? scheme.onSecondaryContainer
              : scheme.onSurfaceVariant,
          fontWeight: bold ? FontWeight.w700 : FontWeight.w400,
          fontFeatures: const [FontFeature.tabularFigures()],
        ),
      ),
    );
  }
}
