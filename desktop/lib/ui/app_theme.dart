import 'package:flutter/foundation.dart' show defaultTargetPlatform;
import 'package:material_ui/material_ui.dart';

import 'flclash/color.dart';
import 'flclash/font.dart';
import 'flclash/shape.dart';
import 'flclash/surface.dart';
import 'flclash/typography.dart';

/// Convenience aliases for the FlClash surface levels. The values live in
/// [SurfaceLevel]; this keeps page code short.
extension AppSurfaces on ColorScheme {
  bool get isDark => brightness == Brightness.dark;

  Color get page => SurfaceLevel.page.colorOf(this);
  Color get card => SurfaceLevel.card.colorOf(this);
  Color get block => SurfaceLevel.block.colorOf(this);
  Color get inset => SurfaceLevel.inset.colorOf(this);
  Color get floating => SurfaceLevel.floatingOf(this).colorOf(this);
  Color get chrome => chromeSurface;
}

TextStyle appText(ThemeData theme, AppTextStyle style) {
  final base = theme.textTheme.bodyMedium ?? const TextStyle();
  return base.copyWith(
    fontSize: style.size,
    height: style.leading / style.size,
    fontWeight: style.weight,
    letterSpacing: 0,
  );
}

/// The FlClash text roles used across pages, so no view has to invent a size.
abstract final class AppType {
  static TextStyle pageTitle(BuildContext context) =>
      AppTextStyle.pageTitle.of(context);

  static TextStyle cardTitle(BuildContext context) => AppTextStyle.footnote
      .of(context, emphasized: true)
      .copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant);

  /// FlClash's settings card header: a 17px headline over the card.
  static TextStyle cardSectionTitle(BuildContext context) => AppTextStyle
      .headline
      .of(context)
      .copyWith(
        color: Theme.of(
          context,
        ).colorScheme.onSurfaceVariant.withValues(alpha: 0.8),
      );

  static TextStyle section(BuildContext context) =>
      AppTextStyle.headline.of(context, emphasized: true);

  static TextStyle body(BuildContext context) => AppTextStyle.body.of(context);

  static TextStyle secondary(BuildContext context) =>
      AppTextStyle.subheadline.of(context);

  static TextStyle caption(BuildContext context) =>
      AppTextStyle.footnote.of(context);

  static TextStyle micro(BuildContext context) =>
      AppTextStyle.caption1.of(context);

  static TextStyle nodeTitle(BuildContext context) =>
      AppTextStyle.title1.of(context).copyWith(fontWeight: FontWeight.w600);

  static TextStyle figure(
    BuildContext context, {
    AppTextStyle style = AppTextStyle.display,
    Color? color,
  }) => style
      .of(context, emphasized: true)
      .copyWith(
        color: color,
        fontFeatures: const [FontFeature.tabularFigures()],
        letterSpacing: -0.02 * style.size,
      );

  static TextStyle tabular(BuildContext context, AppTextStyle style) => style
      .of(context)
      .copyWith(fontFeatures: const [FontFeature.tabularFigures()]);
}

ThemeData buildAppTheme(Brightness brightness, Color seed) {
  final scheme = seededColorScheme(seedColor: seed, brightness: brightness);
  final fallback = appFontFamilyFallback(
    platform: defaultTargetPlatform,
    locales: WidgetsBinding.instance.platformDispatcher.locales,
  );

  final desktop = switch (defaultTargetPlatform) {
    TargetPlatform.windows ||
    TargetPlatform.macOS ||
    TargetPlatform.linux => true,
    _ => false,
  };

  final base = ThemeData(
    useMaterial3: true,
    colorScheme: scheme,
    fontFamilyFallback: fallback,
    visualDensity: desktop ? VisualDensity.compact : VisualDensity.standard,
    materialTapTargetSize: desktop
        ? MaterialTapTargetSize.shrinkWrap
        : MaterialTapTargetSize.padded,
    pageTransitionsTheme: const PageTransitionsTheme(
      builders: {
        TargetPlatform.android: FadeForwardsPageTransitionsBuilder(),
        TargetPlatform.iOS: FadeForwardsPageTransitionsBuilder(),
        TargetPlatform.macOS: FadeForwardsPageTransitionsBuilder(),
        TargetPlatform.windows: FadeForwardsPageTransitionsBuilder(),
        TargetPlatform.linux: FadeForwardsPageTransitionsBuilder(),
      },
    ),
  ).withAppShapes.withAppSurfaces.withAppTypography;

  final buttonText = appText(
    base,
    AppTextStyle.subheadline,
  ).copyWith(fontWeight: FontWeight.w600);
  final filled = FilledButton.styleFrom(
    minimumSize: const Size(0, 40),
    padding: const EdgeInsets.symmetric(horizontal: 16),
    shape: AppShape.full,
    textStyle: buttonText,
  );
  final outlined = OutlinedButton.styleFrom(
    minimumSize: const Size(0, 40),
    padding: const EdgeInsets.symmetric(horizontal: 16),
    shape: AppShape.full,
    side: BorderSide(color: scheme.outlineVariant),
    textStyle: buttonText,
  );
  final text = TextButton.styleFrom(
    minimumSize: const Size(0, 36),
    padding: const EdgeInsets.symmetric(horizontal: 12),
    shape: AppShape.full,
    textStyle: buttonText,
  );
  final icon = IconButton.styleFrom(
    minimumSize: const Size(40, 40),
    padding: const EdgeInsets.all(8),
    iconSize: 20,
    shape: AppShape.full,
  );

  return base.copyWith(
    scaffoldBackgroundColor: scheme.page,
    canvasColor: scheme.surface,
    dividerTheme: DividerThemeData(
      color: scheme.onSurface.withValues(alpha: 0.14),
      thickness: 1,
      space: 1,
    ),
    cardTheme: base.cardTheme.copyWith(
      color: scheme.card,
      elevation: 0,
      margin: EdgeInsets.zero,
    ),
    appBarTheme: AppBarTheme(
      backgroundColor: scheme.page,
      foregroundColor: scheme.onSurface,
      elevation: 0,
      scrolledUnderElevation: 0,
      toolbarHeight: 72,
      titleTextStyle: appText(
        base,
        AppTextStyle.pageTitle,
      ).copyWith(fontWeight: FontWeight.w600, color: scheme.onSurface),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: base.filledButtonTheme.style?.merge(filled) ?? filled,
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: base.outlinedButtonTheme.style?.merge(outlined) ?? outlined,
    ),
    textButtonTheme: TextButtonThemeData(
      style: base.textButtonTheme.style?.merge(text) ?? text,
    ),
    iconButtonTheme: IconButtonThemeData(
      style: base.iconButtonTheme.style?.merge(icon) ?? icon,
    ),
    inputDecorationTheme: base.inputDecorationTheme.copyWith(
      filled: true,
      fillColor: scheme.inset,
      contentPadding: const EdgeInsets.fromLTRB(16, 20, 16, 12),
      border: AppShape.input.copyWith(borderSide: BorderSide.none),
      enabledBorder: AppShape.input.copyWith(borderSide: BorderSide.none),
      focusedBorder: AppShape.input.copyWith(
        borderSide: BorderSide(color: scheme.primary, width: 2),
      ),
      disabledBorder: AppShape.input.copyWith(borderSide: BorderSide.none),
      errorBorder: AppShape.input.copyWith(
        borderSide: BorderSide(color: scheme.error),
      ),
      focusedErrorBorder: AppShape.input.copyWith(
        borderSide: BorderSide(color: scheme.error, width: 2),
      ),
      labelStyle: appText(
        base,
        AppTextStyle.footnote,
      ).copyWith(color: scheme.onSurfaceVariant),
      floatingLabelStyle: appText(
        base,
        AppTextStyle.footnote,
      ).copyWith(color: scheme.onSurfaceVariant),
      hintStyle: appText(
        base,
        AppTextStyle.subheadline,
      ).copyWith(color: scheme.onSurfaceVariant.withValues(alpha: 0.7)),
    ),
    dropdownMenuTheme: base.dropdownMenuTheme.copyWith(
      menuStyle: MenuStyle(
        backgroundColor: WidgetStatePropertyAll(scheme.floating),
        elevation: const WidgetStatePropertyAll(2),
        shape: const WidgetStatePropertyAll(AppShape.md),
      ),
    ),
    popupMenuTheme: base.popupMenuTheme.copyWith(
      color: scheme.floating,
      shape: AppShape.md,
      elevation: 2,
      textStyle: appText(base, AppTextStyle.subheadline),
    ),
    dialogTheme: base.dialogTheme.copyWith(
      backgroundColor: scheme.floating,
      shape: AppShape.all(AppCorner.alert),
      titleTextStyle: appText(
        base,
        AppTextStyle.headline,
      ).copyWith(color: scheme.onSurface),
      contentTextStyle: appText(
        base,
        AppTextStyle.subheadline,
      ).copyWith(color: scheme.onSurfaceVariant),
    ),
    bottomSheetTheme: base.bottomSheetTheme.copyWith(
      backgroundColor: scheme.page,
      modalBackgroundColor: scheme.page,
      shape: AppShape.top(AppCorner.xxl),
    ),
    snackBarTheme: base.snackBarTheme.copyWith(
      backgroundColor: scheme.inverseSurface,
      contentTextStyle: appText(
        base,
        AppTextStyle.subheadline,
      ).copyWith(color: scheme.onInverseSurface),
      behavior: SnackBarBehavior.floating,
      elevation: 2,
      shape: AppShape.xs,
    ),
    chipTheme: base.chipTheme.copyWith(
      backgroundColor: scheme.inset,
      side: BorderSide.none,
      shape: AppShape.sm,
      labelStyle: appText(base, AppTextStyle.footnote),
    ),
    switchTheme: SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return scheme.onPrimary;
        return scheme.outline;
      }),
      trackColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return scheme.primary;
        return scheme.inset;
      }),
      trackOutlineColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.selected)) return scheme.primary;
        return scheme.outline;
      }),
    ),
    progressIndicatorTheme: base.progressIndicatorTheme.copyWith(
      color: scheme.primary,
      borderRadius: AppRadius.full,
    ),
    scrollbarTheme: base.scrollbarTheme.copyWith(
      thickness: const WidgetStatePropertyAll(6),
      radius: const Radius.circular(999),
      thumbColor: WidgetStatePropertyAll(
        scheme.onSurface.withValues(alpha: 0.22),
      ),
    ),
  );
}
