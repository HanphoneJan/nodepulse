import 'dart:convert';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/flclash/typography.dart';
import '../ui/widgets.dart';

const _presets = <String>[
  '#007aff',
  '#cb30e0',
  '#ff2d55',
  '#ff383c',
  '#ff8d28',
  '#ffcc00',
  '#34c759',
  '#8e8e93',
];

class SettingsView extends StatefulWidget {
  const SettingsView({super.key, required this.state});

  final AppState state;

  @override
  State<SettingsView> createState() => _SettingsViewState();
}

class _SettingsViewState extends State<SettingsView> {
  AppState get state => widget.state;
  bool _confirmDelete = false;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: 16,
      children: [
        _appearance(context),
        _environment(context),
        _autoSelection(context),
        _health(context),
        _background(context),
        _backup(context),
        _connection(context),
      ],
    );
  }

  Widget _appearance(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '外观', padding: EdgeInsets.only(bottom: 8)),
          Text(
            '主题色',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              for (final color in _presets)
                _SeedSwatch(
                  color: color,
                  active: state.seed == color,
                  onTap: () => state.setSeed(color),
                ),
              GhostButton(
                label: '恢复默认',
                onPressed: () => state.setSeed('#007aff'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            '当前：${state.seed}',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          Text(
            '主题模式',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerLeft,
            child: SegmentedChoice<ThemeMode>(
              value: state.themeMode,
              onChanged: state.setThemeMode,
              options: const [
                (
                  value: ThemeMode.system,
                  label: '跟随系统',
                  icon: Icons.brightness_auto,
                ),
                (value: ThemeMode.light, label: '亮色', icon: Icons.light_mode),
                (value: ThemeMode.dark, label: '暗色', icon: Icons.dark_mode),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _environment(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final active = state.activeProfile;
    final activeId = (active['id'] ?? '').toString();
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '环境', glyph: Icons.swap_horiz),
          Text(
            '同时切换控制口、订阅和测速参数。',
            style: appText(
              theme,
              AppTextStyle.subheadline,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          _SettingField(
            key: ValueKey('profile-name-$activeId'),
            label: '环境名称',
            value: (active['name'] ?? '').toString(),
            onChanged: state.renameProfile,
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              FilledButton.icon(
                onPressed: state.saveSettings,
                icon: const Icon(Icons.check, size: 18),
                label: const Text('保存当前设置'),
              ),
              OutlinedButton(
                onPressed: state.newProfile,
                child: const Text('新建环境'),
              ),
              TextButton(
                onPressed: state.profiles.length <= 1
                    ? null
                    : () {
                        if (!_confirmDelete) {
                          setState(() => _confirmDelete = true);
                          return;
                        }
                        setState(() => _confirmDelete = false);
                        state.deleteProfile();
                      },
                child: Text(_confirmDelete ? '确认删除？' : '删除当前环境'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          for (final profile in state.profiles.whereType<Map>())
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: _ProfileRow(
                profile: profile,
                active: profile['id'] == state.settings['active_profile'],
                onTap: () => state.activateProfile(profile['id'].toString()),
              ),
            ),
        ],
      ),
    );
  }

  Widget _autoSelection(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '自动选择', glyph: Icons.monitor_heart_outlined),
          Text(
            '余量越大越保守，重启自动优化后生效。',
            style: appText(
              theme,
              AppTextStyle.subheadline,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          LayoutBuilder(
            builder: (context, constraints) {
              final columns = constraints.maxWidth > 620 ? 2 : 1;
              final width = columns == 2
                  ? (constraints.maxWidth - 12) / 2
                  : constraints.maxWidth;
              return Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: const ValueKey('auto-margin'),
                      label: '切换余量 (%)',
                      value:
                          (((state.settings['auto_margin'] as num?) ?? 0.2) *
                                  100)
                              .round()
                              .toString(),
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'auto_margin',
                        ((int.tryParse(value) ?? 20).clamp(1, 100)) / 100,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: const ValueKey('auto-confirm'),
                      label: '连续确认 (轮)',
                      value: '${state.settings['auto_confirm'] ?? 2}',
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'auto_confirm',
                        (int.tryParse(value) ?? 2).clamp(1, 20).toInt(),
                      ),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: const ValueKey('auto-min'),
                      label: '最低成功率 (%)',
                      value:
                          (((state.settings['auto_min_success'] as num?) ??
                                      0.8) *
                                  100)
                              .round()
                              .toString(),
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'auto_min_success',
                        ((int.tryParse(value) ?? 80).clamp(1, 100)) / 100,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: const ValueKey('daemon-interval'),
                      label: '检查间隔 (秒)',
                      value: '${state.settings['daemon_interval'] ?? 300}',
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'daemon_interval',
                        (int.tryParse(value) ?? 300).clamp(30, 86400).toInt(),
                      ),
                    ),
                  ),
                ],
              );
            },
          ),
        ],
      ),
    );
  }

  Widget _health(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final report = state.settings['_health'];
    final checks = report is Map
        ? (report['checks'] as List?) ?? const []
        : const [];
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '环境检查', glyph: Icons.check_circle_outline),
          Text(
            '只读检查，不会修改配置。',
            style: appText(
              theme,
              AppTextStyle.subheadline,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          if (checks.isEmpty)
            Text(
              '尚未检查',
              style: appText(
                theme,
                AppTextStyle.subheadline,
              ).copyWith(color: scheme.onSurfaceVariant),
            )
          else
            for (final check in checks.whereType<Map>())
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: SurfaceBox(
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Container(
                        width: 8,
                        height: 8,
                        margin: const EdgeInsets.only(top: 6),
                        decoration: BoxDecoration(
                          shape: BoxShape.circle,
                          color: switch (check['status']) {
                            'ok' => scheme.success,
                            'warn' => scheme.warning,
                            _ => scheme.error,
                          },
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Expanded(
                                  child: Text(
                                    (check['label'] ?? '').toString(),
                                    style: appText(
                                      theme,
                                      AppTextStyle.subheadline,
                                    ),
                                  ),
                                ),
                                Text(
                                  switch (check['status']) {
                                    'ok' => '正常',
                                    'warn' => '注意',
                                    _ => '异常',
                                  },
                                  style: appText(
                                    theme,
                                    AppTextStyle.footnote,
                                  ).copyWith(color: scheme.onSurfaceVariant),
                                ),
                              ],
                            ),
                            if ((check['detail'] ?? '').toString().isNotEmpty)
                              Text(
                                check['detail'].toString(),
                                style: appText(
                                  theme,
                                  AppTextStyle.footnote,
                                ).copyWith(color: scheme.onSurfaceVariant),
                              ),
                            if ((check['hint'] ?? '').toString().isNotEmpty)
                              Text(
                                check['hint'].toString(),
                                style: appText(theme, AppTextStyle.footnote)
                                    .copyWith(
                                      color: scheme.onSurfaceVariant.withValues(
                                        alpha: 0.75,
                                      ),
                                    ),
                              ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
          const SizedBox(height: 4),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton.icon(
              onPressed: state.refreshHealth,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('重新检查'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _background(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(
            label: '后台与通知',
            glyph: Icons.desktop_windows_outlined,
            padding: EdgeInsets.only(bottom: 4),
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            value: state.settings['close_to_tray'] != false,
            onChanged: (value) => state.updateSetting('close_to_tray', value),
            title: Text(
              '关闭窗口时驻留托盘',
              style: appText(theme, AppTextStyle.subheadline),
            ),
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            value: state.autostart,
            onChanged: state.toggleAutostart,
            title: Text(
              '开机自动启动',
              style: appText(theme, AppTextStyle.subheadline),
            ),
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            value: state.settings['notify_on_switch'] != false,
            onChanged: (value) =>
                state.updateSetting('notify_on_switch', value),
            title: Text(
              '切换、回滚时发送系统通知',
              style: appText(theme, AppTextStyle.subheadline),
            ),
          ),
          Text(
            '关闭后从托盘退出。开机启动会静默拉起后端。',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }

  Widget _backup(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '备份与迁移', glyph: Icons.archive_outlined),
          Text(
            '覆盖前自动备份最近 20 次。',
            style: appText(
              theme,
              AppTextStyle.subheadline,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              OutlinedButton.icon(
                onPressed: () async {
                  final data = await state.backupSettings();
                  if (data == null) return;
                  final path = await FilePicker.saveFile(
                    dialogTitle: '导出设置备份',
                    fileName: 'nodepulse-settings.json',
                  );
                  if (path != null) {
                    await File(path).writeAsString(data);
                  }
                },
                icon: const Icon(Icons.download_outlined, size: 18),
                label: const Text('导出设置备份'),
              ),
              OutlinedButton.icon(
                onPressed: () async {
                  final result = await FilePicker.pickFiles(
                    type: FileType.custom,
                    allowedExtensions: const ['json'],
                    withData: true,
                  );
                  if (result == null || result.files.isEmpty) return;
                  final file = result.files.first;
                  final text = file.bytes != null
                      ? utf8.decode(file.bytes!)
                      : await File(file.path!).readAsString();
                  await state.restoreSettings(text);
                },
                icon: const Icon(Icons.refresh, size: 18),
                label: const Text('从备份恢复'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            '备份包含 secret，请妥善保管。',
            style: appText(
              theme,
              AppTextStyle.footnote,
            ).copyWith(color: scheme.warning),
          ),
        ],
      ),
    );
  }

  Widget _connection(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final impact = state.exportImpact;
    final profileKey = state.settings['active_profile'];
    return CommonCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const InfoHeader(label: '连接设置', glyph: Icons.tune),
          LayoutBuilder(
            builder: (context, constraints) {
              final two = constraints.maxWidth > 620;
              final width = two
                  ? (constraints.maxWidth - 12) / 2
                  : constraints.maxWidth;
              return Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: ValueKey('api-$profileKey'),
                      label: 'Clash 控制口',
                      value: (state.settings['api'] ?? '').toString(),
                      hint: 'http://127.0.0.1:9090',
                      onChanged: (value) => state.updateSetting('api', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: ValueKey('secret-$profileKey'),
                      label: 'Secret',
                      value: (state.settings['secret'] ?? '').toString(),
                      obscure: true,
                      onChanged: (value) =>
                          state.updateSetting('secret', value),
                    ),
                  ),
                  SizedBox(
                    width: constraints.maxWidth,
                    child: _SettingField(
                      key: ValueKey('config-$profileKey'),
                      label: '订阅配置',
                      value: (state.settings['config_path'] ?? '').toString(),
                      onChanged: (value) =>
                          state.updateSetting('config_path', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: ValueKey('group-$profileKey'),
                      label: 'Selector 组',
                      value: (state.settings['group'] ?? '').toString(),
                      hint: '留空自动选择',
                      onChanged: (value) => state.updateSetting('group', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: ValueKey('mihomo-$profileKey'),
                      label: 'mihomo 路径',
                      value: (state.settings['mihomo_path'] ?? '').toString(),
                      onChanged: (value) =>
                          state.updateSetting('mihomo_path', value),
                    ),
                  ),
                  SizedBox(
                    width: constraints.maxWidth,
                    child: _SettingField(
                      key: ValueKey('url-$profileKey'),
                      label: '测试 URL',
                      value: (state.settings['url'] ?? '').toString(),
                      onChanged: (value) => state.updateSetting('url', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: ValueKey('concurrency-$profileKey'),
                      label: '并发节点',
                      value: '${state.settings['concurrency'] ?? 8}',
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'concurrency',
                        (int.tryParse(value) ?? 8).clamp(1, 100).toInt(),
                      ),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: _SettingField(
                      key: ValueKey('timeout-$profileKey'),
                      label: '单轮超时 (ms)',
                      value: '${state.settings['timeout_ms'] ?? 5000}',
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'timeout_ms',
                        (int.tryParse(value) ?? 5000).clamp(500, 60000).toInt(),
                      ),
                    ),
                  ),
                ],
              );
            },
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              FilledButton.icon(
                onPressed: state.saveSettings,
                icon: const Icon(Icons.check, size: 18),
                label: const Text('保存设置'),
              ),
              FilledButton.tonal(
                onPressed: state.busy ? null : state.testOnly,
                child: Text(
                  state.busy && state.busyKind == 'test' ? '测速中…' : '测速',
                ),
              ),
              FilledButton.tonal(
                onPressed: state.busy ? null : state.switchSelected,
                child: const Text('切换选中节点'),
              ),
              OutlinedButton(
                onPressed: state.busy ? null : () => state.exportPreview(false),
                child: const Text('导出稳定配置'),
              ),
            ],
          ),
          if (impact != null) ...[
            const SizedBox(height: 12),
            _ExportPreview(state: state, impact: impact),
          ],
          const SizedBox(height: 12),
          Divider(
            height: 1,
            color: scheme.outlineVariant.withValues(alpha: 0.4),
          ),
          const SizedBox(height: 12),
          Text(
            '控制口状态',
            style: appText(
              theme,
              AppTextStyle.headline,
            ).copyWith(color: scheme.onSurfaceVariant.withValues(alpha: 0.8)),
          ),
          const SizedBox(height: 8),
          if (state.groups.isEmpty)
            Text(
              '控制口未连接',
              style: appText(
                theme,
                AppTextStyle.subheadline,
              ).copyWith(color: scheme.onSurfaceVariant),
            )
          else
            for (final group in state.groups.whereType<Map>())
              SettingRow(
                title: (group['name'] ?? '').toString(),
                subtitle: '${group['type']} · ${group['now'] ?? '—'}',
                trailing: Text(
                  '${group['members'] ?? 0}',
                  style: appText(
                    theme,
                    AppTextStyle.footnote,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              ),
        ],
      ),
    );
  }
}

class _SeedSwatch extends StatelessWidget {
  const _SeedSwatch({
    required this.color,
    required this.active,
    required this.onTap,
  });

  final String color;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final parsed =
        int.tryParse(color.replaceFirst('#', ''), radix: 16) ?? 0x007aff;
    return InkWell(
      onTap: onTap,
      customBorder: const CircleBorder(),
      child: Container(
        width: 32,
        height: 32,
        decoration: BoxDecoration(
          color: Color(0xff000000 | parsed),
          shape: BoxShape.circle,
          border: active
              ? Border.all(color: scheme.onSurface, width: 2)
              : Border.all(color: scheme.outlineVariant, width: 0.5),
        ),
        child: active
            ? Icon(
                Icons.check,
                size: 16,
                color:
                    ThemeData.estimateBrightnessForColor(
                          Color(0xff000000 | parsed),
                        ) ==
                        Brightness.dark
                    ? Colors.white
                    : Colors.black,
              )
            : null,
      ),
    );
  }
}

class _ProfileRow extends StatelessWidget {
  const _ProfileRow({
    required this.profile,
    required this.active,
    required this.onTap,
  });

  final Map profile;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onTap: onTap,
      child: Container(
        constraints: const BoxConstraints(minHeight: 56),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          color: active ? scheme.secondaryContainer : scheme.inset,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    (profile['name'] ?? '').toString(),
                    style: appText(theme, AppTextStyle.subheadline).copyWith(
                      color: active
                          ? scheme.onSecondaryContainer
                          : scheme.onSurface,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  Text(
                    '${profile['api'] ?? ''} · ${baseName(profile['config_path']?.toString())}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: appText(theme, AppTextStyle.footnote).copyWith(
                      color: active
                          ? scheme.onSecondaryContainer.withValues(alpha: 0.8)
                          : scheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            if (active) Icon(Icons.check, size: 18, color: scheme.success),
          ],
        ),
      ),
    );
  }
}

class _SettingField extends StatelessWidget {
  const _SettingField({
    super.key,
    required this.label,
    required this.value,
    required this.onChanged,
    this.hint,
    this.keyboardType,
    this.obscure = false,
  });

  final String label;
  final String value;
  final ValueChanged<String> onChanged;
  final String? hint;
  final TextInputType? keyboardType;
  final bool obscure;

  @override
  Widget build(BuildContext context) {
    return TextFormField(
      initialValue: value,
      onChanged: onChanged,
      keyboardType: keyboardType,
      obscureText: obscure,
      decoration: InputDecoration(labelText: label, hintText: hint),
    );
  }
}

class _ExportPreview extends StatelessWidget {
  const _ExportPreview({required this.state, required this.impact});

  final AppState state;
  final Map<String, dynamic> impact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return SurfaceBox(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  '导出影响',
                  style: appText(
                    theme,
                    AppTextStyle.subheadline,
                  ).copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              Text(
                impact['preserve'] == true ? '保留原规则/分组' : '最小配置',
                style: appText(
                  theme,
                  AppTextStyle.footnote,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 16,
            runSpacing: 8,
            children: [
              _Impact(label: '保留节点', value: '${impact['included'] ?? 0}'),
              _Impact(
                label: '排除节点',
                value: '${(impact['excluded'] as List?)?.length ?? 0}',
              ),
              _Impact(label: '分组', value: '${impact['groups'] ?? 0}'),
              _Impact(label: '规则', value: '${impact['rules'] ?? 0}'),
            ],
          ),
          const SizedBox(height: 8),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            value: state.exportMinimal,
            onChanged: (value) => state.exportPreview(value),
            title: Text(
              '最小配置（替换规则、分组、DNS/TUN）',
              style: appText(theme, AppTextStyle.footnote),
            ),
          ),
          Wrap(
            spacing: 8,
            children: [
              FilledButton.icon(
                onPressed: state.confirmExport,
                icon: const Icon(Icons.download, size: 18),
                label: const Text('确认导出'),
              ),
              TextButton(
                onPressed: state.cancelExport,
                child: const Text('取消'),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _Impact extends StatelessWidget {
  const _Impact({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          value,
          style: appText(
            theme,
            AppTextStyle.title3,
          ).copyWith(fontWeight: FontWeight.w600),
        ),
        Text(
          label,
          style: appText(
            theme,
            AppTextStyle.footnote,
          ).copyWith(color: scheme.onSurfaceVariant),
        ),
      ],
    );
  }
}
