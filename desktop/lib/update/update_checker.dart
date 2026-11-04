import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

class ReleaseInfo {
  const ReleaseInfo({
    required this.version,
    required this.url,
    required this.notes,
    this.publishedAt,
  });

  final String version;
  final String url;
  final String notes;
  final DateTime? publishedAt;

  factory ReleaseInfo.fromJson(Map<String, dynamic> json) {
    final tag = (json['tag_name'] ?? '').toString();
    return ReleaseInfo(
      version: UpdateChecker.normalizeVersion(tag),
      url: (json['html_url'] ?? '').toString(),
      notes: (json['body'] ?? '').toString().trim(),
      publishedAt: DateTime.tryParse((json['published_at'] ?? '').toString()),
    );
  }
}

abstract final class UpdateChecker {
  static const repository = 'HanphoneJan/nodepulse';

  static Future<ReleaseInfo?> latest() async {
    final response = await http
        .get(
          Uri.parse('https://api.github.com/repos/$repository/releases/latest'),
          headers: const {
            'Accept': 'application/vnd.github+json',
            'User-Agent': 'NodePulse',
          },
        )
        .timeout(const Duration(seconds: 10));
    if (response.statusCode != 200) return null;
    final decoded = jsonDecode(utf8.decode(response.bodyBytes));
    if (decoded is! Map<String, dynamic>) return null;
    return ReleaseInfo.fromJson(decoded);
  }

  static String normalizeVersion(String tag) =>
      tag.trim().replaceFirst(RegExp(r'^[vV]'), '');

  static bool isNewer(String candidate, String current) {
    final next = _parts(candidate);
    final now = _parts(current);
    if (next.isEmpty || now.isEmpty) return false;
    for (var i = 0; i < 3; i++) {
      final left = i < next.length ? next[i] : 0;
      final right = i < now.length ? now[i] : 0;
      if (left != right) return left > right;
    }
    return false;
  }

  static List<int> _parts(String version) {
    final core = version.split('+').first.split('-').first;
    final parts = <int>[];
    for (final piece in core.split('.')) {
      final number = int.tryParse(piece);
      if (number == null) return const [];
      parts.add(number);
    }
    return parts;
  }
}

Future<bool> openExternalUrl(String url) async {
  final uri = Uri.tryParse(url);
  if (uri == null) return false;
  try {
    return await launchUrl(uri, mode: LaunchMode.externalApplication);
  } catch (_) {
    return false;
  }
}
