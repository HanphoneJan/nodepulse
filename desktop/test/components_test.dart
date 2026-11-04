import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:nodepulse/ui/app_theme.dart';
import 'package:nodepulse/ui/widgets.dart';

Widget _host(Widget child) => MaterialApp(
  theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
  home: Scaffold(
    body: Center(child: SizedBox(width: 420, child: child)),
  ),
);

void main() {
  testWidgets('segmented choice reports the chosen value', (tester) async {
    var value = 0;
    await tester.pumpWidget(
      _host(
        StatefulBuilder(
          builder: (context, setState) => SegmentedChoice<int>(
            value: value,
            options: const [
              (value: 0, label: '全部', icon: null),
              (value: 1, label: '达标', icon: null),
              (value: 2, label: '失败', icon: null),
            ],
            onChanged: (next) => setState(() => value = next),
          ),
        ),
      ),
    );
    expect(find.text('全部'), findsOneWidget);
    await tester.tap(find.text('失败'));
    await tester.pumpAndSettle();
    expect(value, 2);
  });

  testWidgets('switch toggles its value', (tester) async {
    var value = false;
    await tester.pumpWidget(
      _host(
        StatefulBuilder(
          builder: (context, setState) => AppSwitch(
            value: value,
            onChanged: (next) => setState(() => value = next),
          ),
        ),
      ),
    );
    await tester.tap(find.byType(AppSwitch));
    await tester.pumpAndSettle();
    expect(value, isTrue);
  });

  testWidgets('status chip shows its label', (tester) async {
    await tester.pumpWidget(
      _host(const StatusChip(label: '已连接', tone: StatusTone.ok, dot: true)),
    );
    expect(find.text('已连接'), findsOneWidget);
  });

  testWidgets('node card shows score, metrics and recommendation', (
    tester,
  ) async {
    await tester.pumpWidget(
      _host(
        NodeCard(
          rank: 1,
          name: '香港家宽专线',
          source: '订阅 A',
          metrics: const NodeMetrics(p90: 171, jitter: 103, score: 0.935),
          successText: '100%',
          best: true,
        ),
      ),
    );
    expect(find.text('香港家宽专线'), findsOneWidget);
    expect(find.text('0.935'), findsOneWidget);
    expect(find.text('推荐'), findsOneWidget);
    expect(find.text('171ms'), findsOneWidget);
  });

  testWidgets('card handles taps', (tester) async {
    var tapped = false;
    await tester.pumpWidget(
      _host(AppCard(onTap: () => tapped = true, child: const Text('可点击'))),
    );
    await tester.tap(find.text('可点击'));
    await tester.pumpAndSettle();
    expect(tapped, isTrue);
  });

  testWidgets('confirm dialog returns the chosen action', (tester) async {
    bool? result;
    await tester.pumpWidget(
      MaterialApp(
        theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () async {
                result = await showAppConfirm(
                  context,
                  title: '删除当前环境？',
                  message: '这会移除该环境。',
                  confirmLabel: '删除',
                  destructive: true,
                );
              },
              child: const Text('打开'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('打开'));
    await tester.pumpAndSettle();
    expect(find.text('删除当前环境？'), findsOneWidget);
    await tester.tap(find.text('删除'));
    await tester.pumpAndSettle();
    expect(result, isTrue);
  });
}
