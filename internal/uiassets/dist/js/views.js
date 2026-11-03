import { html } from "./html.js";
import { useMemo, useState } from "../vendor/hooks.module.js";
import {
  Button,
  Icon,
  M3Select,
  ScoreRing,
  SeedPicker,
  Segmented,
  State,
  SwitchRow,
  HeroPending,
  LoadingRows,
} from "./components.js";
import { GroupTable, NodeTable } from "./tables.js";
import { baseName, confidenceLabel, formatClock, hostOf, kindLabel, reasonLabel, timeAgo } from "./format.js";

function Field({ label, id, value, onChange, onBlur, placeholder, type = "text", inputMode, autoComplete, spellcheck = false, className = "" }) {
  return html`<label class=${"text-field " + className}>
    <span>${label}</span>
    <input
      id=${id}
      type=${type}
      value=${value}
      placeholder=${placeholder}
      inputmode=${inputMode}
      autocomplete=${autoComplete}
      spellcheck=${spellcheck}
      onInput=${(event) => onChange(event.currentTarget.value)}
      onBlur=${onBlur ? (event) => onBlur(event.currentTarget.value) : undefined}
    />
  </label>`;
}

export function Dashboard({
  settings, status, results, best, selected, busy, busyKind, elapsed, progress, onCancel, lastTestAt,
  controllers, clients, subscriptions, daemon, daemonInterval, latestEvent, groups, subURL,
  downloadUrl, onDownloadUrl, downloadResult, onDownloadMode,
  detecting, onDetect,
  onOneClick, onTest, onTestAll, onSelectNode, onRefresh, onController, onSettings,
  onRounds, onDaemon, onInterval, onSubURL, onImport,
}) {
  const connected = !!status.connected;
  const current = status.current || "—";
  const meta = status.groupName ? `${status.groupName} · ${status.members} 个成员` : "连接客户端后显示";
  const heroLoading = busy && ["oneclick", "test", "all"].includes(busyKind);
  const testing = heroLoading;
  const canImport = !!subURL.trim();
  const intervalOptions = [
    { value: "300", label: "5 分钟" },
    { value: "600", label: "10 分钟" },
    { value: "1800", label: "30 分钟" },
  ];
  if (!intervalOptions.some((option) => option.value === String(daemonInterval))) {
    intervalOptions.unshift({ value: String(daemonInterval), label: `${daemonInterval} 秒` });
  }
  const eventCode = latestEvent ? (latestEvent.reason_code || latestEvent.reason) : "";
  const eventScores = latestEvent
    ? [Number(latestEvent.current_score) || 0, Number(latestEvent.best_score) || 0]
    : [0, 0];

  return html`<section id="view-dashboard" class="view active">
    <div class="hero md-card" aria-busy=${heroLoading ? "true" : undefined}>
      <div class="hero-main">
        <div class="label-row">
          <span class="label">当前节点</span>
          <span class=${"status-chip " + (status.error ? "error" : connected ? "ok" : "warn")}>${status.error ? "连接失败" : connected ? "已连接" : "未连接"}</span>
        </div>
        <div class="hero-node" title=${current}>${current}</div>
        <div class="hero-meta" title=${meta}>${meta}</div>
        <div class="hero-actions">
          <${Button} variant="filled" icon="play" loading=${busy && busyKind === "oneclick"} onClick=${onOneClick}>
            ${busy && busyKind === "oneclick" ? "优化中…" : "一键优化"}
          <//>
          <${Button} variant="tonal" icon="refresh" loading=${busy && busyKind === "test"} onClick=${onTest}>
            ${busy && busyKind === "test" ? "测速中…" : "只测速"}
          <//>
          <${Button} variant="tonal" icon="list" loading=${busy && busyKind === "all"} onClick=${onTestAll}>
            ${busy && busyKind === "all" ? "联合测速中…" : "全部订阅"}
          <//>
          <div class="strength"><span>强度</span>
            <${M3Select}
              label="测速强度"
              value=${String(settings.rounds)}
              options=${[
                { value: "3", label: "快速 3 轮" },
                { value: "5", label: "标准 5 轮" },
                { value: "10", label: "精确 10 轮" },
              ]}
              onChange=${(value) => onRounds(Number(value))}
            />
          </div>
          ${testing ? html`<${Button} variant="text" size="small" onClick=${onCancel}>取消<//>` : null}
        </div>
      </div>
      <div class="hero-side">
        ${best
          ? html`<${ScoreRing} value=${best.score} bestName=${best.name} />`
          : html`<${HeroPending} members=${status.members} rounds=${settings.rounds} concurrency=${settings.concurrency} lastTestAt=${lastTestAt} />`}
      </div>
    </div>

    <div class="grid-2">
      <div class="md-card">
        <div class="card-title"><${Icon} name="link" />连接
          <button type="button" class="card-action" disabled=${detecting} onClick=${onDetect}>${detecting ? "检测中…" : "重新检测"}</button>
        </div>
        <p class="muted">${controllers.length
          ? "已自动检测本机 Clash 控制口"
          : clients.some((client) => client.running)
            ? "已检测到客户端，但控制口没有响应。请先在客户端里打开外部控制。"
            : "没检测到客户端。也可以只选订阅，用 mihomo 测速。"}</p>
        ${clients.length
          ? html`<div class="client-list" aria-label="检测到的客户端">
              ${clients.map((client) => html`<div class=${"client-row" + (client.running ? " on" : "")}>
                <span class="client-dot" aria-hidden="true"></span>
                <span class="client-body">
                  <strong>${client.name}</strong>
                  <small>${client.hint || (client.controller ? `控制口 ${client.controller}` : "已安装")}</small>
                </span>
                <span class="client-state">${client.running ? "运行中" : "未运行"}</span>
              </div>`)}
            </div>`
          : null}
        <div class="chips" role="group" aria-label="Clash 控制口">
          ${controllers.length === 0
            ? html`<span class="muted">未发现可用控制口</span>`
            : controllers.map((controller) => {
                const needSecret = controller.auth_required && !settings.secret;
                const label = needSecret ? `${controller.url}（需要 secret）` : controller.url + (controller.version ? ` · ${controller.version}` : "");
                return html`<button
                  type="button"
                  class=${"chip" + (settings.api === controller.url ? " active" : "")}
                  aria-pressed=${settings.api === controller.url ? "true" : "false"}
                  onClick=${() => onController(controller.url, controller.secret)}
                ><span class="dot"></span>${label}</button>`;
              })}
        </div>

        <details class="help">
          <summary>没检测到？</summary>
          <p><b>FlClash</b>：设置 → 网络 → 外部控制，填 <code>127.0.0.1:9090</code>。<br />
          <b>Clash Verge</b>：设置 → Clash → 外部控制。<br />
          不开也能用：选订阅 + mihomo，只测速不切换。</p>
        </details>

        <div class="divider"></div>
        <div class="card-title small"><${Icon} name="download" />订阅</div>
        <div class="row mt sub-row">
          <${M3Select}
            label="订阅文件"
            value=${settings.configPath}
            options=${(() => {
              const options = subscriptions.map((path) => ({ value: path, label: path }));
              if (settings.configPath && !subscriptions.includes(settings.configPath)) {
                options.unshift({ value: settings.configPath, label: settings.configPath });
              }
              return options.length ? options : [{ value: "", label: "没有自动找到订阅文件" }];
            })()}
            onChange=${(value) => onSettings({ configPath: value })}
          />
        </div>
        <div class="row mt sub-row">
          <input
            class="text-field-input"
            id="subURL"
            type="url"
            inputmode="url"
            spellcheck="false"
            placeholder="https://…/subscribe?token=…"
            aria-label="订阅 URL"
            value=${subURL}
            onInput=${(event) => onSubURL(event.currentTarget.value)}
          />
          <${Button} variant="tonal" size="small" loading=${busy && busyKind === "import"} disabled=${!canImport} onClick=${onImport}>导入<//>
        </div>
        <p class="sub-path clamp-2" title=${settings.configPath}>${settings.configPath ? `当前订阅: ${settings.configPath}` : "当前没有选择订阅文件"}</p>
      </div>

      <div class="md-card daemon-card">
        <div class="card-title"><${Icon} name="settings" />后台自动优化</div>
        <p class="muted">当前节点明显变差时自动切换，失败会回滚。</p>
        <div class="mt">
          <${SwitchRow} label="自动优化" checked=${daemon} onChange=${onDaemon} />
        </div>
        <div class="row between mt">
          <span class="muted">检查间隔</span>
          <${M3Select}
            label="自动优化检查间隔"
            value=${String(daemonInterval)}
            options=${intervalOptions}
            onChange=${(value) => onInterval(Number(value))}
          />
        </div>
        <div class="meta-list">
          <div class="meta-item"><${Icon} name="checkCircle" />连续 ${settings.autoConfirm} 次确认后切换</div>
          <div class="meta-item"><${Icon} name="checkCircle" />切换后验证，失败自动回滚</div>
          <div class="meta-item"><${Icon} name="checkCircle" />评分滞后 ${Math.round(settings.autoMargin * 100)}% 才行动</div>
        </div>
        <div class=${"auto-decision" + (latestEvent && (!latestEvent.ok || ["rollback", "error"].includes(latestEvent.kind)) ? " auto-decision--error" : "")}>
          <div class="auto-decision-head"><${Icon} name="monitoring" />最近自动选择</div>
          ${latestEvent
            ? html`<div class="auto-decision-title">${latestEvent.kind === "switch" ? "已切换" : latestEvent.kind === "rollback" ? "已回滚" : (latestEvent.kind === "error" || !latestEvent.ok) ? "检查失败" : "保持当前"}</div>
                <div class="auto-decision-main">${latestEvent.from || "—"} ${latestEvent.to ? `→ ${latestEvent.to}` : ""}</div>
                <div class="auto-decision-meta">${formatClock(latestEvent.time)} · ${kindLabel(latestEvent.kind)} · ${reasonLabel(eventCode)}</div>
                ${eventScores[0] || eventScores[1]
                  ? html`<div class="auto-decision-scores">当前 ${eventScores[0].toFixed(3)} · 候选 ${eventScores[1].toFixed(3)}${latestEvent.confidence ? ` · ${confidenceLabel(latestEvent.confidence)}置信` : ""}</div>`
                  : null}
                ${latestEvent.rollback_attempted
                  ? html`<div class="auto-decision-rollback">${latestEvent.rollback_ok ? "已回滚到原节点" : "回滚失败，请手动检查"}</div>`
                  : null}`
            : html`<div class="muted">开启后，每次检查会在这里显示“为什么切换”。</div>`}
        </div>
        <div class="status-line mt" aria-live="polite">${daemon ? `运行中 · 每 ${daemonInterval < 60 ? `${daemonInterval} 秒` : `${Math.round(daemonInterval / 60)} 分钟`}检查` : "未开启"}</div>
      </div>
    </div>

    <div class="md-card">
      <div class="card-title"><${Icon} name="download" />下载模式</div>
      <p class="muted">粘贴直接文件 URL。先按常规评分选前 5 个候选，再对目标站点做 3 MiB / 5 秒的小范围下载探测；不可达节点淘汰，只在开始前切换一次。</p>
      <div class="download-row mt">
        <${Field}
          label="目标文件 URL"
          id="downloadUrl"
          value=${downloadUrl}
          placeholder="https://example.com/file.apk"
          onChange=${onDownloadUrl}
        />
        <${Button} variant="filled" icon="download" loading=${busy && busyKind === "download"} onClick=${() => onDownloadMode()}>测速并选节点<//>
      </div>
      ${downloadResult
        ? html`<div class=${"download-result" + (downloadResult.reason_code === "download-unreachable" ? " download-result--error" : "")}>
            <div class="download-result-head">
              <strong>${downloadResult.reason_code === "download-unreachable" ? "没有可用路径" : downloadResult.reason_code === "direct-recommended" ? "建议直连" : `已选择 ${downloadResult.download?.best?.node || downloadResult.best?.name || "—"}`}</strong>
              ${downloadResult.download?.cache_hits ? html`<span class="muted">缓存 ${downloadResult.download.cache_hits}/${downloadResult.download.top_k}</span>` : null}
            </div>
            <div class="download-result-main">${downloadResult.message}</div>
            ${downloadResult.download?.direct
              ? html`<div class=${"probe-row probe-row--direct" + (downloadResult.download.direct.ok ? "" : " probe-row--bad")}>
                  <span class="probe-node">直连</span>
                  <span>${downloadResult.download.direct.ok ? `${Math.round(downloadResult.download.direct.p25_kibps)} KiB/s · ${downloadResult.download.direct.ttfb_ms}ms` : (downloadResult.download.direct.error || "不可达")}</span>
                  <span class="muted">${downloadResult.download.direct.cached ? "缓存" : `${Math.round((downloadResult.download.direct.bytes || 0) / 1024)} KiB`}${downloadResult.download.direct_recommended ? " · 推荐" : ""}</span>
                </div>`
              : null}
            ${downloadResult.download?.probes?.length
              ? html`<div class="probe-list">
                  ${downloadResult.download.probes.map((probe) => html`<div class=${"probe-row" + (probe.ok ? "" : " probe-row--bad")}>
                    <span class="probe-node">${probe.node}</span>
                    <span>${probe.ok ? `${Math.round(probe.p25_kibps)} KiB/s · ${probe.ttfb_ms}ms` : (probe.error || "不可达")}</span>
                    <span class="muted">${probe.cached ? "缓存" : `${Math.round(probe.bytes / 1024)} KiB`}</span>
                  </div>`)}
                </div>`
              : null}
          </div>`
        : null}
    </div>

    <div class="md-card">
      <div class="row between">
        <div class="card-title"><${Icon} name="list" />测速结果</div>
        <span class="muted">${best ? `推荐: ${best.name}` : testing ? `已运行 ${elapsed}s` : ""}</span>
      </div>
      ${testing
        ? html`<div class="progress-line" role="status" aria-live="polite">
            <span>${progress && progress.total ? `${progress.done || 0}/${progress.total}` : "正在准备…"}${progress && progress.current ? ` · ${progress.current}` : ""}</span>
            <span>${elapsed}s</span>
          </div>
          <${LoadingRows} label="正在测速，请稍候" />`
        : results.length
          ? html`<${NodeTable} rows=${results} best=${best} selected=${selected} onSelect=${onSelectNode} label="测速结果" />`
          : html`<${State}
              icon="speed"
              title="还没有测速结果"
              body="点“一键优化”开始，或先检查控制口和订阅。"
              actions=${html`<${Button} variant="tonal" size="small" onClick=${onTest}>只测速<//>`}
            />`}
    </div>
  </section>`;
}

export function Nodes({ results, best, selected, onSelect, onRefresh, onSwitch, canSwitch, busy }) {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return q ? results.filter((row) => row.name.toLowerCase().includes(q)) : results;
  }, [query, results]);

  return html`<section id="view-nodes" class="view active">
    <div class="md-card">
      <div class="nodes-header">
        <div class="nodes-title card-title">节点排名</div>
        <div class="nodes-tools">
          <div class="search">
            <${Icon} name="search" />
            <input
              type="search"
              placeholder="搜索节点"
              aria-label="搜索节点"
              value=${query}
              onInput=${(event) => setQuery(event.currentTarget.value)}
            />
          </div>
          <${Button} variant="text" icon="refresh" onClick=${onRefresh}>刷新状态<//>
        </div>
      </div>

      ${busy
        ? html`<${LoadingRows} label="正在刷新节点状态" />`
        : results.length === 0
          ? html`<${State} icon="monitoring" title="还没有测试结果" body="先在仪表盘点“一键优化”，这里会显示完整排名。" />`
          : filtered.length === 0
            ? html`<${State} icon="search" title="没有匹配的节点" body="换一个关键词试试。" />`
            : html`<${NodeTable} rows=${filtered} best=${best} selected=${selected} onSelect=${onSelect} label="节点排名" />`}

      ${filtered.length > 0 && selected
        ? html`<div class="nodes-selection">
            <span>已选择 <strong>${selected}</strong></span>
            <${Button} variant="tonal" size="small" disabled=${!canSwitch} onClick=${onSwitch}>切换选中节点<//>
          </div>`
        : null}
      <span class="sr-only" aria-live="polite">${filtered.length} 个节点</span>
    </div>
  </section>`;
}

export function Settings({
  settings, groups, seed, mode, busy, busyKind, progress, onCancel, lastTestAt,
  onSettings, onSeed, onResetSeed, onMode, onSave, onTest, onSwitch, onExport, canSwitch, canExport,
  exportImpact, exportMinimal, onConfirmExport, onCancelExport, onExportMinimal,
  desktop, autostart, onAutostart, onBackup, onRestore,
  health, healthBusy, onHealth,
  profiles, activeProfile, onActivateProfile, onRenameProfile, onProfileNameBlur, onSaveProfile, onNewProfile, onDeleteProfile,
}) {
  const currentProfile = profiles.find((profile) => profile.id === activeProfile) || profiles[0];
  const [confirmDelete, setConfirmDelete] = useState(false);
  return html`<section id="view-settings" class="view active">
    <div class="md-card">
      <div class="card-title">外观</div>
      <p class="muted">主题色会生成整套 Material 3 明暗色板；默认跟随系统主题，字体使用系统字体。</p>
      <div class="appearance-row">
        <${SeedPicker} seed=${seed} onSeed=${onSeed} onReset=${onResetSeed} />
        <div class="mode-block">
          <span class="field-label">主题模式</span>
          <${Segmented}
            label="主题模式"
            value=${mode}
            onChange=${onMode}
            options=${[
              { value: "system", label: "跟随系统", icon: "monitor" },
              { value: "light", label: "亮色", icon: "sun" },
              { value: "dark", label: "暗色", icon: "moon" },
            ]}
          />
        </div>
      </div>
    </div>

    <div class="md-card">
      <div class="card-title"><${Icon} name="swap" />环境</div>
      <p class="muted">环境会同时切换控制口、Secret、订阅、Selector 组和测速参数；顶栏的切换按钮可在任意页面快速切换。</p>
      <div class="text-field-grid mt">
        <${Field} label="环境名称" id="profileName" className="span-2" value=${currentProfile ? currentProfile.name : ""} placeholder="例如：FlClash + 日本订阅" onChange=${onRenameProfile} onBlur=${onProfileNameBlur} />
      </div>
      <div class="profile-settings-actions">
        <${Button} variant="tonal" icon="check" onClick=${onSaveProfile}>保存当前设置<//>
        <${Button} variant="outlined" onClick=${onNewProfile}>新建环境<//>
        <${Button} variant="text" disabled=${profiles.length <= 1 || busy} onClick=${() => {
          if (!confirmDelete) {
            setConfirmDelete(true);
            return;
          }
          setConfirmDelete(false);
          onDeleteProfile();
        }}>${confirmDelete ? "确认删除？" : "删除当前环境"}<//>
      </div>
      <div class="profile-settings-list" role="group" aria-label="已保存环境">
        ${profiles.map((profile) => html`<button
          key=${profile.id}
          type="button"
          class=${"profile-settings-row" + (profile.id === activeProfile ? " active" : "")}
          aria-pressed=${profile.id === activeProfile ? "true" : "false"}
          onClick=${() => { setConfirmDelete(false); onActivateProfile(profile.id); }}
        >
          <span class="profile-item-body">
            <strong>${profile.name}</strong>
            <small>${hostOf(profile.api)} · ${baseName(profile.configPath)}</small>
          </span>
          ${profile.id === activeProfile ? html`<${Icon} name="check" />` : null}
        </button>`)}
      </div>
    </div>


    <div class="md-card">
      <div class="card-title"><${Icon} name="monitoring" />自动选择</div>
      <p class="muted">只有当前节点明显变差时才切换；余量越大越保守。修改后在下一次开启或重启自动优化时生效。</p>
      <div class="text-field-grid mt">
        <${Field} label="切换余量 (%)" id="autoMargin" type="number" value=${Math.round(settings.autoMargin * 100)} onChange=${(value) => onSettings({ autoMargin: Math.max(1, Math.min(100, Number(value) || 20)) / 100 })} />
        <${Field} label="连续确认 (轮)" id="autoConfirm" type="number" value=${settings.autoConfirm} onChange=${(value) => onSettings({ autoConfirm: Math.max(1, Number(value) || 2) })} />
        <${Field} label="最低成功率 (%)" id="autoMin" type="number" value=${Math.round(settings.autoMinSuccess * 100)} onChange=${(value) => onSettings({ autoMinSuccess: Math.max(1, Math.min(100, Number(value) || 80)) / 100 })} />
        <${Field} label="检查间隔 (秒)" id="daemonInterval" type="number" value=${settings.daemonInterval} onChange=${(value) => onSettings({ daemonInterval: Math.max(30, Number(value) || 300) })} />
      </div>
    </div>

    <div class="md-card">
      <div class="card-title"><${Icon} name="checkCircle" />环境检查</div>
      <p class="muted">检查控制口、订阅、mihomo 和自动选择组。检查只读，不会修改配置。</p>
      ${health && Array.isArray(health.checks)
        ? html`<div class="health-list">
            ${health.checks.map((check) => html`<div class=${"health-row health-row--" + check.status}>
              <span class="health-dot" aria-hidden="true"></span>
              <div class="health-body">
                <div class="health-head"><strong>${check.label}</strong><span>${check.status === "ok" ? "正常" : check.status === "warn" ? "注意" : "异常"}</span></div>
                ${check.detail ? html`<div class="health-detail">${check.detail}</div>` : null}
                ${check.hint ? html`<div class="health-hint">${check.hint}</div>` : null}
              </div>
            </div>`)}
          </div>`
        : html`<p class="muted mt">${healthBusy ? "正在检查…" : "尚未检查"}</p>`}
      <div class="settings-actions mt">
        <${Button} variant="outlined" icon="refresh" loading=${healthBusy} onClick=${onHealth}>重新检查<//>
      </div>
    </div>

    ${desktop
      ? html`<div class="md-card">
          <div class="card-title"><${Icon} name="monitoring" />后台与通知</div>
          <p class="muted">自动选择在后台运行时，只把需要你知道的变化发成系统通知。修改立即生效。</p>
          <div class="switch-list">
            <${SwitchRow} label="关闭窗口时驻留托盘" checked=${settings.closeToTray} onChange=${(value) => onSettings({ closeToTray: value })} />
            <${SwitchRow} label="切换、回滚时发送系统通知" checked=${settings.notifyOnSwitch} onChange=${(value) => onSettings({ notifyOnSwitch: value })} />
            <${SwitchRow} label="开机自动启动（隐藏到托盘）" checked=${autostart} onChange=${onAutostart} />
          </div>
          <p class="muted">设置文件在覆盖前会自动备份到 ~/.local/share/nodepulse/backups。</p>
        </div>`
      : null}

    <div class="md-card">
      <div class="card-title"><${Icon} name="download" />备份与迁移</div>
      <p class="muted">保存设置前会自动留一份最近 20 次的备份；导出的配置文件在覆盖前也会备份。</p>
      <div class="settings-actions mt">
        <${Button} variant="outlined" icon="download" onClick=${onBackup}>导出设置备份<//>
        <label class="md-button outlined file-button">
          <${Icon} name="refresh" />
          <span>从备份恢复</span>
          <input type="file" accept="application/json,.json" hidden onChange=${async (event) => {
            const file = event.currentTarget.files && event.currentTarget.files[0];
            if (!file) return;
            const text = await file.text();
            event.currentTarget.value = "";
            await onRestore(text);
          }} />
        </label>
      </div>
      <p class="impact-warn mt">备份包含控制口 secret，请妥善保管，不要提交到代码仓库。</p>
    </div>

    <div class="md-card">
      <div class="card-title">连接设置</div>
      <div class="text-field-grid mt">
        <${Field} label="Clash 控制口" id="api" value=${settings.api} placeholder="http://127.0.0.1:9090" onChange=${(value) => onSettings({ api: value })} />
        <${Field} label="Secret" id="secret" type="password" autoComplete="current-password" value=${settings.secret} placeholder="可留空" onChange=${(value) => onSettings({ secret: value })} />
        <${Field} label="订阅配置" id="config" className="span-2" value=${settings.configPath} placeholder="~/sub.yaml" onChange=${(value) => onSettings({ configPath: value })} />
        <${Field} label="Selector 组" id="group" value=${settings.group} placeholder="留空自动选择" onChange=${(value) => onSettings({ group: value })} />
        <${Field} label="mihomo 路径" id="mihomo" value=${settings.mihomoPath} placeholder="/usr/local/bin/mihomo" onChange=${(value) => onSettings({ mihomoPath: value })} />
        <${Field} label="测试 URL" id="url" className="span-2" type="url" inputMode="url" value=${settings.url} placeholder="https://www.gstatic.com/generate_204" onChange=${(value) => onSettings({ url: value })} />
        <${Field} label="并发节点" id="conc" type="number" inputMode="numeric" value=${settings.concurrency} onChange=${(value) => onSettings({ concurrency: Number(value) || 8 })} />
        <${Field} label="单轮超时 (ms)" id="timeout" type="number" inputMode="numeric" value=${settings.timeoutMs} onChange=${(value) => onSettings({ timeoutMs: Number(value) || 5000 })} />
      </div>
      <div class="settings-actions">
        <${Button} variant="filled" icon="check" loading=${busy && busyKind === "save"} onClick=${onSave}>保存设置<//>
        <${Button} variant="tonal" loading=${busy && busyKind === "test"} onClick=${onTest}>测速<//>
        ${busy && busyKind === "test" ? html`<${Button} variant="text" onClick=${onCancel}>取消测速<//>` : null}
        <${Button} variant="tonal" disabled=${!canSwitch || busy} onClick=${onSwitch}>切换选中节点<//>
        <${Button} variant="outlined" disabled=${!canExport || busy} onClick=${() => onExport()}>导出稳定配置<//>
      </div>
      ${busy && busyKind === "test"
        ? html`<p class="status-line mt" role="status" aria-live="polite">测速中 ${progress && progress.total ? `${progress.done || 0}/${progress.total}` : ""}${progress && progress.current ? ` · ${progress.current}` : ""}</p>`
        : null}
      ${exportImpact
        ? html`<div class="export-preview" role="region" aria-label="导出影响预览">
            <div class="export-preview-head">
              <${Icon} name="search" />
              <strong>导出影响</strong>
              <span class="muted">${exportImpact.preserve ? "保留原规则/分组" : "最小配置"}</span>
            </div>
            <div class="impact-grid">
              <div><strong>${exportImpact.included}</strong><span>保留节点</span></div>
              <div><strong>${(exportImpact.excluded || []).length}</strong><span>排除节点</span></div>
              <div><strong>${exportImpact.groups}</strong><span>分组</span></div>
              <div><strong>${exportImpact.rules}</strong><span>规则</span></div>
            </div>
            <div class="impact-flags">
              <span class=${"impact-flag" + (exportImpact.dns ? "" : " dropped")}>DNS ${exportImpact.dns ? "保留" : "丢弃"}</span>
              <span class=${"impact-flag" + (exportImpact.tun ? "" : " dropped")}>TUN ${exportImpact.tun ? "保留" : "丢弃"}</span>
              <span class=${"impact-flag" + (exportImpact.rule_providers ? "" : " dropped")}>规则集 ${exportImpact.rule_providers ? `保留 ${exportImpact.rule_providers} 个` : "无/丢弃"}</span>
            </div>
            ${(exportImpact.excluded || []).length
              ? html`<p class="impact-line">排除示例：${exportImpact.excluded.slice(0, 5).join("、")}${exportImpact.excluded.length > 5 ? ` 等 ${exportImpact.excluded.length} 个` : ""}</p>`
              : null}
            ${(exportImpact.changed_groups || []).length
              ? html`<div class="impact-groups">
                  <div class="impact-line">受影响分组</div>
                  ${exportImpact.changed_groups.slice(0, 6).map((group) => html`<div class="impact-group">
                    <span>${group.name}</span>
                    <span class=${group.fallback ? "impact-warn" : "muted"}>${group.fallback ? "空组 → DIRECT" : `移除 ${group.removed.length} 个`}</span>
                  </div>`)}
                  ${exportImpact.changed_groups.length > 6 ? html`<div class="impact-line muted">还有 ${exportImpact.changed_groups.length - 6} 个分组…</div>` : null}
                </div>`
              : null}
            ${(exportImpact.warnings || []).map((warning) => html`<p class="impact-warn">${warning}</p>`)}
            <${SwitchRow} label="最小配置（替换规则、分组、DNS/TUN）" checked=${exportMinimal} onChange=${onExportMinimal} />
            <div class="export-preview-actions">
              <${Button} variant="filled" icon="download" loading=${busy && busyKind === "export"} onClick=${onConfirmExport}>确认导出<//>
              <${Button} variant="text" onClick=${onCancelExport}>取消<//>
            </div>
          </div>`
        : null}
      <div class="divider"></div>
      <div class="card-title small">控制口状态</div>
      <div class="groups-block">
        ${groups.length
          ? html`<${GroupTable} groups=${groups} />`
          : html`<${State} icon="wifiOff" title="控制口未连接" body="连接 Clash 控制口后，这里会显示组、类型和当前节点。" />`}
      </div>
    </div>
  </section>`;
}

export function History({ events }) {
  return html`<section id="view-history" class="view active">
    <div class="md-card">
      <div class="card-title">自动优化事件</div>
      ${events.length === 0
        ? html`<${State} icon="history" title="暂无事件" body="开启自动优化后，这里会显示切换、回滚和错误记录。" />`
        : html`<ol class="history">
            ${events.map((event, index) => {
              const cur = Number(event.current_score) || 0;
              const best = Number(event.best_score) || 0;
              const scores = cur || best ? ` · 当前 ${cur.toFixed(3)} / 候选 ${best.toFixed(3)}` : "";
              const confidence = event.confidence ? ` · ${confidenceLabel(event.confidence)}置信` : "";
              const rollback = event.rollback_attempted ? (event.rollback_ok ? " · 已回滚" : " · 回滚失败") : "";
              return html`<li class=${"ev" + ((!event.ok || ["rollback", "error"].includes(event.kind)) ? " ev--bad" : "")} key=${`${event.time}-${index}`}>
                <time class="ev-time" datetime=${event.time}>${formatClock(event.time)}</time>
                <span class="ev-kind">${kindLabel(event.kind)}</span>
                <span class="ev-main">
                  ${event.to ? `${event.from || "—"} → ${event.to} · ` : event.from ? `保持 ${event.from} · ` : event.best ? `候选 ${event.best} · ` : ""}
                  ${reasonLabel(event.reason_code || event.reason)}${scores}${confidence}${rollback}
                </span>
              </li>`;
            })}
          </ol>`}
      ${events.length ? html`<p class="muted mt">最近 ${events.length} 条 · ${lastEventLabel(events)}</p>` : null}
    </div>
  </section>`;
}

function lastEventLabel(events) {
  const last = events[0];
  if (!last || !last.time) return "";
  return `最后一条 ${timeAgo(last.time)}`;
}
