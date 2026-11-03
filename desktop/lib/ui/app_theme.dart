import 'package:material_ui/material_ui.dart';

import 'flclash/color.dart';
import 'flclash/typography.dart';

extension AppSurfaces on ColorScheme {
  bool get isDark => brightness == Brightness.dark;

  Color get page => isDark ? surface : surfaceContainer;
  Color get card => isDark ? surfaceContainer : surfaceBright;
  Color get block => isDark ? surfaceContainerHigh : surfaceContainer;
  Color get inset => surfaceContainerHighest;
  Color get floating => isDark ? block : card;
  Color get chrome => isDark ? surfaceContainerLow : surfaceContainerHighest;
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

ThemeData buildAppTheme(Brightness brightness, Color seed) {
  final scheme = seededColorScheme(seedColor: seed, brightness: brightness);
  final base = ThemeData(
    useMaterial3: true,
    colorScheme: scheme,
    scaffoldBackgroundColor: scheme.page,
    canvasColor: scheme.surface,
    dividerColor: scheme.onSurface.withValues(alpha: 0.14),
    visualDensity: VisualDensity.standard,
  ).withAppTypography;

  final buttonText = appText(
    base,
    AppTextStyle.subheadline,
  ).copyWith(fontWeight: FontWeight.w600);
  final pill = RoundedRectangleBorder(borderRadius: BorderRadius.circular(999));

  return base.copyWith(
    cardTheme: CardThemeData(
      color: scheme.card,
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
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
      ).copyWith(color: scheme.onSurface),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(0, 40),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        shape: pill,
        textStyle: buttonText,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(0, 40),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        shape: pill,
        side: BorderSide(color: scheme.outlineVariant),
        textStyle: buttonText,
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        minimumSize: const Size(0, 32),
        padding: const EdgeInsets.symmetric(horizontal: 12),
        shape: pill,
        textStyle: buttonText,
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(
        minimumSize: const Size(40, 40),
        padding: const EdgeInsets.all(8),
        iconSize: 20,
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: scheme.inset,
      contentPadding: const EdgeInsets.fromLTRB(16, 20, 16, 12),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: BorderSide.none,
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: BorderSide.none,
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: BorderSide(color: scheme.primary, width: 2),
      ),
      labelStyle: appText(
        base,
        AppTextStyle.footnote,
      ).copyWith(color: scheme.onSurfaceVariant),
      floatingLabelStyle: appText(
        base,
        AppTextStyle.footnote,
      ).copyWith(color: scheme.primary),
      hintStyle: appText(
        base,
        AppTextStyle.subheadline,
      ).copyWith(color: scheme.onSurfaceVariant.withValues(alpha: 0.7)),
    ),
    dropdownMenuTheme: DropdownMenuThemeData(
      menuStyle: MenuStyle(
        backgroundColor: WidgetStatePropertyAll(scheme.floating),
        elevation: const WidgetStatePropertyAll(2),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        ),
      ),
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: scheme.inverseSurface,
      contentTextStyle: appText(
        base,
        AppTextStyle.subheadline,
      ).copyWith(color: scheme.onInverseSurface),
      behavior: SnackBarBehavior.floating,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
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
  );
}
