import 'dart:io';

import 'package:flutter/services.dart';

const _channel = MethodChannel('com.nodepulse.nodepulse/platform');

Future<String?> androidNativeLibraryDir() async {
  if (!Platform.isAndroid) return null;
  try {
    return await _channel.invokeMethod<String>('nativeLibraryDir');
  } catch (_) {
    return null;
  }
}
