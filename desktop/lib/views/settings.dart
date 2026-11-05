import 'dart:convert';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:material_ui/material_ui.dart';

import '../format.dart';
import '../state/app_state.dart';
import '../ui/app_theme.dart';
import '../ui/flclash/color.dart';
import '../ui/flclash/shape.dart';
import '../ui/flclash/typography.dart';
import '../ui/widgets.dart';
import '../update/update_checker.dart';

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

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      spacing: AppSpace.lg,
      children: [
        _appearance(context),
        _environment(context),
        _autoSelection(context),
        _connection(context),
        _health(context),
        _background(context),
        _backup(context),
        _about(context),
      ],
    );
  }

  Widget _appearance(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(
            title: '外观',
            glyph: Glyphs.settings,
            section: true,
          ),
          Text(
            '主题色',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.sm),
          Wrap(
            spacing: AppSpace.sm,
            runSpacing: AppSpace.sm,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              for (final color in _presets)
                _SeedSwatch(
                  color: color,
                  active: state.seed == color,
                  onTap: () => state.setSeed(color),
                ),
              AppButton(
                label: '恢复默认',
                variant: AppButtonVariant.ghost,
                onPressed: () => state.setSeed('#007aff'),
              ),
            ],
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            '当前：${state.seed}',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.lg),
          Text(
            '主题模式',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.sm),
          Align(
            alignment: AlignmentDirectional.centerStart,
            child: SizedBox(
              width: 420,
              child: SegmentedChoice<ThemeMode>(
                value: state.themeMode,
                onChanged: state.setThemeMode,
                options: const [
                  (
                    value: ThemeMode.system,
                    label: '跟随系统',
                    icon: Glyphs.monitor,
                  ),
                  (value: ThemeMode.light, label: '亮色', icon: Glyphs.sun),
                  (value: ThemeMode.dark, label: '暗色', icon: Glyphs.moon),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _environment(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final active = state.activeProfile;
    final activeId = (active['id'] ?? '').toString();
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(title: '环境', glyph: Glyphs.swap, section: true),
          Text(
            '同时切换控制口、订阅和测速参数。',
            style: AppType.secondary(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.md),
          AppTextField(
            key: ValueKey('profile-name-$activeId'),
            label: '环境名称',
            initialValue: (active['name'] ?? '').toString(),
            onChanged: state.renameProfile,
          ),
          const SizedBox(height: AppSpace.md),
          Wrap(
            spacing: AppSpace.sm,
            runSpacing: AppSpace.sm,
            children: [
              AppButton(
                label: '保存当前设置',
                icon: Glyphs.check,
                onPressed: state.saveSettings,
              ),
              AppButton(
                label: '新建环境',
                icon: Glyphs.plus,
                variant: AppButtonVariant.outline,
                onPressed: state.newProfile,
              ),
              AppButton(
                label: '删除当前环境',
                icon: Glyphs.trash,
                variant: AppButtonVariant.ghost,
                onPressed: state.profiles.length <= 1
                    ? null
                    : () async {
                        final confirmed = await showAppConfirm(
                          context,
                          title: '删除当前环境？',
                          message: '这会移除该环境及其本地设置，且无法撤销。',
                          confirmLabel: '删除',
                          destructive: true,
                        );
                        if (confirmed) await state.deleteProfile();
                      },
              ),
            ],
          ),
          const SizedBox(height: AppSpace.lg),
          for (final profile in state.profiles.whereType<Map>())
            Padding(
              padding: const EdgeInsets.only(bottom: AppSpace.sm),
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
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(
            title: '测速与自动选择',
            glyph: Glyphs.signal,
            section: true,
          ),
          Text(
            '余量越大越保守，重启自动优化后生效。',
            style: AppType.secondary(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.md),
          LayoutBuilder(
            builder: (context, constraints) {
              final columns = constraints.maxWidth > 620 ? 2 : 1;
              final width = columns == 2
                  ? (constraints.maxWidth - AppSpace.md) / 2
                  : constraints.maxWidth;
              return Wrap(
                spacing: AppSpace.md,
                runSpacing: AppSpace.md,
                children: [
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: const ValueKey('auto-margin'),
                      label: '切换余量 (%)',
                      initialValue:
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
                    child: AppTextField(
                      key: const ValueKey('auto-confirm'),
                      label: '连续确认 (轮)',
                      initialValue: '${state.settings['auto_confirm'] ?? 2}',
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'auto_confirm',
                        (int.tryParse(value) ?? 2).clamp(1, 20).toInt(),
                      ),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: const ValueKey('auto-min'),
                      label: '最低成功率 (%)',
                      initialValue:
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
                    child: AppTextField(
                      key: const ValueKey('daemon-interval'),
                      label: '检查间隔 (秒)',
                      initialValue:
                          '${state.settings['daemon_interval'] ?? 300}',
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
    final scheme = Theme.of(context).colorScheme;
    final report = state.settings['_health'];
    final checks = report is Map
        ? (report['checks'] as List?) ?? const []
        : const [];
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppCardHeader(
            title: '环境检查',
            glyph: Glyphs.checkCircleOutline,
            section: true,
            actions: [
              AppButton(
                label: '重新检查',
                icon: Glyphs.refresh,
                variant: AppButtonVariant.ghost,
                onPressed: state.refreshHealth,
              ),
            ],
          ),
          Text(
            '只读检查，不会修改配置。',
            style: AppType.secondary(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.md),
          if (checks.isEmpty)
            Text(
              '尚未检查',
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            )
          else
            for (final check in checks.whereType<Map>())
              Padding(
                padding: const EdgeInsets.only(bottom: AppSpace.sm),
                child: SurfaceBox(
                  padding: const EdgeInsets.all(AppSpace.md),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Container(
                        width: 8,
                        height: 8,
                        margin: const EdgeInsets.only(top: 5),
                        decoration: BoxDecoration(
                          shape: BoxShape.circle,
                          color: switch (check['status']) {
                            'ok' => scheme.success,
                            'warn' => scheme.warning,
                            _ => scheme.error,
                          },
                        ),
                      ),
                      const SizedBox(width: AppSpace.sm),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Expanded(
                                  child: Text(
                                    (check['label'] ?? '').toString(),
                                    style: AppType.secondary(context),
                                  ),
                                ),
                                Text(
                                  switch (check['status']) {
                                    'ok' => '正常',
                                    'warn' => '注意',
                                    _ => '异常',
                                  },
                                  style: AppType.caption(
                                    context,
                                  ).copyWith(color: scheme.onSurfaceVariant),
                                ),
                              ],
                            ),
                            if ((check['detail'] ?? '').toString().isNotEmpty)
                              Text(
                                check['detail'].toString(),
                                style: AppType.caption(
                                  context,
                                ).copyWith(color: scheme.onSurfaceVariant),
                              ),
                            if ((check['hint'] ?? '').toString().isNotEmpty)
                              Text(
                                check['hint'].toString(),
                                style: AppType.micro(context).copyWith(
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
        ],
      ),
    );
  }

  Widget _background(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(
            title: '后台与通知',
            glyph: Glyphs.monitor,
            section: true,
          ),
          _SettingSwitch(
            title: '关闭窗口时驻留托盘',
            value: state.settings['close_to_tray'] != false,
            onChanged: (value) => state.updateSetting('close_to_tray', value),
          ),
          _divider(scheme),
          _SettingSwitch(
            title: '开机自动启动',
            value: state.autostart,
            onChanged: state.toggleAutostart,
          ),
          _divider(scheme),
          _SettingSwitch(
            title: '切换、回滚时发送系统通知',
            value: state.settings['notify_on_switch'] != false,
            onChanged: (value) =>
                state.updateSetting('notify_on_switch', value),
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            '关闭后从托盘退出。开机启动会静默拉起后端。',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }

  Widget _backup(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(
            title: '备份与迁移',
            glyph: Glyphs.archive,
            section: true,
          ),
          Text(
            '覆盖前自动备份最近 20 次。',
            style: AppType.secondary(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.md),
          Wrap(
            spacing: AppSpace.sm,
            runSpacing: AppSpace.sm,
            children: [
              AppButton(
                label: '导出设置备份',
                icon: Glyphs.download,
                variant: AppButtonVariant.outline,
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
              ),
              AppButton(
                label: '从备份恢复',
                icon: Glyphs.refresh,
                variant: AppButtonVariant.outline,
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
              ),
            ],
          ),
          const SizedBox(height: AppSpace.sm),
          Text(
            '备份包含 secret，请妥善保管。',
            style: AppType.caption(context).copyWith(color: scheme.warning),
          ),
        ],
      ),
    );
  }

  Widget _connection(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final impact = state.exportImpact;
    final profileKey = state.settings['active_profile'];
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(title: '连接设置', glyph: Glyphs.tune, section: true),
          LayoutBuilder(
            builder: (context, constraints) {
              final two = constraints.maxWidth > 620;
              final width = two
                  ? (constraints.maxWidth - AppSpace.md) / 2
                  : constraints.maxWidth;
              return Wrap(
                spacing: AppSpace.md,
                runSpacing: AppSpace.md,
                children: [
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: ValueKey('api-$profileKey'),
                      label: 'Clash 控制口',
                      initialValue: (state.settings['api'] ?? '').toString(),
                      hint: 'http://127.0.0.1:9090',
                      onChanged: (value) => state.updateSetting('api', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: ValueKey('secret-$profileKey'),
                      label: 'Secret',
                      initialValue: (state.settings['secret'] ?? '').toString(),
                      obscure: true,
                      onChanged: (value) =>
                          state.updateSetting('secret', value),
                    ),
                  ),
                  SizedBox(
                    width: constraints.maxWidth,
                    child: AppTextField(
                      key: ValueKey('config-$profileKey'),
                      label: '订阅配置',
                      initialValue: (state.settings['config_path'] ?? '')
                          .toString(),
                      onChanged: (value) =>
                          state.updateSetting('config_path', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: ValueKey('group-$profileKey'),
                      label: 'Selector 组',
                      initialValue: (state.settings['group'] ?? '').toString(),
                      hint: '留空自动选择',
                      onChanged: (value) => state.updateSetting('group', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: ValueKey('mihomo-$profileKey'),
                      label: 'mihomo 路径',
                      initialValue: (state.settings['mihomo_path'] ?? '')
                          .toString(),
                      onChanged: (value) =>
                          state.updateSetting('mihomo_path', value),
                    ),
                  ),
                  SizedBox(
                    width: constraints.maxWidth,
                    child: AppTextField(
                      key: ValueKey('url-$profileKey'),
                      label: '测试 URL',
                      initialValue: (state.settings['url'] ?? '').toString(),
                      onChanged: (value) => state.updateSetting('url', value),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: ValueKey('concurrency-$profileKey'),
                      label: '并发节点',
                      initialValue: '${state.settings['concurrency'] ?? 8}',
                      keyboardType: TextInputType.number,
                      onChanged: (value) => state.updateSetting(
                        'concurrency',
                        (int.tryParse(value) ?? 8).clamp(1, 100).toInt(),
                      ),
                    ),
                  ),
                  SizedBox(
                    width: width,
                    child: AppTextField(
                      key: ValueKey('timeout-$profileKey'),
                      label: '单轮超时 (ms)',
                      initialValue: '${state.settings['timeout_ms'] ?? 5000}',
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
          const SizedBox(height: AppSpace.lg),
          Wrap(
            spacing: AppSpace.sm,
            runSpacing: AppSpace.sm,
            children: [
              AppButton(
                label: '保存设置',
                icon: Glyphs.check,
                onPressed: state.saveSettings,
              ),
              AppButton(
                label: state.busy && state.busyKind == 'test' ? '测速中…' : '测速',
                variant: AppButtonVariant.tonal,
                onPressed: state.busy ? null : state.testOnly,
              ),
              AppButton(
                label: '切换选中节点',
                variant: AppButtonVariant.tonal,
                onPressed: state.busy ? null : state.switchSelected,
              ),
              AppButton(
                label: '导出稳定配置',
                variant: AppButtonVariant.outline,
                onPressed: state.busy ? null : () => state.exportPreview(false),
              ),
            ],
          ),
          if (impact != null) ...[
            const SizedBox(height: AppSpace.lg),
            _ExportPreview(state: state, impact: impact),
          ],
          const SizedBox(height: AppSpace.lg),
          Divider(color: scheme.outlineVariant.withValues(alpha: 0.5)),
          const SizedBox(height: AppSpace.md),
          Text(
            '控制口状态',
            style: AppType.caption(
              context,
            ).copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: AppSpace.xs),
          if (state.groups.isEmpty)
            Text(
              '控制口未连接',
              style: AppType.secondary(
                context,
              ).copyWith(color: scheme.onSurfaceVariant),
            )
          else
            for (final group in state.groups.whereType<Map>())
              SettingRow(
                title: (group['name'] ?? '').toString(),
                subtitle: '${group['type']} · ${group['now'] ?? '—'}',
                trailing: Text(
                  '${group['members'] ?? 0}',
                  style: AppType.caption(
                    context,
                  ).copyWith(color: scheme.onSurfaceVariant),
                ),
              ),
        ],
      ),
    );
  }

  Widget _about(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AppCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AppCardHeader(title: '关于', glyph: Glyphs.info, section: true),
          SettingRow(
            title: 'NodePulse',
            subtitle: state.version.isEmpty ? '版本未知' : 'v${state.version}',
          ),
          _divider(scheme),
          SettingRow(
            title: '检查更新',
            subtitle: state.updateStatusText,
            trailing: AppButton(
              label: state.checkingUpdate ? '检查中…' : '检查',
              icon: Glyphs.refresh,
              variant: AppButtonVariant.outline,
              loading: state.checkingUpdate,
              onPressed: () => state.checkForUpdate(manual: true),
            ),
          ),
          if (state.update != null) ...[
            const SizedBox(height: AppSpace.sm),
            SurfaceBox(
              padding: const EdgeInsets.all(AppSpace.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    spacing: AppSpace.sm,
                    children: [
                      Expanded(
                        child: Text(
                          '发现新版本 v${state.update!.version}',
                          style: AppType.secondary(
                            context,
                          ).copyWith(fontWeight: FontWeight.w600),
                        ),
                      ),
                      const StatusChip(label: '可更新', tone: StatusTone.warn),
                    ],
                  ),
                  if (state.update!.notes.isNotEmpty) ...[
                    const SizedBox(height: AppSpace.xs),
                    Text(
                      state.update!.notes,
                      maxLines: 4,
                      overflow: TextOverflow.ellipsis,
                      style: AppType.caption(
                        context,
                      ).copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ],
                  const SizedBox(height: AppSpace.sm),
                  Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: AppButton(
                      label: '前往下载',
                      icon: Glyphs.external,
                      onPressed: () => openExternalUrl(state.update!.url),
                    ),
                  ),
                ],
              ),
            ),
          ],
          _divider(scheme),
          SettingRow(title: '许可证', subtitle: 'GPL-3.0'),
          _divider(scheme),
          SettingRow(title: '设计语言', subtitle: 'FlClash contributors · GPL-3.0'),
        ],
      ),
    );
  }

  Widget _divider(ColorScheme scheme) => Padding(
    padding: const EdgeInsets.symmetric(vertical: AppSpace.xs),
    child: Divider(color: scheme.outlineVariant.withValues(alpha: 0.5)),
  );
}

class _SettingSwitch extends StatelessWidget {
  const _SettingSwitch({
    required this.title,
    required this.value,
    required this.onChanged,
  });

  final String title;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return SettingRow(
      title: title,
      trailing: AppSwitch(value: value, onChanged: onChanged),
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
    final parsedColor = Color(0xff000000 | parsed);
    return InkWell(
      onTap: onTap,
      customBorder: const CircleBorder(),
      child: Container(
        width: 32,
        height: 32,
        decoration: BoxDecoration(
          color: parsedColor,
          shape: BoxShape.circle,
          border: Border.all(
            color: active ? scheme.onSurface : scheme.outlineVariant,
            width: active ? 2 : 0.5,
          ),
        ),
        child: active
            ? Icon(
                Glyphs.check,
                size: 16,
                color:
                    ThemeData.estimateBrightnessForColor(parsedColor) ==
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
    final scheme = Theme.of(context).colorScheme;
    return InkWell(
      borderRadius: BorderRadius.circular(AppCorner.md),
      onTap: onTap,
      child: Container(
        constraints: const BoxConstraints(minHeight: 56),
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpace.lg,
          vertical: AppSpace.sm,
        ),
        decoration: ShapeDecoration(
          color: active ? scheme.primaryContainer : scheme.inset,
          shape: AppShape.all(AppCorner.md),
        ),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    (profile['name'] ?? '').toString(),
                    style: AppType.secondary(context).copyWith(
                      color: active
                          ? scheme.onPrimaryContainer
                          : scheme.onSurface,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  Text(
                    '${profile['api'] ?? ''} · ${baseName(profile['config_path']?.toString())}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppType.caption(context).copyWith(
                      color: active
                          ? scheme.onPrimaryContainer.withValues(alpha: 0.8)
                          : scheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            if (active)
              Icon(Glyphs.checkCircle, size: 18, color: scheme.primary),
          ],
        ),
      ),
    );
  }
}

class _ExportPreview extends StatelessWidget {
  const _ExportPreview({required this.state, required this.impact});

  final AppState state;
  final Map<String, dynamic> impact;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SurfaceBox(
      padding: const EdgeInsets.all(AppSpace.lg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  '导出影响',
                  style: AppType.secondary(
                    context,
                  ).copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              Text(
                impact['preserve'] == true ? '保留原规则/分组' : '最小配置',
                style: AppType.caption(
                  context,
                ).copyWith(color: scheme.onSurfaceVariant),
              ),
            ],
          ),
          const SizedBox(height: AppSpace.md),
          Wrap(
            spacing: AppSpace.lg,
            runSpacing: AppSpace.sm,
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
          const SizedBox(height: AppSpace.md),
          SettingRow(
            title: '最小配置（替换规则、分组、DNS/TUN）',
            trailing: AppSwitch(
              value: state.exportMinimal,
              onChanged: (value) => state.exportPreview(value),
            ),
          ),
          const SizedBox(height: AppSpace.sm),
          Wrap(
            spacing: AppSpace.sm,
            children: [
              AppButton(
                label: '确认导出',
                icon: Glyphs.download,
                onPressed: state.confirmExport,
              ),
              AppButton(
                label: '取消',
                variant: AppButtonVariant.ghost,
                onPressed: state.cancelExport,
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
    final scheme = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(value, style: AppType.figure(context, style: AppTextStyle.title3)),
        Text(
          label,
          style: AppType.caption(
            context,
          ).copyWith(color: scheme.onSurfaceVariant),
        ),
      ],
    );
  }
}
