import 'dart:math' as math;

import 'package:material_ui/material_ui.dart';

import 'app_theme.dart';
import 'flclash/color.dart';
import 'flclash/typography.dart';

class CommonCard extends StatelessWidget {
  const CommonCard({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(16),
    this.color,
  });

  final Widget child;
  final EdgeInsets padding;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: color ?? scheme.card,
        borderRadius: BorderRadius.circular(12),
      ),
      child: child,
    );
  }
}

class InfoHeader extends StatelessWidget {
  const InfoHeader({
    super.key,
    required this.label,
    this.glyph,
    this.actions = const [],
    this.padding = const EdgeInsets.only(bottom: 12),
  });

  final String label;
  final IconData? glyph;
  final List<Widget> actions;
  final EdgeInsets padding;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Padding(
      padding: padding,
      child: Row(
        children: [
          if (glyph != null) ...[
            Icon(glyph, size: 20, color: scheme.onSurfaceVariant),
            const SizedBox(width: 12),
          ],
          Expanded(
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: appText(
                theme,
                AppTextStyle.headline,
              ).copyWith(color: scheme.onSurfaceVariant.withValues(alpha: 0.8)),
            ),
          ),
          if (actions.isNotEmpty) ...[
            const SizedBox(width: 8),
            Row(mainAxisSize: MainAxisSize.min, spacing: 4, children: actions),
          ],
        ],
      ),
    );
  }
}

class StatusChip extends StatelessWidget {
  const StatusChip({
    super.key,
    required this.label,
    this.tone = StatusTone.neutral,
  });

  final String label;
  final StatusTone tone;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = switch (tone) {
      StatusTone.ok => scheme.success,
      StatusTone.warn => scheme.warning,
      StatusTone.error => scheme.error,
      StatusTone.neutral => scheme.onSurfaceVariant,
    };
    return Container(
      height: 28,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.10),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: color.withValues(alpha: 0.45)),
      ),
      alignment: Alignment.center,
      child: Text(
        label,
        style: appText(
          Theme.of(context),
          AppTextStyle.footnote,
        ).copyWith(color: color, fontWeight: FontWeight.w600),
      ),
    );
  }
}

enum StatusTone { neutral, ok, warn, error }

class ActionPill extends StatelessWidget {
  const ActionPill({
    super.key,
    required this.label,
    required this.onPressed,
    this.icon,
    this.filled = false,
    this.loading = false,
    this.enabled = true,
  });

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final bool filled;
  final bool loading;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final style = ButtonStyle(
      minimumSize: const WidgetStatePropertyAll(Size(0, 40)),
      padding: const WidgetStatePropertyAll(
        EdgeInsets.symmetric(horizontal: 16),
      ),
      shape: WidgetStatePropertyAll(
        RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
      ),
      backgroundColor: WidgetStatePropertyAll(
        filled ? scheme.primary : scheme.secondaryContainer,
      ),
      foregroundColor: WidgetStatePropertyAll(
        filled ? scheme.onPrimary : scheme.onSecondaryContainer,
      ),
      textStyle: WidgetStatePropertyAll(
        appText(
          Theme.of(context),
          AppTextStyle.subheadline,
        ).copyWith(fontWeight: FontWeight.w600),
      ),
    );
    final child = Row(
      mainAxisSize: MainAxisSize.min,
      spacing: 8,
      children: [
        if (loading)
          const SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2),
          )
        else if (icon != null)
          Icon(icon, size: 18),
        Text(label),
      ],
    );
    if (filled) {
      return FilledButton(
        style: style,
        onPressed: enabled && !loading ? onPressed : null,
        child: child,
      );
    }
    return FilledButton(
      style: style,
      onPressed: enabled && !loading ? onPressed : null,
      child: child,
    );
  }
}

class GhostButton extends StatelessWidget {
  const GhostButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.icon,
    this.enabled = true,
  });

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return TextButton.icon(
      onPressed: enabled ? onPressed : null,
      icon: icon == null ? const SizedBox.shrink() : Icon(icon, size: 18),
      label: Text(label),
    );
  }
}

class SegmentedChoice<T> extends StatelessWidget {
  const SegmentedChoice({
    super.key,
    required this.value,
    required this.options,
    required this.onChanged,
  });

  final T value;
  final List<({T value, String label, IconData? icon})> options;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      height: 40,
      padding: const EdgeInsets.all(2),
      decoration: BoxDecoration(
        color: scheme.inset,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final option in options)
            GestureDetector(
              onTap: () => onChanged(option.value),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                height: 36,
                padding: const EdgeInsets.symmetric(horizontal: 12),
                decoration: BoxDecoration(
                  color: option.value == value
                      ? scheme.secondaryContainer
                      : null,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  mainAxisAlignment: MainAxisAlignment.center,
                  spacing: 6,
                  children: [
                    if (option.icon != null)
                      Icon(
                        option.icon,
                        size: 16,
                        color: option.value == value
                            ? scheme.onSecondaryContainer
                            : scheme.onSurfaceVariant,
                      ),
                    Text(
                      option.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: appText(Theme.of(context), AppTextStyle.footnote)
                          .copyWith(
                            fontWeight: FontWeight.w600,
                            color: option.value == value
                                ? scheme.onSecondaryContainer
                                : scheme.onSurfaceVariant,
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

class AppSelect<T> extends StatelessWidget {
  const AppSelect({
    super.key,
    required this.value,
    required this.items,
    required this.onChanged,
    this.label,
    this.height = 48,
  });

  final T value;
  final List<({T value, String label})> items;
  final ValueChanged<T> onChanged;
  final String? label;
  final double height;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      height: height,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: BoxDecoration(
        color: scheme.inset,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: scheme.outlineVariant.withValues(alpha: 0.5)),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<T>(
          value: items.any((item) => item.value == value) ? value : null,
          isExpanded: true,
          hint: label == null ? null : Text(label!),
          icon: Icon(
            Icons.keyboard_arrow_down,
            size: 18,
            color: scheme.onSurfaceVariant,
          ),
          style: appText(
            Theme.of(context),
            AppTextStyle.subheadline,
          ).copyWith(color: scheme.onSurface),
          dropdownColor: scheme.floating,
          borderRadius: BorderRadius.circular(12),
          items: [
            for (final item in items)
              DropdownMenuItem<T>(
                value: item.value,
                child: Text(item.label, overflow: TextOverflow.ellipsis),
              ),
          ],
          onChanged: (next) {
            if (next != null) onChanged(next);
          },
        ),
      ),
    );
  }
}

class SettingRow extends StatelessWidget {
  const SettingRow({
    super.key,
    required this.title,
    this.subtitle,
    this.trailing,
    this.onTap,
    this.height = 56,
  });

  final String title;
  final String? subtitle;
  final Widget? trailing;
  final VoidCallback? onTap;
  final double height;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final content = Container(
      constraints: BoxConstraints(minHeight: height),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: appText(
                    theme,
                    AppTextStyle.body,
                  ).copyWith(color: scheme.onSurface),
                ),
                if (subtitle != null)
                  Text(
                    subtitle!,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: appText(
                      theme,
                      AppTextStyle.footnote,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
              ],
            ),
          ),
          if (trailing != null) ...[const SizedBox(width: 12), trailing!],
        ],
      ),
    );
    if (onTap == null) return content;
    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onTap: onTap,
      child: content,
    );
  }
}

class SurfaceBox extends StatelessWidget {
  const SurfaceBox({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(12),
  });

  final Widget child;
  final EdgeInsets padding;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.inset,
        borderRadius: BorderRadius.circular(8),
      ),
      child: child,
    );
  }
}

class EmptyState extends StatelessWidget {
  const EmptyState({
    super.key,
    required this.icon,
    required this.title,
    this.body,
    this.action,
  });

  final IconData icon;
  final String title;
  final String? body;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 24),
      child: Column(
        children: [
          Icon(icon, size: 32, color: scheme.onSurfaceVariant),
          const SizedBox(height: 8),
          Text(
            title,
            textAlign: TextAlign.center,
            style: appText(
              theme,
              AppTextStyle.body,
            ).copyWith(fontWeight: FontWeight.w600),
          ),
          if (body != null) ...[
            const SizedBox(height: 4),
            Text(
              body!,
              textAlign: TextAlign.center,
              style: appText(
                theme,
                AppTextStyle.subheadline,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
          if (action != null) ...[const SizedBox(height: 12), action!],
        ],
      ),
    );
  }
}

class ScoreRing extends StatelessWidget {
  const ScoreRing({super.key, required this.value, required this.bestName});

  final double value;
  final String bestName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final ratio = value.clamp(0.0, 1.0);
    return SizedBox(
      width: 128,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '推荐节点',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 4),
          SizedBox(
            width: 96,
            height: 96,
            child: Stack(
              alignment: Alignment.center,
              children: [
                CustomPaint(
                  size: const Size.square(96),
                  painter: _RingPainter(
                    ratio: ratio,
                    track: scheme.inset,
                    value: scheme.primary,
                  ),
                ),
                Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      '${(ratio * 100).round()}',
                      style: appText(
                        theme,
                        AppTextStyle.title2,
                      ).copyWith(fontWeight: FontWeight.w700),
                    ),
                    Text(
                      '评分',
                      style: appText(
                        theme,
                        AppTextStyle.caption2,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 4),
          Text(
            bestName,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
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

class _RingPainter extends CustomPainter {
  _RingPainter({required this.ratio, required this.track, required this.value});

  final double ratio;
  final Color track;
  final Color value;

  @override
  void paint(Canvas canvas, Size size) {
    const stroke = 8.0;
    final rect = Offset.zero & size;
    final center = rect.center;
    final radius = math.min(size.width, size.height) / 2 - stroke / 2;
    final base = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = stroke
      ..color = track;
    canvas.drawCircle(center, radius, base);
    final arc = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = stroke
      ..strokeCap = StrokeCap.round
      ..color = value;
    canvas.drawArc(
      Rect.fromCircle(center: center, radius: radius),
      -math.pi / 2,
      math.pi * 2 * ratio,
      false,
      arc,
    );
  }

  @override
  bool shouldRepaint(covariant _RingPainter oldDelegate) =>
      oldDelegate.ratio != ratio ||
      oldDelegate.track != track ||
      oldDelegate.value != value;
}

class LoadingList extends StatelessWidget {
  const LoadingList({super.key, this.rows = 3});

  final int rows;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Column(
      children: [
        for (var i = 0; i < rows; i++) ...[
          Container(
            height: 56,
            decoration: BoxDecoration(
              color: scheme.block.withValues(alpha: 0.7 - i * 0.15),
              borderRadius: BorderRadius.circular(8),
            ),
          ),
          const SizedBox(height: 8),
        ],
      ],
    );
  }
}
