import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:nodepulse/state/app_state.dart';
import 'package:nodepulse/ui/app_theme.dart';
import 'package:nodepulse/update/update_checker.dart';
import 'package:nodepulse/views/dashboard.dart';
import 'package:nodepulse/views/history.dart';
import 'package:nodepulse/views/nodes.dart';
import 'package:nodepulse/views/settings.dart';

AppState _state() {
  final state = AppState();
  state.ready = true;
  state.version = '0.3.2';
  state.status = {
    'connected': true,
    'current': '香港家宽专线',
    'group': '自动选择',
    'members': 48,
  };
  state.settings = {
    'rounds': 5,
    'concurrency': 8,
    'auto_confirm': 2,
    'auto_margin': 0.2,
    'auto_min_success': 0.8,
    'daemon_interval': 300,
    'api': 'http://127.0.0.1:9090',
    'profiles': <Map<String, dynamic>>[],
  };
  state.results = [
    {
      'name': '香港家宽专线',
      'source': '订阅 A',
      'success': 5,
      'total': 5,
      'p90_ms': 171,
      'jitter_ms': 103,
      'score': 0.935,
    },
    {
      'name': '美国5号-0.1倍率',
      'source': '订阅 B',
      'success': 4,
      'total': 5,
      'p90_ms': 532,
      'jitter_ms': 21,
      'score': 0.848,
    },
  ];
  state.best = state.results.first as Map<String, dynamic>;
  state.lastTestAt = '2026-11-04T15:00:00Z';
  state.update = const ReleaseInfo(
    version: '0.3.2',
    url: 'https://github.com/HanphoneJan/nodepulse/releases/tag/v0.3.2',
    notes: 'Bug fixes and a smoother update check.',
  );
  state.history = [
    {
      'time': '2026-11-04T14:58:00Z',
      'kind': 'switch',
      'ok': true,
      'from': '美国5号',
      'to': '香港家宽专线',
      'reason_code': 'better-score',
      'current_score': 0.52,
      'best_score': 0.93,
    },
  ];
  return state;
}

Future<void> _pump(
  WidgetTester tester,
  Widget view, {
  required Size size,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MaterialApp(
      theme: buildAppTheme(Brightness.light, const Color(0xFF007AFF)),
      home: Scaffold(body: SingleChildScrollView(child: view)),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('dashboard lays out wide, compact and mobile', (tester) async {
    final state = _state();
    for (final width in [1440.0, 1000.0, 700.0]) {
      await _pump(tester, DashboardView(state: state), size: Size(width, 900));
      expect(tester.takeException(), isNull, reason: 'width $width');
    }
  });

  testWidgets('nodes lay out wide and narrow', (tester) async {
    final state = _state();
    for (final width in [1440.0, 1000.0, 700.0]) {
      await _pump(tester, NodesView(state: state), size: Size(width, 900));
      expect(tester.takeException(), isNull, reason: 'width $width');
      expect(find.text('香港家宽专线'), findsOneWidget);
    }
  });

  testWidgets('history renders a timeline with events', (tester) async {
    final state = _state();
    for (final width in [960.0, 700.0]) {
      await _pump(tester, HistoryView(state: state), size: Size(width, 900));
      expect(tester.takeException(), isNull, reason: 'width $width');
      expect(find.text('切换'), findsNWidgets(2));
      expect(find.textContaining('香港家宽专线'), findsOneWidget);
    }
  });

  testWidgets('settings lays out at its max width and narrow', (tester) async {
    final state = _state();
    for (final width in [960.0, 700.0]) {
      await _pump(tester, SettingsView(state: state), size: Size(width, 900));
      expect(tester.takeException(), isNull, reason: 'width $width');
      expect(find.text('外观'), findsOneWidget);
      expect(find.text('连接设置'), findsOneWidget);
    }
  });
}
