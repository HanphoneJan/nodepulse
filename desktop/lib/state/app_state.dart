import 'dart:async';
import 'dart:io';
import 'dart:math';

import 'package:launch_at_startup/launch_at_startup.dart';
import 'package:material_ui/material_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/api_client.dart';
import '../platform/sidecar.dart';

const _seedKey = 'nodepulse.seed.v3';
const _modeKey = 'nodepulse.themeMode.v3';
const _defaultSeed = '#007aff';

class AppState extends ChangeNotifier {
  final GlobalKey<ScaffoldMessengerState> messengerKey =
      GlobalKey<ScaffoldMessengerState>();

  Sidecar? _sidecar;
  ApiClient? api;
  Timer? _saveTimer;
  Timer? _pollTimer;
  Timer? _historyTimer;
  bool _saving = false;

  bool ready = false;
  String? error;
  String version = '';
  static const _startView = String.fromEnvironment(
    'START_VIEW',
    defaultValue: 'dashboard',
  );
  String view = _startView;

  Map<String, dynamic> settings = <String, dynamic>{};
  Map<String, dynamic> status = <String, dynamic>{
    'connected': false,
    'current': '',
    'group': '',
    'members': 0,
  };
  List<dynamic> groups = const [];
  List<dynamic> controllers = const [];
  List<dynamic> clients = const [];
  List<dynamic> subscriptions = const [];
  List<dynamic> results = const [];
  List<dynamic> history = const [];
  Map<String, dynamic>? best;
  Map<String, dynamic>? downloadResult;
  String? selected;
  String? lastTestAt;

  bool busy = false;
  String busyKind = '';
  bool daemon = false;
  bool autostart = false;
  String? _lastEventKey;
  bool detecting = false;
  Map<String, dynamic> progress = const {};

  String seed = _defaultSeed;
  ThemeMode themeMode = ThemeMode.system;

  String downloadUrl = '';
  String search = '';
  bool exportMinimal = false;
  Map<String, dynamic>? exportImpact;

  bool get connected => status['connected'] == true;
  String get currentNode => (status['current'] ?? '').toString();
  String get groupName => (status['group'] ?? '').toString();
  int get memberCount => (status['members'] as num?)?.toInt() ?? 0;

  List<dynamic> get profiles =>
      (settings['profiles'] as List?)?.cast<dynamic>() ?? const [];

  Map<String, dynamic> get activeProfile {
    final id = settings['active_profile'];
    for (final profile in profiles) {
      if (profile is Map && profile['id'] == id) {
        return Map<String, dynamic>.from(profile);
      }
    }
    if (profiles.isNotEmpty && profiles.first is Map) {
      return Map<String, dynamic>.from(profiles.first as Map);
    }
    return const {};
  }

  Future<void> init() async {
    final prefs = await SharedPreferences.getInstance();
    seed = prefs.getString(_seedKey) ?? _defaultSeed;
    themeMode = switch (prefs.getString(_modeKey)) {
      'light' => ThemeMode.light,
      'dark' => ThemeMode.dark,
      _ => ThemeMode.system,
    };
    const debugTheme = String.fromEnvironment('START_THEME');
    if (debugTheme == 'dark') themeMode = ThemeMode.dark;
    if (debugTheme == 'light') themeMode = ThemeMode.light;
    try {
      try {
        launchAtStartup.setup(
          appName: 'NodePulse',
          appPath: Platform.resolvedExecutable,
          packageName: 'com.nodepulse.nodepulse',
        );
        autostart = await launchAtStartup.isEnabled();
      } catch (_) {}
      _sidecar = await Sidecar.start();
      api = ApiClient(baseUrl: _sidecar!.baseUrl, token: _sidecar!.token);
      version = await api!.version();
      final bootstrap = await api!.bootstrap();
      settings = Map<String, dynamic>.from(bootstrap['settings'] as Map? ?? {});
      controllers = (bootstrap['controllers'] as List?) ?? const [];
      clients = (bootstrap['clients'] as List?) ?? const [];
      subscriptions = (bootstrap['subscriptions'] as List?) ?? const [];
      await _loadResults();
      history = await api!.history();
      if (history.isNotEmpty && history.first is Map) {
        _lastEventKey = _eventKey(history.first as Map);
      }
      daemon = await api!.daemonStatus();
      await refreshStatus(notifyError: false);
      ready = true;
      _historyTimer = Timer.periodic(const Duration(seconds: 6), (_) async {
        if (!ready || _sidecar == null) return;
        try {
          final next = await api!.history();
          if (next.length != history.length) {
            history = next;
            if (next.isNotEmpty && next.first is Map) {
              final latest = next.first as Map;
              final key = _eventKey(latest);
              if (_lastEventKey != null &&
                  key != _lastEventKey &&
                  settings['notify_on_switch'] != false) {
                _notifyEvent(latest);
              }
              _lastEventKey = key;
            }
            notifyListeners();
          }
        } catch (_) {}
      });
    } catch (e) {
      error = e.toString();
      ready = true;
    }
    notifyListeners();
  }

  Future<void> _loadResults() async {
    try {
      final file = await api!.resultsFile();
      results = (file['results'] as List?) ?? const [];
      lastTestAt = file['generated_at']?.toString();
      best = results.isNotEmpty ? _map(results.first) : null;
      selected = best?['name']?.toString();
    } catch (_) {
      results = const [];
    }
  }

  Future<void> disposeSidecar() async {
    _saveTimer?.cancel();
    _pollTimer?.cancel();
    _historyTimer?.cancel();
    await _sidecar?.stop();
    api?.close();
  }

  void setView(String next) {
    view = next;
    notifyListeners();
  }

  void setSearch(String value) {
    search = value;
    notifyListeners();
  }

  void selectNode(String? name) {
    selected = name;
    notifyListeners();
  }

  void setDownloadUrl(String value) => downloadUrl = value;

  Future<void> setSeed(String value) async {
    seed = value.toLowerCase();
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_seedKey, seed);
    notifyListeners();
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    themeMode = mode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_modeKey, switch (mode) {
      ThemeMode.light => 'light',
      ThemeMode.dark => 'dark',
      _ => 'system',
    });
    notifyListeners();
  }

  void updateSetting(String key, Object? value) {
    settings = {...settings, key: value};
    notifyListeners();
    _scheduleSave();
  }

  void _scheduleSave() {
    _saveTimer?.cancel();
    _saveTimer = Timer(const Duration(milliseconds: 450), saveSettings);
  }

  Future<void> saveSettings({bool quiet = false}) async {
    if (api == null || _saving) return;
    _saving = true;
    try {
      await api!.saveSettings(settings);
      if (!quiet) _toast('设置已保存');
    } catch (e) {
      _toast(e.toString(), isError: true);
    } finally {
      _saving = false;
    }
  }

  Future<void> activateProfile(String id) async {
    final profile = profiles.cast<Map>().firstWhere(
      (item) => item['id'] == id,
      orElse: () => <String, dynamic>{},
    );
    if (profile.isEmpty) return;
    const fields = [
      'api',
      'secret',
      'group',
      'config_path',
      'mihomo_path',
      'url',
      'rounds',
      'concurrency',
      'timeout_ms',
    ];
    final next = Map<String, dynamic>.from(settings);
    for (final field in fields) {
      next[field] = profile[field] ?? next[field];
    }
    next['active_profile'] = id;
    settings = next;
    await api!.saveSettings(settings);
    await refreshStatus();
    notifyListeners();
  }

  void renameProfile(String name) {
    final id = settings['active_profile'];
    final list = profiles.map((item) {
      if (item is Map && item['id'] == id) {
        return {...Map<String, dynamic>.from(item), 'name': name};
      }
      return item;
    }).toList();
    updateSetting('profiles', list);
  }

  Future<void> newProfile() async {
    final profile = <String, dynamic>{
      'id': 'env-${DateTime.now().millisecondsSinceEpoch.toRadixString(36)}',
      'name': '环境 ${profiles.length + 1}',
      'api': settings['api'] ?? '',
      'secret': settings['secret'] ?? '',
      'group': settings['group'] ?? '',
      'config_path': settings['config_path'] ?? '',
      'mihomo_path': settings['mihomo_path'] ?? '',
      'url': settings['url'] ?? '',
      'rounds': settings['rounds'] ?? 5,
      'concurrency': settings['concurrency'] ?? 8,
      'timeout_ms': settings['timeout_ms'] ?? 5000,
    };
    updateSetting('profiles', [...profiles, profile]);
    updateSetting('active_profile', profile['id']);
    await saveSettings(quiet: true);
  }

  Future<void> deleteProfile() async {
    if (profiles.length <= 1) {
      _toast('至少保留一个环境', isError: true);
      return;
    }
    final remaining = profiles
        .where(
          (item) => item is! Map || item['id'] != settings['active_profile'],
        )
        .toList();
    final next = Map<String, dynamic>.from(remaining.first as Map);
    final nextSettings = Map<String, dynamic>.from(settings);
    for (final field in [
      'api',
      'secret',
      'group',
      'config_path',
      'mihomo_path',
      'url',
      'rounds',
      'concurrency',
      'timeout_ms',
    ]) {
      nextSettings[field] = next[field] ?? nextSettings[field];
    }
    nextSettings['profiles'] = remaining;
    nextSettings['active_profile'] = next['id'];
    settings = nextSettings;
    await api!.saveSettings(settings);
    await refreshStatus();
    notifyListeners();
  }

  Future<void> refreshStatus({bool notifyError = true}) async {
    final client = api;
    if (client == null) return;
    if ((settings['api'] ?? '').toString().isEmpty) {
      status = {
        'connected': false,
        'current': '',
        'group': settings['group'] ?? '',
        'members': 0,
      };
      groups = const [];
      notifyListeners();
      return;
    }
    try {
      final list = await client.status(settings);
      groups = list;
      Map<String, dynamic>? group;
      for (final item in list) {
        if (item is Map && item['name'] == settings['group']) {
          group = Map<String, dynamic>.from(item);
          break;
        }
      }
      if (group == null) {
        for (final item in list.whereType<Map>()) {
          if ('${item['type']}'.toLowerCase() == 'selector') {
            group = Map<String, dynamic>.from(item);
            break;
          }
        }
      }
      if (group == null && list.isNotEmpty && list.first is Map) {
        group = Map<String, dynamic>.from(list.first as Map);
      }
      status = {
        'connected': true,
        'current': group?['now'] ?? '',
        'group': group?['name'] ?? settings['group'] ?? '',
        'members': group?['members'] ?? 0,
      };
      if (group != null && group['name'] != settings['group']) {
        settings = {...settings, 'group': group['name']};
      }
    } catch (e) {
      status = {
        'connected': false,
        'current': '',
        'group': settings['group'] ?? '',
        'members': 0,
        'error': e.toString(),
      };
      groups = const [];
      if (notifyError) _toast(e.toString(), isError: true);
    }
    notifyListeners();
  }

  Future<void> detect() async {
    if (api == null || detecting) return;
    detecting = true;
    notifyListeners();
    try {
      final bootstrap = await api!.bootstrap();
      controllers = (bootstrap['controllers'] as List?) ?? const [];
      clients = (bootstrap['clients'] as List?) ?? const [];
      subscriptions = (bootstrap['subscriptions'] as List?) ?? const [];
      final found = Map<String, dynamic>.from(
        bootstrap['settings'] as Map? ?? {},
      );
      final patch = <String, dynamic>{};
      if ((settings['api'] ?? '').toString().isEmpty && found['api'] != null) {
        patch['api'] = found['api'];
      }
      if ((settings['secret'] ?? '').toString().isEmpty &&
          found['secret'] != null) {
        patch['secret'] = found['secret'];
      }
      if (patch.isNotEmpty) {
        settings = {...settings, ...patch};
      }
      await refreshStatus(notifyError: false);
      await saveSettings(quiet: true);
      final running = clients
          .whereType<Map>()
          .where((c) => c['running'] == true)
          .toList();
      _toast(
        running.isEmpty
            ? '没有检测到运行中的客户端'
            : '检测到 ${running.map((c) => c['name']).join('、')}',
      );
    } catch (e) {
      _toast(e.toString(), isError: true);
    } finally {
      detecting = false;
      notifyListeners();
    }
  }

  Future<void> _run(
    String kind,
    Future<void> Function(ApiClient client) action,
  ) async {
    if (busy || api == null) return;
    busy = true;
    busyKind = kind;
    progress = const {};
    notifyListeners();
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(milliseconds: 500), (_) async {
      try {
        final next = await api!.progress();
        progress = next;
        notifyListeners();
      } catch (_) {}
    });
    try {
      await action(api!);
      await _loadResults();
      await refreshStatus(notifyError: false);
      try {
        history = await api!.history();
      } catch (_) {}
    } catch (e) {
      if (e.toString().contains('取消')) {
        _toast('已取消');
      } else {
        _toast(e.toString(), isError: true);
      }
    } finally {
      _pollTimer?.cancel();
      progress = const {};
      busy = false;
      busyKind = '';
      notifyListeners();
    }
  }

  Future<void> oneClick() => _run('oneclick', (client) async {
    final result = await client.oneClick(settings);
    results = (result['results'] as List?) ?? const [];
    best = result['best'] is Map ? _map(result['best']) : null;
    selected = best?['name']?.toString();
    lastTestAt = DateTime.now().toIso8601String();
    _toast((result['message'] ?? '优化完成').toString());
  });

  Future<void> testOnly() => _run('test', (client) async {
    final list = await client.test(settings);
    results = list;
    best = list.isNotEmpty ? _map(list.first) : null;
    selected = best?['name']?.toString();
    lastTestAt = DateTime.now().toIso8601String();
    _toast('测速完成：${list.length} 个节点');
  });

  Future<void> testAll() => _run('all', (client) async {
    final result = await client.testAll(settings);
    results = (result['results'] as List?) ?? const [];
    best = result['best'] is Map ? _map(result['best']) : null;
    selected = null;
    lastTestAt = DateTime.now().toIso8601String();
    _toast((result['message'] ?? '联合测速完成').toString());
  });

  Future<void> downloadMode() => _run('download', (client) async {
    if (!downloadUrl.startsWith('http://') &&
        !downloadUrl.startsWith('https://')) {
      throw ApiException('请填写 http/https 文件 URL');
    }
    final result = await client.downloadMode(settings, downloadUrl);
    downloadResult = result;
    results = (result['results'] as List?) ?? const [];
    best = result['best'] is Map ? _map(result['best']) : null;
    selected = best?['name']?.toString();
    lastTestAt = DateTime.now().toIso8601String();
    _toast((result['message'] ?? '下载模式完成').toString());
  });

  Future<void> cancel() async {
    try {
      await api?.cancel();
    } catch (e) {
      _toast(e.toString(), isError: true);
    }
  }

  Future<void> switchSelected() async {
    final name = selected;
    if (name == null || name.isEmpty) {
      _toast('请先选择节点', isError: true);
      return;
    }
    if (results.any((row) => row is Map && row['source'] != null)) {
      _toast('联合测速只给推荐，不能直接切换', isError: true);
      return;
    }
    await _run('switch', (client) async {
      await client.switchNode(settings, name);
      _toast('已切换到 $name');
    });
  }

  Future<void> toggleAutostart(bool enabled) async {
    try {
      if (enabled) {
        await launchAtStartup.enable();
      } else {
        await launchAtStartup.disable();
      }
      autostart = enabled;
      notifyListeners();
    } catch (e) {
      _toast(e.toString(), isError: true);
    }
  }

  Future<void> toggleDaemon(bool enabled) async {
    if (busy || api == null) return;
    if (enabled && (settings['api'] ?? '').toString().isEmpty) {
      _toast('自动优化需要先连接客户端', isError: true);
      return;
    }
    try {
      if (enabled) {
        await api!.daemonStart(settings);
      } else {
        await api!.daemonStop();
      }
      daemon = enabled;
      settings = {...settings, 'auto_optimize': enabled};
      await api!.saveSettings(settings);
      notifyListeners();
    } catch (e) {
      _toast(e.toString(), isError: true);
      notifyListeners();
    }
  }

  Future<void> importSubscription(String url) async {
    if (url.trim().isEmpty) {
      _toast('请粘贴订阅 URL', isError: true);
      return;
    }
    await _run('import', (client) async {
      final path = await client.importSubscription(url.trim());
      settings = {...settings, 'config_path': path};
      await client.saveSettings(settings);
      final bootstrap = await client.bootstrap();
      subscriptions = (bootstrap['subscriptions'] as List?) ?? subscriptions;
      _toast('订阅已导入');
    });
  }

  Future<void> refreshHealth() async {
    if (api == null) return;
    try {
      final report = await api!.health(settings);
      settings = {...settings, '_health': report};
      notifyListeners();
    } catch (e) {
      _toast(e.toString(), isError: true);
    }
  }

  Future<void> exportPreview(bool minimal) async {
    exportMinimal = minimal;
    try {
      exportImpact = await api!.exportPreview(minimal);
      notifyListeners();
    } catch (e) {
      _toast(e.toString(), isError: true);
    }
  }

  Future<void> confirmExport() async {
    try {
      final message = await api!.exportStable(exportMinimal);
      exportImpact = null;
      _toast(message);
      notifyListeners();
    } catch (e) {
      _toast(e.toString(), isError: true);
    }
  }

  void cancelExport() {
    exportImpact = null;
    notifyListeners();
  }

  Future<String?> backupSettings() async {
    try {
      return await api!.backupSettings();
    } catch (e) {
      _toast(e.toString(), isError: true);
      return null;
    }
  }

  Future<void> restoreSettings(String raw) async {
    try {
      final message = await api!.restoreSettings(raw);
      final fresh = await api!.settings();
      settings = Map<String, dynamic>.from(fresh);
      _toast(message);
      notifyListeners();
    } catch (e) {
      _toast(e.toString(), isError: true);
    }
  }

  String _eventKey(Map event) =>
      '${event['time']}|${event['kind']}|${event['to']}';

  void _notifyEvent(Map event) {
    final kind = event['kind']?.toString() ?? '';
    if (!['switch', 'rollback', 'error'].contains(kind)) return;
    final title = kind == 'switch'
        ? 'NodePulse 已切换节点'
        : kind == 'rollback'
        ? 'NodePulse 已回滚'
        : 'NodePulse 自动优化失败';
    final from = _cleanEventName(event['from']?.toString());
    final to = _cleanEventName(event['to']?.toString());
    final body = kind == 'switch' && to.isNotEmpty
        ? '$from → $to'
        : (event['reason']?.toString().isNotEmpty == true
              ? event['reason'].toString()
              : from);
    _sendNotification(title, body);
  }

  String _cleanEventName(String? value) => (value ?? '')
      .replaceAll(RegExp('[\u{1F1E6}-\u{1F1FF}]', unicode: true), '')
      .trim();

  Future<void> _sendNotification(String title, String body) async {
    try {
      if (Platform.isLinux) {
        await Process.run('gdbus', [
          'call',
          '--session',
          '--dest',
          'org.freedesktop.Notifications',
          '--object-path',
          '/org/freedesktop/Notifications',
          '--method',
          'org.freedesktop.Notifications.Notify',
          'NodePulse',
          '0',
          '',
          title,
          body,
          '[]',
          '{}',
          '-1',
        ]);
      } else if (Platform.isMacOS) {
        final safeTitle = title.replaceAll('"', '\\"');
        final safeBody = body.replaceAll('"', '\\"');
        await Process.run('osascript', [
          '-e',
          'display notification "$safeBody" with title "$safeTitle"',
        ]);
      } else if (Platform.isWindows) {
        final safeTitle = title.replaceAll("'", "''");
        final safeBody = body.replaceAll("'", "''");
        final script =
            r'''
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime] | Out-Null
$t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$n = $t.GetElementsByTagName("text")
$n.Item(0).AppendChild($t.CreateTextNode('__TITLE__')) | Out-Null
$n.Item(1).AppendChild($t.CreateTextNode('__BODY__')) | Out-Null
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("NodePulse").Show([Windows.UI.Notifications.ToastNotification]::new($t))
'''
                .replaceAll('__TITLE__', safeTitle)
                .replaceAll('__BODY__', safeBody);
        await Process.run('powershell', ['-NoProfile', '-Command', script]);
      }
    } catch (_) {}
  }

  void _toast(String message, {bool isError = false}) {
    final messenger = messengerKey.currentState;
    if (messenger == null) return;
    messenger.showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: isError
            ? Theme.of(messenger.context).colorScheme.errorContainer
            : null,
        showCloseIcon: true,
      ),
    );
  }

  Map<String, dynamic> _map(dynamic value) =>
      value is Map ? Map<String, dynamic>.from(value) : <String, dynamic>{};

  String get busyLabel => switch (busyKind) {
    'oneclick' => '优化中',
    'test' => '测速中',
    'all' => '联合测速中',
    'download' => '下载探测中',
    _ => '处理中',
  };

  String get seedInk {
    final hex = seed.replaceFirst('#', '');
    final value = int.tryParse(hex, radix: 16) ?? 0;
    final r = (value >> 16) & 0xff;
    final g = (value >> 8) & 0xff;
    final b = value & 0xff;
    final luminance = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
    return luminance > 0.55 ? '#181c23' : '#ffffff';
  }

  static String randomSeed() {
    final random = Random();
    final value = 0xff000000 | random.nextInt(0xffffff);
    return '#${value.toRadixString(16).substring(2)}';
  }
}
