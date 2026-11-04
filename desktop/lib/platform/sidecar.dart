import 'dart:async';
import 'dart:io';
import 'dart:math';

import 'package:http/http.dart' as http;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

import 'android_platform.dart';

class Sidecar {
  Sidecar._(this.baseUrl, this.token, this._process);

  final String baseUrl;
  final String token;
  final Process _process;

  static Future<Sidecar> start() async {
    final port = await _freePort();
    final token = _randomToken();
    final executable = await _findExecutable();
    final support = await getApplicationSupportDirectory();
    await support.create(recursive: true);
    final process = await Process.start(
      executable,
      [
        'api',
        '--addr',
        '127.0.0.1:$port',
        '--token',
        token,
        '--results',
        p.join(support.path, 'results.json'),
        '--history',
        p.join(support.path, 'history.jsonl'),
      ],
      workingDirectory: Platform.isAndroid
          ? support.path
          : Directory.current.path,
    );
    process.stdout.drain<void>();
    process.stderr.drain<void>();

    final baseUrl = 'http://127.0.0.1:$port';
    final deadline = DateTime.now().add(const Duration(seconds: 15));
    while (DateTime.now().isBefore(deadline)) {
      if (await process.exitCode.timeout(
            const Duration(milliseconds: 50),
            onTimeout: () => -1,
          ) !=
          -1) {
        throw StateError('NodePulse 后端已退出，无法启动');
      }
      try {
        final response = await http
            .get(
              Uri.parse('$baseUrl/api/version'),
              headers: {'Authorization': 'Bearer $token'},
            )
            .timeout(const Duration(milliseconds: 700));
        if (response.statusCode == 200) {
          return Sidecar._(baseUrl, token, process);
        }
      } catch (_) {
        await Future<void>.delayed(const Duration(milliseconds: 150));
      }
    }
    process.kill(ProcessSignal.sigkill);
    throw StateError('NodePulse 后端启动超时');
  }

  Future<void> stop() async {
    _process.kill();
    try {
      await _process.exitCode.timeout(const Duration(seconds: 2));
    } catch (_) {
      _process.kill(ProcessSignal.sigkill);
    }
  }

  static Future<int> _freePort() async {
    final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final port = socket.port;
    await socket.close();
    return port;
  }

  static String _randomToken() {
    final random = Random.secure();
    final bytes = List<int>.generate(24, (_) => random.nextInt(256));
    return bytes.map((value) => value.toRadixString(16).padLeft(2, '0')).join();
  }

  static Future<String> _findExecutable() async {
    final env = Platform.environment['NODEPULSE_BIN'];
    if (env != null && env.isNotEmpty && File(env).existsSync()) {
      return env;
    }
    if (Platform.isAndroid) {
      final dir = await androidNativeLibraryDir();
      if (dir != null) {
        final candidate = p.join(dir, 'libnodepulse_core.so');
        if (File(candidate).existsSync()) return candidate;
      }
      throw StateError('Android 后端缺失：libnodepulse_core.so 未打包。');
    }
    final ext = Platform.isWindows ? '.exe' : '';
    final exeDir = p.dirname(Platform.resolvedExecutable);
    final candidates = <String>[
      p.join(exeDir, 'nodepulse-core$ext'),
      p.join(exeDir, '..', 'Resources', 'nodepulse-core$ext'),
      p.join(Directory.current.path, '..', 'bin', 'nodepulse$ext'),
      p.join(Directory.current.path, 'bin', 'nodepulse$ext'),
    ];
    for (final candidate in candidates) {
      if (File(candidate).existsSync()) return candidate;
    }
    throw StateError(
      '找不到 NodePulse 后端（nodepulse-core）。请先运行 make build，或设置 NODEPULSE_BIN。',
    );
  }
}
