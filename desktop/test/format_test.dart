import 'package:flutter_test/flutter_test.dart';
import 'package:nodepulse/format.dart';

void main() {
  test('percent uses success and total', () {
    expect(percent({'success': 8, 'total': 10}), '80%');
    expect(percent({'success': 0, 'total': 0}), '—');
  });

  test('score keeps three decimals', () {
    expect(scoreText(0.91234), '0.912');
  });

  test('cleanName removes flag regional indicators', () {
    expect(cleanName('🇺🇸美国纽约'), '美国纽约');
  });

  test('baseName handles empty paths', () {
    expect(baseName('/tmp/sub.yaml'), 'sub.yaml');
    expect(baseName(''), '未选择');
  });
}
