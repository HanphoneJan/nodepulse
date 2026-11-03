import 'package:path/path.dart' as p;

import 'api/api_client.dart';

String percent(dynamic row) {
  if (row is! Map) return '—';
  final success = (row['success'] as num?)?.toInt() ?? 0;
  final total = (row['total'] as num?)?.toInt() ?? 0;
  if (total <= 0) return '—';
  return '${(success * 100 / total).round()}%';
}

String ms(dynamic value) {
  final number = (value as num?)?.toInt() ?? 0;
  return number <= 0 ? '—' : '${number}ms';
}

String scoreText(dynamic value) {
  final number = (value as num?)?.toDouble() ?? 0;
  return number.toStringAsFixed(3);
}

String cleanName(String? value) {
  if (value == null || value.isEmpty) return '';
  return value
      .replaceAll(RegExp('[\u{1F1E6}-\u{1F1FF}]', unicode: true), '')
      .trim();
}

String baseName(String? path) {
  if (path == null || path.isEmpty) return '未选择';
  return p.basename(path);
}

String timeAgo(String? raw) {
  if (raw == null || raw.isEmpty) return '';
  final parsed = DateTime.tryParse(raw);
  if (parsed == null) return raw;
  final diff = DateTime.now().difference(parsed);
  if (diff.inSeconds < 60) return '刚刚';
  if (diff.inMinutes < 60) return '${diff.inMinutes} 分钟前';
  if (diff.inHours < 24) return '${diff.inHours} 小时前';
  return '${diff.inDays} 天前';
}

String clockText(String? raw) {
  if (raw == null || raw.isEmpty) return '';
  final parsed = DateTime.tryParse(raw);
  if (parsed == null) return raw;
  final local = parsed.toLocal();
  String two(int v) => v.toString().padLeft(2, '0');
  return '${two(local.month)}-${two(local.day)} ${two(local.hour)}:${two(local.minute)}';
}

String kindLabel(String? kind) => switch (kind) {
  'cycle' => '检查',
  'switch' => '切换',
  'rollback' => '回滚',
  'error' => '错误',
  _ => kind ?? '事件',
};

String reasonLabel(String? code) => switch (code) {
  'better-score' => '候选评分更高',
  'below-min-success' => '成功率不足',
  'current-missing' => '当前节点缺失',
  'controller-error' => '控制口错误',
  'group-error' => '分组错误',
  'no-candidates' => '没有候选',
  'download-unreachable' => '目标不可达',
  _ => code ?? '',
};

String errorText(Object error) =>
    error is ApiException ? error.message : error.toString();
