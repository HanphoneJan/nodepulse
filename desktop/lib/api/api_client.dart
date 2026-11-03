import 'dart:convert';

import 'package:http/http.dart' as http;

class ApiException implements Exception {
  ApiException(this.message, {this.statusCode});

  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

class ApiClient {
  ApiClient({required this.baseUrl, required this.token});

  final String baseUrl;
  final String token;
  final http.Client _client = http.Client();

  Uri _uri(String path, [Map<String, String>? query]) =>
      Uri.parse('$baseUrl$path').replace(queryParameters: query);

  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (token.isNotEmpty) 'Authorization': 'Bearer $token',
  };

  Future<dynamic> _decode(http.Response response) async {
    final text = utf8.decode(response.bodyBytes);
    dynamic data;
    if (text.isNotEmpty) {
      try {
        data = jsonDecode(text);
      } catch (_) {
        data = null;
      }
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final message = data is Map && data['error'] != null
          ? data['error'].toString()
          : (text.isEmpty ? response.reasonPhrase : text);
      throw ApiException(
        message ?? '请求失败 (${response.statusCode})',
        statusCode: response.statusCode,
      );
    }
    return data;
  }

  Future<dynamic> get(String path, [Map<String, String>? query]) async {
    final response = await _client.get(_uri(path, query), headers: _headers);
    return _decode(response);
  }

  Future<dynamic> post(String path, [Object? body]) async {
    final response = await _client.post(
      _uri(path),
      headers: _headers,
      body: jsonEncode(body ?? const <String, dynamic>{}),
    );
    return _decode(response);
  }

  Future<dynamic> getText(String path) async {
    final response = await _client.get(_uri(path), headers: _headers);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(
        '请求失败 (${response.statusCode})',
        statusCode: response.statusCode,
      );
    }
    return utf8.decode(response.bodyBytes);
  }

  Future<String> version() async {
    final data = await get('/api/version') as Map<String, dynamic>;
    return (data['version'] ?? '').toString();
  }

  Future<Map<String, dynamic>> bootstrap() async =>
      Map<String, dynamic>.from(await get('/api/bootstrap') as Map);

  Future<Map<String, dynamic>> settings() async =>
      Map<String, dynamic>.from(await get('/api/settings') as Map);

  Future<void> saveSettings(Map<String, dynamic> value) =>
      post('/api/settings', value);

  Future<List<dynamic>> status(Map<String, dynamic> settings) async {
    final data = await post('/api/status', {
      'api': settings['api'] ?? '',
      'secret': settings['secret'] ?? '',
    });
    return data is List ? data : const [];
  }

  Future<Map<String, dynamic>> resultsFile() async {
    final data = await get('/api/results');
    if (data is Map) return Map<String, dynamic>.from(data);
    return const {'results': <dynamic>[]};
  }

  Future<List<dynamic>> history() async {
    final data = await get('/api/history');
    return data is List ? data : const [];
  }

  Future<Map<String, dynamic>> progress() async {
    final data = await get('/api/progress');
    return data is Map ? Map<String, dynamic>.from(data) : const {};
  }

  Future<void> cancel() => post('/api/cancel', const {});

  Future<void> clearResults() => post('/api/results/clear', const {});

  Map<String, dynamic> _settingsPayload(Map<String, dynamic> settings) => {
    'api': settings['api'] ?? '',
    'secret': settings['secret'] ?? '',
    'group': settings['group'] ?? '',
    'config_path': settings['config_path'] ?? '',
    'mihomo_path': settings['mihomo_path'] ?? '',
    'url': settings['url'] ?? '',
    'rounds': settings['rounds'] ?? 5,
    'concurrency': settings['concurrency'] ?? 8,
    'timeout_ms': settings['timeout_ms'] ?? 5000,
    'profiles': settings['profiles'] ?? const [],
    'active_profile': settings['active_profile'] ?? '',
    'auto_optimize': settings['auto_optimize'] ?? false,
    'daemon_interval': settings['daemon_interval'] ?? 300,
    'auto_margin': settings['auto_margin'] ?? 0.2,
    'auto_confirm': settings['auto_confirm'] ?? 2,
    'auto_min_success': settings['auto_min_success'] ?? 0.8,
    'close_to_tray': settings['close_to_tray'] ?? true,
    'notify_on_switch': settings['notify_on_switch'] ?? true,
  };

  Future<Map<String, dynamic>> oneClick(Map<String, dynamic> settings) async {
    final data = await post('/api/oneclick', {
      ..._settingsPayload(settings),
      'switch': true,
    });
    return Map<String, dynamic>.from(data as Map);
  }

  Future<List<dynamic>> test(Map<String, dynamic> settings) async {
    final data = await post('/api/test', _settingsPayload(settings));
    return data is List ? data : const [];
  }

  Future<Map<String, dynamic>> testAll(Map<String, dynamic> settings) async {
    final data = await post('/api/test/all', {
      'api': settings['api'] ?? '',
      'secret': settings['secret'] ?? '',
      'mihomo_path': settings['mihomo_path'] ?? '',
      'url': settings['url'] ?? '',
      'rounds': settings['rounds'] ?? 5,
      'concurrency': settings['concurrency'] ?? 8,
      'timeout_ms': settings['timeout_ms'] ?? 5000,
    });
    return Map<String, dynamic>.from(data as Map);
  }

  Future<Map<String, dynamic>> downloadMode(
    Map<String, dynamic> settings,
    String target,
  ) async {
    final data = await post('/api/download-mode', {
      ..._settingsPayload(settings),
      'target_url': target,
      'probe_bytes': 3 << 20,
      'probe_seconds': 5,
      'probe_top_k': 5,
    });
    return Map<String, dynamic>.from(data as Map);
  }

  Future<void> switchNode(Map<String, dynamic> settings, String node) =>
      post('/api/switch', {
        'api': settings['api'] ?? '',
        'secret': settings['secret'] ?? '',
        'group': settings['group'] ?? '',
        'node': node,
      });

  Future<String> importSubscription(String url) async {
    final data = await post('/api/import', {'url': url});
    return ((data as Map)['path'] ?? '').toString();
  }

  Future<Map<String, dynamic>> health(Map<String, dynamic> settings) async {
    final data = await post('/api/health', {
      'api': settings['api'] ?? '',
      'secret': settings['secret'] ?? '',
      'group': settings['group'] ?? '',
      'config_path': settings['config_path'] ?? '',
      'mihomo_path': settings['mihomo_path'] ?? '',
      'url': settings['url'] ?? '',
    });
    return Map<String, dynamic>.from(data as Map);
  }

  Future<void> daemonStart(Map<String, dynamic> settings) =>
      post('/api/daemon/start', {
        'api': settings['api'] ?? '',
        'secret': settings['secret'] ?? '',
        'group': settings['group'] ?? '',
        'interval_sec': settings['daemon_interval'] ?? 300,
        'rounds': settings['rounds'] ?? 5,
        'margin': settings['auto_margin'] ?? 0.2,
        'confirm': settings['auto_confirm'] ?? 2,
        'min_success': settings['auto_min_success'] ?? 0.8,
      });

  Future<void> daemonStop() => post('/api/daemon/stop', const {});

  Future<bool> daemonStatus() async {
    final data = await get('/api/daemon/status') as Map;
    return data['running'] == true;
  }

  Future<Map<String, dynamic>> exportPreview(bool minimal) async {
    final data = await post('/api/export/preview', {
      'min_success': 0.8,
      'minimal': minimal,
    });
    return Map<String, dynamic>.from(data as Map);
  }

  Future<String> exportStable(bool minimal) async {
    final data = await post('/api/export', {
      'out': 'stable.yaml',
      'min_success': 0.8,
      'minimal': minimal,
    });
    return ((data as Map)['message'] ?? '已导出 stable.yaml').toString();
  }

  Future<String> backupSettings() async =>
      (await getText('/api/settings/backup')).toString();

  Future<String> restoreSettings(String raw) async {
    final data = await post('/api/settings/restore', jsonDecode(raw));
    return ((data as Map)['message'] ?? '设置已恢复').toString();
  }

  void close() => _client.close();
}
