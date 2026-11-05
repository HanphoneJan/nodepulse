import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:nodepulse/ui/app_theme.dart';
import 'package:nodepulse/ui/flclash/typography.dart';

void main() {
  testWidgets('text roles use even leading like FlClash', (tester) async {
    late TextStyle style;
    await tester.pumpWidget(
      MaterialApp(
        theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
        home: Builder(
          builder: (context) {
            style = AppTextStyle.body.of(context);
            return const SizedBox();
          },
        ),
      ),
    );

    expect(style.leadingDistribution, TextLeadingDistribution.even);
    expect(style.textBaseline, TextBaseline.alphabetic);
  });

  testWidgets('windows uses the CJK UI face as the primary font', (
    tester,
  ) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.windows;

    late TextStyle style;
    await tester.pumpWidget(
      MaterialApp(
        theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
        home: Builder(
          builder: (context) {
            style = AppTextStyle.subheadline.of(context);
            return const SizedBox();
          },
        ),
      ),
    );

    expect(style.fontFamily, 'Microsoft YaHei UI');
    expect(style.fontFamilyFallback, contains('Microsoft YaHei'));
    debugDefaultTargetPlatformOverride = null;
  });

  testWidgets('micro text is FlClash caption1 size', (tester) async {
    late TextStyle style;
    await tester.pumpWidget(
      MaterialApp(
        theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
        home: Builder(
          builder: (context) {
            style = AppType.micro(context);
            return const SizedBox();
          },
        ),
      ),
    );

    expect(style.fontSize, AppTextStyle.caption1.size);
  });
}
