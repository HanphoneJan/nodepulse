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
import { formatClock, reasonLabel, timeAgo } from "./format.js";

function Field({ label, id, value, onChange, placeholder, type = "text", inputMode, autoComplete, spellcheck = false, className = "" }) {
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
    />
  </label>`;
}

export function Dashboard({
  settings, status, results, best, selected, busy, busyKind, elapsed, lastTestAt,
  controllers, subscriptions, daemon, daemonInterval, groups, subURL,
  onOneClick, onTest, onSelectNode, onRefresh, onController, onSettings,
  onRounds, onDaemon, onInterval, onSubURL, onImport,
}) {
  const connected = !!status.connected;
  const current = status.current || "—";
  const meta = status.groupName ? `${status.groupName} · ${status.members} 个成员` : "连接客户端后显示";
  const heroLoading = busy && (busyKind === "oneclick" || busyKind === "test");
  const testing = heroLoading;
  const canImport = !!subURL.trim();

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
          <${Button} variant="filled" size="big" icon="play" loading=${busy && busyKind === "oneclick"} onClick=${onOneClick}>
            ${busy && busyKind === "oneclick" ? "优化中…" : "一键优化"}
          <//>
          <${Button} variant="tonal" icon="refresh" loading=${busy && busyKind === "test"} onClick=${onTest}>
            ${busy && busyKind === "test" ? "测速中…" : "只测速"}
          <//>
          <label class="strength">强度
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
          </label>
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
        <div class="card-title"><${Icon} name="link" />连接</div>
        <p class="muted">${controllers.length ? "已自动检测本机 Clash 控制口" : "没检测到控制口。也可以只选订阅，用 mihomo 测速。"}</p>
        <div class="chips" role="group" aria-label="Clash 控制口">
          ${controllers.length === 0
            ? html`<span class="muted">未发现控制口</span>`
            : controllers.map((controller) => {
                const needSecret = controller.auth_required && !settings.secret;
                const label = needSecret ? `${controller.url}（需要 secret）` : controller.url + (controller.version ? ` · ${controller.version}` : "");
                return html`<button
                  type="button"
                  class=${"chip" + (settings.api === controller.url ? " active" : "")}
                  aria-pressed=${settings.api === controller.url ? "true" : "false"}
                  onClick=${() => onController(controller.url)}
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
            options=${[
              { value: "300", label: "5 分钟" },
              { value: "600", label: "10 分钟" },
              { value: "1800", label: "30 分钟" },
            ]}
            onChange=${(value) => onInterval(Number(value))}
          />
        </div>
        <div class="meta-list">
          <div class="meta-item"><${Icon} name="checkCircle" />连续 2 次确认后切换</div>
          <div class="meta-item"><${Icon} name="checkCircle" />切换失败自动回滚</div>
          <div class="meta-item"><${Icon} name="checkCircle" />评分滞后 20% 才行动</div>
        </div>
        <div class="status-line mt" aria-live="polite">${daemon ? "运行中 · 正在按间隔检查" : "未开启"}</div>
      </div>
    </div>

    <div class="md-card">
      <div class="row between">
        <div class="card-title"><${Icon} name="list" />测速结果</div>
        <span class="muted">${best ? `推荐: ${best.name}` : testing ? `已运行 ${elapsed}s` : ""}</span>
      </div>
      ${testing
        ? html`<${LoadingRows} label="正在测速，请稍候" />`
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
  settings, groups, seed, mode, busy, busyKind, lastTestAt,
  onSettings, onSeed, onResetSeed, onMode, onSave, onTest, onSwitch, onExport, canSwitch, canExport,
}) {
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
        <${Button} variant="tonal" disabled=${!canSwitch || busy} onClick=${onSwitch}>切换选中节点<//>
        <${Button} variant="outlined" disabled=${!canExport || busy} onClick=${onExport}>导出稳定配置<//>
      </div>
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
            ${events.map((event, index) => html`<li class="ev" key=${`${event.time}-${index}`}>
              <time class="ev-time" datetime=${event.time}>${formatClock(event.time)}</time>
              <span class="ev-kind">${event.kind}</span>
              <span class="ev-main">
                ${event.from || event.to ? `${event.from || "—"} → ${event.to || "—"} · ` : ""}
                ${reasonLabel(event.reason)}
                ${event.best ? `（推荐 ${event.best}）` : ""}
              </span>
            </li>`)}
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
