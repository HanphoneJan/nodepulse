import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:nodepulse/ui/app_theme.dart';
import 'package:nodepulse/ui/window_header.dart';

void main() {
  testWidgets('window header offers the FlClash controls', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
        home: const Scaffold(body: AppWindowHeader()),
      ),
    );
    await tester.pump();

    expect(find.byTooltip('置顶窗口'), findsOneWidget);
    expect(find.byTooltip('最小化'), findsOneWidget);
    expect(find.byTooltip('最大化'), findsOneWidget);
    expect(find.byTooltip('关闭'), findsOneWidget);
  });
}
