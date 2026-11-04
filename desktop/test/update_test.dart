import 'package:flutter_test/flutter_test.dart';
import 'package:nodepulse/update/update_checker.dart';

void main() {
  test('normalizes tags', () {
    expect(UpdateChecker.normalizeVersion('v0.3.1'), '0.3.1');
    expect(UpdateChecker.normalizeVersion('0.3.1'), '0.3.1');
  });

  test('compares semantic versions', () {
    expect(UpdateChecker.isNewer('0.3.1', '0.3.0'), isTrue);
    expect(UpdateChecker.isNewer('0.4.0', '0.3.9'), isTrue);
    expect(UpdateChecker.isNewer('1.0.0', '0.9.9'), isTrue);
    expect(UpdateChecker.isNewer('0.3.0', '0.3.0'), isFalse);
    expect(UpdateChecker.isNewer('0.2.9', '0.3.0'), isFalse);
    expect(UpdateChecker.isNewer('', '0.3.0'), isFalse);
  });

  test('reads release JSON', () {
    final release = ReleaseInfo.fromJson({
      'tag_name': 'v0.3.1',
      'html_url':
          'https://github.com/HanphoneJan/nodepulse/releases/tag/v0.3.1',
      'body': 'Fixes and improvements',
      'published_at': '2026-11-04T16:00:00Z',
    });
    expect(release.version, '0.3.1');
    expect(release.notes, 'Fixes and improvements');
    expect(release.publishedAt, isNotNull);
  });
}
