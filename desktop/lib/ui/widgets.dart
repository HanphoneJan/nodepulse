import 'dart:math' as math;

import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:material_ui/material_ui.dart';

import 'app_theme.dart';
import 'flclash/color.dart';
import 'flclash/metrics.dart';
import 'flclash/shape.dart';
import 'flclash/surface.dart';
import 'flclash/typography.dart';

/// The app's icon set. Using Cupertino glyphs keeps the line weight and
/// rounded terminals FlClash gets from its own SF-Symbols-style set without
/// porting the whole glyph painter.
abstract final class Glyphs {
  static const dashboard = CupertinoIcons.square_grid_2x2;
  static const dashboardFill = CupertinoIcons.square_grid_2x2_fill;
  static const nodes = CupertinoIcons.rectangle_3_offgrid;
  static const settings = CupertinoIcons.gear;
  static const settingsFill = CupertinoIcons.gear_alt_fill;
  static const history = CupertinoIcons.clock;
  static const clock = CupertinoIcons.clock;
  static const refresh = CupertinoIcons.arrow_clockwise;
  static const link = CupertinoIcons.link;
  static const download = CupertinoIcons.arrow_down_to_line;
  static const downloadCircle = CupertinoIcons.arrow_down_circle;
  static const upload = CupertinoIcons.arrow_up_to_line;
  static const speed = CupertinoIcons.speedometer;
  static const monitor = CupertinoIcons.desktopcomputer;
  static const sun = CupertinoIcons.sun_max;
  static const moon = CupertinoIcons.moon;
  static const check = CupertinoIcons.checkmark;
  static const checkCircle = CupertinoIcons.checkmark_circle_fill;
  static const checkCircleOutline = CupertinoIcons.checkmark_circle;
  static const warning = CupertinoIcons.exclamationmark_triangle;
  static const error = CupertinoIcons.exclamationmark_circle;
  static const info = CupertinoIcons.info_circle;
  static const close = CupertinoIcons.xmark;
  static const closeCircle = CupertinoIcons.xmark_circle;
  static const chevronDown = CupertinoIcons.chevron_down;
  static const chevronRight = CupertinoIcons.chevron_forward;
  static const chevronLeft = CupertinoIcons.chevron_back;
  static const search = CupertinoIcons.search;
  static const play = CupertinoIcons.play_fill;
  static const pause = CupertinoIcons.pause_fill;
  static const stop = CupertinoIcons.stop_fill;
  static const folder = CupertinoIcons.folder;
  static const doc = CupertinoIcons.doc_text;
  static const shield = CupertinoIcons.shield;
  static const wifi = CupertinoIcons.wifi;
  static const globe = CupertinoIcons.globe;
  static const archive = CupertinoIcons.archivebox;
  static const tray = CupertinoIcons.tray;
  static const lock = CupertinoIcons.lock;
  static const eye = CupertinoIcons.eye;
  static const more = CupertinoIcons.ellipsis;
  static const plus = CupertinoIcons.plus;
  static const minus = CupertinoIcons.minus;
  static const trash = CupertinoIcons.trash;
  static const edit = CupertinoIcons.pencil;
  static const flag = CupertinoIcons.flag;
  static const star = CupertinoIcons.star_fill;
  static const timer = CupertinoIcons.timer;
  static const gauge = CupertinoIcons.gauge;
  static const chart = CupertinoIcons.chart_bar;
  static const swap = CupertinoIcons.arrow_up_down_circle;
  static const signal = CupertinoIcons.antenna_radiowaves_left_right;
  static const tune = CupertinoIcons.slider_horizontal_3;
  static const external = CupertinoIcons.arrow_up_right_square;
  static const server = CupertinoIcons.tray_full;
  static const pin = CupertinoIcons.pin;
  static const pinFill = CupertinoIcons.pin_fill;
}

abstract final class AppSpace {
  static const double xs = 4;
  static const double sm = 8;
  static const double md = 12;
  static const double lg = 16;
  static const double xl = 20;
  static const double xxl = 24;
}

/// The physical height a text role needs after the system text scale is
/// applied. Fixed-height controls use it as a floor so Windows text scaling
/// never clips a label.
double _scaledLineHeight(BuildContext context, AppTextStyle style) =>
    MediaQuery.textScalerOf(context).scale(style.size) *
    (style.leading / style.size);

abstract final class AppShadows {
  static List<BoxShadow> e1(ColorScheme scheme) {
    if (scheme.brightness == Brightness.dark) {
      return [
        BoxShadow(
          color: Colors.black.withValues(alpha: 0.5),
          offset: const Offset(0, 1),
          blurRadius: 2,
        ),
        BoxShadow(
          color: Colors.black.withValues(alpha: 0.55),
          offset: const Offset(0, 10),
          blurRadius: 28,
          spreadRadius: -6,
        ),
      ];
    }
    return [
      BoxShadow(
        color: Colors.black.withValues(alpha: 0.05),
        offset: const Offset(0, 0.5),
        blurRadius: 1,
      ),
      BoxShadow(
        color: Colors.black.withValues(alpha: 0.04),
        offset: const Offset(0, 2),
        blurRadius: 6,
      ),
      BoxShadow(
        color: Colors.black.withValues(alpha: 0.09),
        offset: const Offset(0, 12),
        blurRadius: 32,
        spreadRadius: -6,
      ),
    ];
  }
}

/// A surface-aware card. Nested cards step one level down automatically:
/// page → card → block → inset.
class AppCard extends StatelessWidget {
  const AppCard({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(AppSpace.xl),
    this.radius = AppCorner.md,
    this.color,
    this.shadow = false,
    this.selected = false,
    this.invalid = false,
    this.onTap,
    this.level,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;
  final double radius;
  final Color? color;
  final bool shadow;
  final bool selected;
  final bool invalid;
  final VoidCallback? onTap;
  final SurfaceLevel? level;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final surface = level ?? context.raisedSurfaceLevel;
    final shape = AppShape.all(radius);
    final background =
        color ?? (invalid ? scheme.errorContainer : surface.colorOf(scheme));
    final decorated = Container(
      decoration: ShapeDecoration(
        color: background,
        shape: selected
            ? shape.copyWith(
                side: BorderSide(color: scheme.primary, width: 1.5),
              )
            : shape,
        shadows: shadow ? AppShadows.e1(scheme) : null,
      ),
      child: Material(
        type: MaterialType.transparency,
        child: onTap == null
            ? Padding(padding: padding, child: child)
            : InkWell(
                customBorder: shape,
                onTap: onTap,
                child: Padding(padding: padding, child: child),
              ),
      ),
    );
    return SurfaceScope(level: surface, child: decorated);
  }
}

class AppCardHeader extends StatelessWidget {
  const AppCardHeader({
    super.key,
    required this.title,
    this.glyph,
    this.actions = const [],
    this.padding = const EdgeInsets.only(bottom: AppSpace.md),
    this.trailing,
    this.section = false,
  });

  final String title;
  final IconData? glyph;
  final List<Widget> actions;
  final EdgeInsetsGeometry padding;
  final Widget? trailing;

  /// Settings groups use FlClash's 17px card header; dashboard cards keep the
  /// 13px label above a figure.
  final bool section;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final style = section
        ? AppType.cardSectionTitle(context)
        : AppType.cardTitle(context);
    return Padding(
      padding: padding,
      child: ConstrainedBox(
        constraints: BoxConstraints(
          minHeight: section
              ? math.max(26, _scaledLineHeight(context, AppTextStyle.headline))
              : 24,
        ),
        child: Row(
          spacing: AppSpace.sm,
          children: [
            if (glyph != null)
              Icon(
                glyph,
                size: section ? 20 : 18,
                color: scheme.onSurfaceVariant,
              ),
            Expanded(
              child: Text(
                title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: style,
              ),
            ),
            ?trailing,
            if (actions.isNotEmpty)
              Row(
                mainAxisSize: MainAxisSize.min,
                spacing: AppSpace.xs,
                children: actions,
              ),
          ],
        ),
      ),
    );
  }
}

/// Kept as an alias for the older view code.
class InfoHeader extends StatelessWidget {
  const InfoHeader({
    super.key,
    required this.label,
    this.glyph,
    this.actions = const [],
    this.padding = const EdgeInsets.only(bottom: AppSpace.md),
  });

  final String label;
  final IconData? glyph;
  final List<Widget> actions;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) => AppCardHeader(
    title: label,
    glyph: glyph,
    actions: actions,
    padding: padding,
  );
}

enum StatusTone { neutral, ok, warn, error }

class StatusChip extends StatelessWidget {
  const StatusChip({
    super.key,
    required this.label,
    this.tone = StatusTone.neutral,
    this.dot = false,
  });

  final String label;
  final StatusTone tone;
  final bool dot;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final (foreground, background) = switch (tone) {
      StatusTone.ok => (scheme.success, scheme.successContainer),
      StatusTone.warn => (scheme.warning, scheme.warningContainer),
      StatusTone.error => (scheme.error, scheme.errorContainer),
      StatusTone.neutral => (
        scheme.onSurfaceVariant,
        scheme.surfaceContainerHighest,
      ),
    };
    return Container(
      height: math.max(
        26,
        _scaledLineHeight(context, AppTextStyle.caption1) + 10,
      ),
      padding: const EdgeInsets.symmetric(horizontal: 10),
      decoration: ShapeDecoration(color: background, shape: AppShape.full),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        spacing: 5,
        children: [
          if (dot)
            Container(
              width: 6,
              height: 6,
              decoration: BoxDecoration(
                color: foreground,
                shape: BoxShape.circle,
              ),
            ),
          Text(
            label,
            style: AppTextStyle.caption1
                .of(context, emphasized: true)
                .copyWith(color: foreground),
          ),
        ],
      ),
    );
  }
}

enum AppButtonVariant { filled, tonal, outline, ghost }

class AppButton extends StatelessWidget {
  const AppButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.icon,
    this.variant = AppButtonVariant.filled,
    this.loading = false,
    this.enabled = true,
  });

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final AppButtonVariant variant;
  final bool loading;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final text = AppTextStyle.subheadline.of(context, emphasized: true);
    final child = Row(
      mainAxisSize: MainAxisSize.min,
      spacing: AppSpace.sm,
      children: [
        if (loading)
          SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(
              strokeWidth: 2,
              color: switch (variant) {
                AppButtonVariant.filled => scheme.onPrimary,
                AppButtonVariant.tonal => scheme.onSecondaryContainer,
                AppButtonVariant.outline ||
                AppButtonVariant.ghost => scheme.primary,
              },
            ),
          )
        else if (icon != null)
          Icon(icon, size: 18),
        Text(label),
      ],
    );
    final style = ButtonStyle(
      minimumSize: WidgetStatePropertyAll(
        Size(
          0,
          math.max(
            40,
            _scaledLineHeight(context, AppTextStyle.subheadline) + 20,
          ),
        ),
      ),
      padding: const WidgetStatePropertyAll(
        EdgeInsets.symmetric(horizontal: 16),
      ),
      shape: const WidgetStatePropertyAll(AppShape.full),
      textStyle: WidgetStatePropertyAll(text),
      backgroundColor: WidgetStatePropertyAll(switch (variant) {
        AppButtonVariant.filled => scheme.primary,
        AppButtonVariant.tonal => scheme.secondaryContainer,
        AppButtonVariant.outline ||
        AppButtonVariant.ghost => Colors.transparent,
      }),
      foregroundColor: WidgetStatePropertyAll(switch (variant) {
        AppButtonVariant.filled => scheme.onPrimary,
        AppButtonVariant.tonal => scheme.onSecondaryContainer,
        AppButtonVariant.outline => scheme.onSurface,
        AppButtonVariant.ghost => scheme.primary,
      }),
      side: variant == AppButtonVariant.outline
          ? WidgetStatePropertyAll(BorderSide(color: scheme.outlineVariant))
          : null,
    );
    final active = enabled && !loading;
    return switch (variant) {
      AppButtonVariant.outline => OutlinedButton(
        style: style,
        onPressed: active ? onPressed : null,
        child: child,
      ),
      AppButtonVariant.ghost => TextButton(
        style: style,
        onPressed: active ? onPressed : null,
        child: child,
      ),
      _ => FilledButton(
        style: style,
        onPressed: active ? onPressed : null,
        child: child,
      ),
    };
  }
}

/// Backwards-compatible name used by the existing views.
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
  Widget build(BuildContext context) => AppButton(
    label: label,
    onPressed: onPressed,
    icon: icon,
    variant: filled ? AppButtonVariant.filled : AppButtonVariant.tonal,
    loading: loading,
    enabled: enabled,
  );
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
  Widget build(BuildContext context) => AppButton(
    label: label,
    onPressed: onPressed,
    icon: icon,
    variant: AppButtonVariant.ghost,
    enabled: enabled,
  );
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
    final index = options.indexWhere((option) => option.value == value);
    final count = options.length;
    return Container(
      height: math.max(
        44,
        _scaledLineHeight(context, AppTextStyle.subheadline) + 24,
      ),
      padding: const EdgeInsets.all(4),
      decoration: ShapeDecoration(
        color: context.raisedSurfaceLevel.colorOf(scheme),
        shape: AppShape.full,
      ),
      child: Stack(
        children: [
          if (index >= 0)
            AnimatedAlign(
              duration: const Duration(milliseconds: 250),
              curve: Curves.easeOutCubic,
              alignment: AlignmentDirectional(
                count == 1 ? 0 : index * 2 / (count - 1) - 1,
                0,
              ),
              child: FractionallySizedBox(
                widthFactor: 1 / count,
                heightFactor: 1,
                child: DecoratedBox(
                  decoration: ShapeDecoration(
                    color: scheme.primary,
                    shape: AppShape.full,
                  ),
                ),
              ),
            ),
          Row(
            children: [
              for (final option in options)
                Expanded(
                  child: Semantics(
                    selected: option.value == value,
                    button: true,
                    child: FocusRing(
                      shape: AppShape.full,
                      standsOff: option.value == value,
                      child: Material(
                        type: MaterialType.transparency,
                        shape: AppShape.full,
                        clipBehavior: Clip.antiAlias,
                        child: InkWell(
                          customBorder: AppShape.full,
                          onTap: option.value == value
                              ? () {}
                              : () => onChanged(option.value),
                          child: Center(
                            child: AnimatedDefaultTextStyle(
                              duration: const Duration(milliseconds: 200),
                              style: AppTextStyle.subheadline
                                  .of(
                                    context,
                                    emphasized: option.value == value,
                                  )
                                  .copyWith(
                                    color: option.value == value
                                        ? scheme.onPrimary
                                        : scheme.onSurfaceVariant,
                                  ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                spacing: 6,
                                children: [
                                  if (option.icon != null)
                                    Icon(
                                      option.icon,
                                      size: 16,
                                      color: option.value == value
                                          ? scheme.onPrimary
                                          : scheme.onSurfaceVariant,
                                    ),
                                  Flexible(
                                    child: Text(
                                      option.label,
                                      maxLines: 1,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

class AppSwitch extends StatelessWidget {
  const AppSwitch({
    super.key,
    required this.value,
    required this.onChanged,
    this.enabled = true,
  });

  final bool value;
  final ValueChanged<bool>? onChanged;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final active = enabled ? onChanged : null;
    return Semantics(
      checked: value,
      enabled: enabled,
      child: GestureDetector(
        onTap: active == null ? null : () => active(!value),
        child: SizedBox(
          width: 52,
          height: 44,
          child: Center(
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOutCubic,
              width: 52,
              height: 32,
              padding: const EdgeInsets.all(2),
              decoration: ShapeDecoration(
                color: value ? scheme.primary : scheme.inset,
                shape: AppShape.full,
              ),
              child: AnimatedAlign(
                duration: const Duration(milliseconds: 200),
                curve: Curves.easeOutCubic,
                alignment: value
                    ? AlignmentDirectional.centerEnd
                    : AlignmentDirectional.centerStart,
                child: Container(
                  width: 28,
                  height: 28,
                  decoration: ShapeDecoration(
                    color: value ? scheme.onPrimary : scheme.outline,
                    shape: AppShape.full,
                    shadows: value
                        ? [
                            BoxShadow(
                              color: Colors.black.withValues(alpha: 0.18),
                              blurRadius: 4,
                              offset: const Offset(0, 1),
                            ),
                          ]
                        : null,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class AppTextField extends StatefulWidget {
  const AppTextField({
    super.key,
    this.label,
    this.hint,
    this.initialValue,
    this.controller,
    this.onChanged,
    this.keyboardType,
    this.obscure = false,
    this.suffix,
    this.maxLines = 1,
    this.enabled = true,
  });

  final String? label;
  final String? hint;
  final String? initialValue;
  final TextEditingController? controller;
  final ValueChanged<String>? onChanged;
  final TextInputType? keyboardType;
  final bool obscure;
  final Widget? suffix;
  final int maxLines;
  final bool enabled;

  @override
  State<AppTextField> createState() => _AppTextFieldState();
}

class _AppTextFieldState extends State<AppTextField> {
  late bool _obscured = widget.obscure;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return TextFormField(
      controller: widget.controller,
      initialValue: widget.controller == null ? widget.initialValue : null,
      onChanged: widget.onChanged,
      keyboardType: widget.keyboardType,
      obscureText: _obscured,
      maxLines: widget.obscure ? 1 : widget.maxLines,
      enabled: widget.enabled,
      decoration: InputDecoration(
        labelText: widget.label,
        hintText: widget.hint,
        border: AppShape.input.copyWith(borderSide: BorderSide.none),
        enabledBorder: AppShape.input.copyWith(borderSide: BorderSide.none),
        disabledBorder: AppShape.input.copyWith(borderSide: BorderSide.none),
        focusedBorder: AppShape.input.copyWith(
          borderSide: BorderSide(color: scheme.primary, width: 2),
        ),
        errorBorder: AppShape.input.copyWith(
          borderSide: BorderSide(color: scheme.error),
        ),
        focusedErrorBorder: AppShape.input.copyWith(
          borderSide: BorderSide(color: scheme.error, width: 2),
        ),
        suffixIcon: widget.obscure
            ? IconButton(
                tooltip: _obscured ? '显示' : '隐藏',
                icon: Icon(Glyphs.eye, size: 18),
                onPressed: () => setState(() => _obscured = !_obscured),
              )
            : widget.suffix,
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
    this.height = 56,
  });

  final T value;
  final List<({T value, String label})> items;
  final ValueChanged<T> onChanged;
  final String? label;
  final double height;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final hasValue = items.any((item) => item.value == value);
    return SizedBox(
      height: height,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        decoration: ShapeDecoration(
          color: scheme.inset,
          shape: AppShape.all(AppCorner.md),
        ),
        child: DropdownButtonHideUnderline(
          child: DropdownButton<T>(
            value: hasValue ? value : null,
            isExpanded: true,
            hint: label == null
                ? null
                : Text(
                    label!,
                    style: AppType.secondary(
                      context,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
            icon: Icon(
              Glyphs.chevronDown,
              size: 18,
              color: scheme.onSurfaceVariant,
            ),
            style: AppType.secondary(context).copyWith(color: scheme.onSurface),
            dropdownColor: scheme.floating,
            borderRadius: BorderRadius.circular(AppCorner.md),
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
    this.height = listRowMinHeight,
  });

  final String title;
  final String? subtitle;
  final Widget? trailing;
  final VoidCallback? onTap;
  final double height;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final content = ConstrainedBox(
      constraints: BoxConstraints(
        minHeight: math.max(
          height,
          _scaledLineHeight(context, AppTextStyle.body) +
              (subtitle == null
                  ? 0
                  : _scaledLineHeight(context, AppTextStyle.subheadline)) +
              16,
        ),
      ),
      child: Padding(
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
                    style: AppType.body(context),
                  ),
                  if (subtitle != null)
                    Text(
                      subtitle!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: AppType.secondary(
                        context,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                ],
              ),
            ),
            if (trailing != null) ...[
              const SizedBox(width: AppSpace.md),
              trailing!,
            ],
          ],
        ),
      ),
    );
    if (onTap == null) return content;
    return InkWell(
      borderRadius: BorderRadius.circular(AppCorner.md),
      onTap: onTap,
      child: content,
    );
  }
}

class SurfaceBox extends StatelessWidget {
  const SurfaceBox({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(AppSpace.lg),
    this.level,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;
  final SurfaceLevel? level;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final resolved = level ?? context.raisedSurfaceLevel;
    return Container(
      padding: padding,
      decoration: ShapeDecoration(
        color: resolved.colorOf(scheme),
        shape: AppShape.all(AppCorner.lg),
      ),
      child: SurfaceScope(level: resolved, child: child),
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
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpace.lg,
        vertical: AppSpace.xxl,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 32, color: scheme.onSurfaceVariant),
          const SizedBox(height: AppSpace.sm),
          Text(
            title,
            textAlign: TextAlign.center,
            style: AppType.body(context).copyWith(fontWeight: FontWeight.w600),
          ),
          if (body != null) ...[
            const SizedBox(height: AppSpace.xs),
            Text(
              body!,
              textAlign: TextAlign.center,
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
          if (action != null) ...[const SizedBox(height: AppSpace.md), action!],
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
    final scheme = Theme.of(context).colorScheme;
    final ratio = value.clamp(0.0, 1.0);
    return SizedBox(
      width: 128,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '推荐节点',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.xs),
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
                      style: AppType.figure(
                        context,
                        style: AppTextStyle.title2,
                      ),
                    ),
                    Text(
                      '评分',
                      style: AppType.micro(
                        context,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: AppSpace.xs),
          Text(
            bestName,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppType.caption(
              context,
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
            decoration: ShapeDecoration(
              color: scheme.block.withValues(alpha: 0.7 - i * 0.15),
              shape: AppShape.all(AppCorner.md),
            ),
          ),
          if (i != rows - 1) const SizedBox(height: AppSpace.sm),
        ],
      ],
    );
  }
}

class NodeMetrics {
  const NodeMetrics({
    required this.p90,
    required this.jitter,
    required this.score,
  });

  final int p90;
  final int jitter;
  final double score;

  factory NodeMetrics.of(Map row) => NodeMetrics(
    p90: (row['p90_ms'] as num?)?.toInt() ?? 0,
    jitter: (row['jitter_ms'] as num?)?.toInt() ?? 0,
    score: (row['score'] as num?)?.toDouble() ?? 0,
  );
}

class NodeCard extends StatelessWidget {
  const NodeCard({
    super.key,
    required this.rank,
    required this.name,
    this.source = '',
    required this.metrics,
    this.successText,
    this.selected = false,
    this.best = false,
    this.failed = false,
    this.onTap,
  });

  final int rank;
  final String name;
  final String source;
  final NodeMetrics metrics;
  final String? successText;
  final bool selected;
  final bool best;
  final bool failed;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      radius: AppCorner.lg,
      padding: const EdgeInsets.all(AppSpace.lg),
      selected: selected,
      onTap: onTap,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            spacing: AppSpace.sm,
            children: [
              Container(
                width: 24,
                height: 24,
                alignment: Alignment.center,
                decoration: ShapeDecoration(
                  color: scheme.inset,
                  shape: AppShape.all(AppCorner.sm),
                ),
                child: Text(
                  '$rank',
                  style: AppType.micro(context).copyWith(
                    color: scheme.onSurfaceVariant,
                    fontFeatures: const [FontFeature.tabularFigures()],
                  ),
                ),
              ),
              Expanded(
                child: Text(
                  name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: AppType.secondary(
                    context,
                  ).copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              if (best) const StatusChip(label: '推荐', tone: StatusTone.ok),
              if (selected)
                Icon(Glyphs.checkCircle, size: 18, color: scheme.primary),
            ],
          ),
          if (source.isNotEmpty) ...[
            const SizedBox(height: 2),
            Text(
              source,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: AppType.caption(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
          const SizedBox(height: AppSpace.md),
          Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                metrics.score.toStringAsFixed(3),
                style: AppType.figure(
                  context,
                  style: AppTextStyle.title3,
                  color: failed ? scheme.error : scheme.onSurface,
                ),
              ),
              const SizedBox(width: AppSpace.sm),
              Expanded(
                child: Wrap(
                  alignment: WrapAlignment.end,
                  spacing: AppSpace.md,
                  runSpacing: 4,
                  children: [
                    _Metric(label: '成功率', value: successText ?? '—'),
                    _Metric(label: 'P90', value: _ms(metrics.p90)),
                    _Metric(label: '抖动', value: _ms(metrics.jitter)),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  static String _ms(int value) => value <= 0 ? '—' : '${value}ms';
}

class NodeRow extends StatelessWidget {
  const NodeRow({
    super.key,
    required this.rank,
    required this.name,
    this.source = '',
    required this.metrics,
    this.successText,
    this.selected = false,
    this.best = false,
    this.failed = false,
    this.wide = true,
    this.onTap,
  });

  final int rank;
  final String name;
  final String source;
  final NodeMetrics metrics;
  final String? successText;
  final bool selected;
  final bool best;
  final bool failed;
  final bool wide;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final content = Container(
      constraints: const BoxConstraints(minHeight: listRowMinHeight),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: ShapeDecoration(
        color: selected ? scheme.primaryContainer : Colors.transparent,
        shape: AppShape.all(AppCorner.md),
      ),
      child: Row(
        children: [
          SizedBox(
            width: 28,
            child: Text(
              '$rank',
              style: AppType.caption(
                context,
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
                          Glyphs.checkCircle,
                          size: 14,
                          color: scheme.success,
                        ),
                      ),
                    Expanded(
                      child: Text(
                        name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppType.secondary(context).copyWith(
                          fontWeight: FontWeight.w600,
                          color: failed ? scheme.error : scheme.onSurface,
                        ),
                      ),
                    ),
                  ],
                ),
                if (source.isNotEmpty)
                  Text(
                    source,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppType.caption(
                      context,
                    ).copyWith(color: scheme.onSurfaceVariant),
                  ),
              ],
            ),
          ),
          if (wide) ...[
            _Number(width: 64, text: successText ?? '—'),
            _Number(width: 64, text: NodeCard._ms(metrics.p90)),
            _Number(width: 64, text: NodeCard._ms(metrics.jitter)),
          ],
          _Number(
            width: 56,
            text: metrics.score.toStringAsFixed(3),
            bold: true,
            color: failed ? scheme.error : null,
          ),
        ],
      ),
    );
    if (onTap == null) return content;
    return InkWell(
      borderRadius: BorderRadius.circular(AppCorner.md),
      onTap: onTap,
      child: content,
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.end,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          value,
          style: AppType.caption(context).copyWith(
            fontFeatures: const [FontFeature.tabularFigures()],
            fontWeight: FontWeight.w600,
          ),
        ),
        Text(
          label,
          style: AppType.micro(
            context,
          ).copyWith(color: scheme.onSurfaceVariant),
        ),
      ],
    );
  }
}

class _Number extends StatelessWidget {
  const _Number({
    required this.width,
    required this.text,
    this.bold = false,
    this.color,
  });

  final double width;
  final String text;
  final bool bold;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SizedBox(
      width: width,
      child: Text(
        text,
        textAlign: TextAlign.right,
        style: AppType.caption(context).copyWith(
          color: color ?? scheme.onSurfaceVariant,
          fontWeight: bold ? FontWeight.w700 : FontWeight.w400,
          fontFeatures: const [FontFeature.tabularFigures()],
        ),
      ),
    );
  }
}

Future<bool> showAppConfirm(
  BuildContext context, {
  required String title,
  required String message,
  String confirmLabel = '确定',
  String cancelLabel = '取消',
  bool destructive = false,
}) async {
  final result = await showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(title),
      content: Text(message),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(false),
          child: Text(cancelLabel),
        ),
        FilledButton(
          style: destructive
              ? FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.error,
                  foregroundColor: Theme.of(context).colorScheme.onError,
                )
              : null,
          onPressed: () => Navigator.of(context).pop(true),
          child: Text(confirmLabel),
        ),
      ],
    ),
  );
  return result ?? false;
}
