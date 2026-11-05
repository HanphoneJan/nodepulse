// Ported from FlClash (https://github.com/chen08209/FlClash).
// Copyright (c) FlClash contributors, licensed under GPL-3.0.
// Modified for NodePulse.

import 'package:material_ui/material_ui.dart';

enum _HanVariant { simplified, traditional, japanese, korean }

// The system fallback picks Han glyph forms by the UI locale, which draws
// Chinese proxy names with Japanese forms under an English UI on Windows.
const _hanFamilies = <TargetPlatform, Map<_HanVariant, List<String>>>{
  TargetPlatform.windows: {
    _HanVariant.simplified: [
      'Microsoft YaHei UI',
      'Microsoft YaHei',
      'DengXian',
      'SimHei',
    ],
    _HanVariant.traditional: ['Microsoft JhengHei UI', 'Microsoft JhengHei'],
    _HanVariant.japanese: ['Yu Gothic UI', 'Meiryo UI'],
    _HanVariant.korean: ['Malgun Gothic'],
  },
  TargetPlatform.macOS: {
    _HanVariant.simplified: ['PingFang SC'],
    _HanVariant.traditional: ['PingFang TC'],
    _HanVariant.japanese: ['Hiragino Sans'],
    _HanVariant.korean: ['Apple SD Gothic Neo'],
  },
  TargetPlatform.linux: {
    _HanVariant.simplified: [
      'Noto Sans CJK SC',
      'Noto Sans SC',
      'Source Han Sans SC',
      'WenQuanYi Micro Hei',
    ],
    _HanVariant.traditional: [
      'Noto Sans CJK TC',
      'Noto Sans TC',
      'Source Han Sans TC',
    ],
    _HanVariant.japanese: [
      'Noto Sans CJK JP',
      'Noto Sans JP',
      'Source Han Sans',
    ],
    _HanVariant.korean: [
      'Noto Sans CJK KR',
      'Noto Sans KR',
      'Source Han Sans K',
    ],
  },
};

_HanVariant _hanVariantOf(Iterable<Locale> locales) {
  for (final locale in locales) {
    switch (locale.languageCode) {
      case 'zh':
        return _isTraditional(locale)
            ? _HanVariant.traditional
            : _HanVariant.simplified;
      case 'ja':
        return _HanVariant.japanese;
      case 'ko':
        return _HanVariant.korean;
    }
  }
  return _HanVariant.simplified;
}

bool _isTraditional(Locale locale) => switch (locale.scriptCode) {
  'Hant' => true,
  'Hans' => false,
  _ => const {'TW', 'HK', 'MO'}.contains(locale.countryCode),
};

/// The CJK UI face is the primary font, not a fallback. Windows otherwise
/// resolves a missing fallback to SimSun, whose thin horizontal strokes make
/// dense characters such as 置 look cramped.
String? appFontFamilyOf(TargetPlatform platform) => switch (platform) {
  TargetPlatform.windows => 'Microsoft YaHei UI',
  TargetPlatform.macOS => 'PingFang SC',
  TargetPlatform.linux => 'Noto Sans CJK SC',
  _ => null,
};

List<String> appFontFamilyFallback({
  required TargetPlatform platform,
  required Iterable<Locale> locales,
  String? fontFamily,
}) {
  final base = Typography.material2021(platform: platform).black.bodyMedium!;
  return [
    if (fontFamily != null) ?base.fontFamily,
    ...?base.fontFamilyFallback,
    ...?_hanFamilies[platform]?[_hanVariantOf(locales)],
    'Twemoji',
  ];
}
