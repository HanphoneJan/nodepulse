import { html } from "./html.js";
import { Icon } from "./components.js";
import { latency, score } from "./format.js";

export function NodeTable({ rows, best, selected, onSelect, label }) {
  const focusIndex = Math.max(0, rows.findIndex((row) => row.name === selected));

  const keyDown = (event, row) => {
    const key = event.key;
    if (key !== "ArrowDown" && key !== "ArrowUp" && key !== "Home" && key !== "End") {
      if (key === "Enter" || key === " ") {
        event.preventDefault();
        onSelect(row.dataset.name);
      }
      return;
    }
    event.preventDefault();
    const options = Array.from(row.parentElement.querySelectorAll('[role="option"]'));
    const index = options.indexOf(row);
    let next = index;
    if (key === "ArrowDown") next = Math.min(options.length - 1, index + 1);
    if (key === "ArrowUp") next = Math.max(0, index - 1);
    if (key === "Home") next = 0;
    if (key === "End") next = options.length - 1;
    const target = options[next];
    if (target) {
      target.focus();
      onSelect(target.dataset.name);
    }
  };

  return html`<div class="data-table data-table--nodes" role="listbox" aria-label=${label}>
    <div class="node-head" aria-hidden="true">
      <div class="col-rank">#</div>
      <div class="col-name">节点</div>
      <div class="col-type">类型</div>
      <div class="col-ok node-num">OK</div>
      <div class="col-p50 node-num">P50</div>
      <div class="col-p90 node-num">P90</div>
      <div class="col-jitter node-num">抖动</div>
      <div class="col-score node-score">评分</div>
    </div>
    ${rows.map((row, index) => {
      const dead = row.success === 0;
      const okClass = row.success === row.total ? "ok" : row.success > 0 ? "warn" : "err";
      const latClass = dead ? "err" : row.p90_ms <= 200 ? "ok" : row.p90_ms <= 500 ? "warn" : "err";
      const isBest = !!best && row.name === best.name;
      const isSelected = row.name === selected;
      return html`<div
        key=${row.name}
        role="option"
        aria-selected=${isSelected ? "true" : "false"}
        tabindex=${index === focusIndex ? "0" : "-1"}
        data-name=${row.name}
        class=${"node-row" + (isBest ? " best" : "") + (isSelected ? " selected" : "")}
        onClick=${() => onSelect(row.name)}
        onKeyDown=${(event) => keyDown(event, event.currentTarget)}
      >
        <div class="col-rank node-rank">${index + 1}</div>
        <div class="col-name node-name-wrap">
          <span class="node-name" title=${row.name}>${row.name}</span>
          ${row.source ? html`<span class="node-source" title=${row.source}>${row.source}</span>` : null}
          ${isBest ? html`<${Icon} name="checkCircle" className="best-icon" /><span class="sr-only">推荐</span>` : null}
        </div>
        <div class="col-type node-type">${row.type}</div>
        <div class=${`col-ok node-num ${okClass}`}>${row.success}/${row.total}</div>
        <div class="col-p50 node-num">${dead ? "—" : latency(row.median_ms)}</div>
        <div class=${`col-p90 node-num ${latClass}`}>${dead ? "—" : latency(row.p90_ms)}</div>
        <div class="col-jitter node-num">${dead ? "—" : latency(row.jitter_ms)}</div>
        <div class="col-score node-score">${score(row.score)}</div>
      </div>`;
    })}
  </div>`;
}

export function GroupTable({ groups }) {
  return html`<table class="group-table">
    <caption class="sr-only">控制口状态</caption>
    <thead>
      <tr>
        <th class="col-rank" scope="col">#</th>
        <th scope="col">组</th>
        <th class="col-type" scope="col">类型</th>
        <th class="col-current num" scope="col">当前</th>
      </tr>
    </thead>
    <tbody>
      ${groups.map((group, index) => html`<tr key=${group.name}>
        <td class="col-rank">${index + 1}</td>
        <td title=${group.name}><span class="node-name">${group.name}</span></td>
        <td class="col-type node-type">${group.type}</td>
        <td class="col-current num" title=${`${group.now || "—"} · ${group.members} 个成员`}>${group.now || "—"} · ${group.members}</td>
      </tr>`)}
    </tbody>
  </table>`;
}
