import { html } from "./html.js";
import { useEffect, useId, useRef, useState } from "../vendor/hooks.module.js";
import { PRESETS, isPreset, seedInk, seedLabel } from "./theme.js";
import { baseName, hostOf } from "./format.js";

export function Icon({ name, className = "" }) {
  return html`<svg class="icon ${className}" aria-hidden="true" focusable="false"><use href="icons.svg#i-${name}"></use></svg>`;
}

export function Button({ children, variant = "filled", size = "", icon, loading = false, disabled = false, onClick, title }) {
  const cls = ["md-button", variant, size, loading ? "loading" : ""].filter(Boolean).join(" ");
  return html`<button
    type="button"
    class=${cls}
    title=${title}
    disabled=${disabled || loading}
    aria-busy=${loading ? "true" : undefined}
    onClick=${onClick}
  >
    ${loading ? html`<span class="spinner" aria-hidden="true"></span>` : icon ? html`<${Icon} name=${icon} />` : null}
    <span>${children}</span>
  </button>`;
}

export function IconButton({ icon, label, onClick, title }) {
  return html`<button type="button" class="icon-button" aria-label=${label} title=${title || label} onClick=${onClick}>
    <${Icon} name=${icon} />
  </button>`;
}

export function Snackbar({ snack, onClose }) {
  if (!snack) return null;
  const error = snack.kind === "error";
  return html`<div class=${"snackbar show" + (error ? " error" : "")} role=${error ? "alert" : "status"} aria-live=${error ? "assertive" : "polite"} aria-atomic="true">
    <span class="snackbar-message">${snack.text}</span>
    <button type="button" class="snackbar-close" aria-label="关闭提示" onClick=${onClose}>
      <${Icon} name="close" />
    </button>
  </div>`;
}

export function M3Select({ label, value, options, onChange, className = "", disabled = false }) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const wrapRef = useRef(null);
  const buttonRef = useRef(null);
  const menuRef = useRef(null);
  const id = useId();
  const listId = `select-${id}`;
  const selectedIndex = Math.max(0, options.findIndex((o) => o.value === value));
  const selected = options[selectedIndex];

  useEffect(() => {
    if (open) setActive(selectedIndex);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    menuRef.current?.querySelector(".m3-option.active")?.scrollIntoView({ block: "nearest" });
  }, [active, open]);

  const close = (restoreFocus = true) => {
    setOpen(false);
    if (restoreFocus) buttonRef.current?.focus();
  };

  const choose = (index) => {
    const option = options[index];
    if (!option || option.disabled) return;
    onChange(option.value);
    close();
  };

  const move = (delta) => {
    if (!options.length) return;
    if (!open) {
      setOpen(true);
      setActive(selectedIndex);
      return;
    }
    setActive((index) => (index + delta + options.length) % options.length);
  };

  useEffect(() => {
    if (!open) return undefined;

    const update = () => {
      const F = window.FloatingUIDOM;
      const button = buttonRef.current;
      const menu = menuRef.current;
      if (!button || !menu) return;
      menu.style.minWidth = `${button.getBoundingClientRect().width}px`;
      if (!F || !F.computePosition) {
        const rect = button.getBoundingClientRect();
        menu.style.left = `${Math.max(8, Math.min(rect.left, window.innerWidth - menu.offsetWidth - 8))}px`;
        menu.style.top = `${rect.bottom + 4}px`;
        return;
      }
      const middleware = [
        F.offset(4),
        F.flip({ padding: 8 }),
        F.shift({ padding: 8 }),
        F.size({
          padding: 8,
          apply({ availableHeight }) {
            menu.style.maxHeight = `${Math.min(280, availableHeight)}px`;
          },
        }),
      ];
      F.computePosition(button, menu, {
        placement: "bottom-start",
        strategy: "fixed",
        middleware,
      }).then(({ x, y }) => {
        menu.style.left = `${x}px`;
        menu.style.top = `${y}px`;
      });
    };

    update();
    let cleanup = null;
    const F = window.FloatingUIDOM;
    if (F && F.autoUpdate && buttonRef.current && menuRef.current) {
      cleanup = F.autoUpdate(buttonRef.current, menuRef.current, update);
    } else {
      window.addEventListener("resize", update);
      window.addEventListener("scroll", update, true);
    }

    const onPointerDown = (event) => {
      if (!wrapRef.current?.contains(event.target)) close(false);
    };
    document.addEventListener("pointerdown", onPointerDown);

    return () => {
      if (cleanup) cleanup();
      else {
        window.removeEventListener("resize", update);
        window.removeEventListener("scroll", update, true);
      }
      document.removeEventListener("pointerdown", onPointerDown);
    };
  }, [open]);

  const onKeyDown = (event) => {
    const key = event.key;
    if (key === "Escape") {
      if (open) {
        event.preventDefault();
        close();
      }
      return;
    }
    if (key === "Tab") {
      close(false);
      return;
    }
    if (key === "ArrowDown") {
      event.preventDefault();
      move(1);
      return;
    }
    if (key === "ArrowUp") {
      event.preventDefault();
      move(-1);
      return;
    }
    if (key === "Home" && open) {
      event.preventDefault();
      setActive(0);
      return;
    }
    if (key === "End" && open) {
      event.preventDefault();
      setActive(options.length - 1);
      return;
    }
    if (key === "Enter" || key === " ") {
      event.preventDefault();
      if (open) choose(active);
      else setOpen(true);
      return;
    }
    if (key.length === 1 && key.trim()) {
      const start = open ? active + 1 : 0;
      const found = options.findIndex((option, index) => index >= start && option.label.toLowerCase().startsWith(key.toLowerCase()));
      const wrapFound = found >= 0 ? found : options.findIndex((option) => option.label.toLowerCase().startsWith(key.toLowerCase()));
      if (wrapFound >= 0) {
        setOpen(true);
        setActive(wrapFound);
      }
    }
  };

  return html`<div class=${"m3-select " + className} ref=${wrapRef}>
    <button
      ref=${buttonRef}
      type="button"
      class="m3-select-button"
      role="combobox"
      aria-haspopup="listbox"
      aria-expanded=${open ? "true" : "false"}
      aria-controls=${listId}
      aria-activedescendant=${open && options[active] ? `${listId}-${active}` : undefined}
      aria-label=${label}
      disabled=${disabled}
      onClick=${() => (open ? close() : setOpen(true))}
      onKeyDown=${onKeyDown}
    >
      <span class="m3-select-label">${selected ? selected.label : label}</span>
      <${Icon} name="chevron" />
    </button>
    ${open && html`<div ref=${menuRef} id=${listId} class="m3-select-menu" role="listbox" aria-label=${label}>
      ${options.length === 0
        ? html`<div class="m3-option" aria-disabled="true">没有选项</div>`
        : options.map((option, index) => html`<div
            id=${`${listId}-${index}`}
            class=${"m3-option" + (index === active ? " active" : "")}
            role="option"
            aria-selected=${option.value === value ? "true" : "false"}
            aria-disabled=${option.disabled ? "true" : undefined}
            onMouseEnter=${() => setActive(index)}
            onClick=${() => choose(index)}
          >
            <span class="m3-option-label">${option.label}</span>
            <${Icon} name="check" />
          </div>`)}
    </div>`}
  </div>`;
}

export function SwitchRow({ label, checked, onChange, disabled = false }) {
  return html`<label class="switch-row">
    <span>${label}</span>
    <span class="switch">
      <input type="checkbox" role="switch" checked=${checked} disabled=${disabled} onChange=${(event) => onChange(event.currentTarget.checked)} />
      <span class="track"><span class="thumb"></span></span>
    </span>
  </label>`;
}

export function Segmented({ label, value, options, onChange }) {
  const name = `seg-${useId()}`;
  return html`<div class="segmented" role="radiogroup" aria-label=${label}>
    ${options.map((option) => html`<label class=${"seg-option" + (option.value === value ? " active" : "")}>
      <input
        type="radio"
        name=${name}
        value=${option.value}
        aria-label=${option.label}
        checked=${option.value === value}
        onChange=${() => onChange(option.value)}
      />
      <span class="seg-content">
        ${option.icon ? html`<${Icon} name=${option.icon} />` : null}
        <span>${option.label}</span>
      </span>
    </label>`)}
  </div>`;
}

export function SeedPicker({ seed, onSeed, onReset }) {
  return html`<div class="seed-block">
    <span class="field-label">主题色</span>
    <div class="seed-controls">
      <div class="seed-presets" role="group" aria-label="主题色预设">
        ${PRESETS.map((preset) => html`<button
          type="button"
          class=${"seed-chip" + (preset.color === seed ? " active" : "")}
          style=${`--seed:${preset.color};--seed-ink:${seedInk(preset.color)}`}
          aria-pressed=${preset.color === seed ? "true" : "false"}
          aria-label=${`主题色 ${preset.name}`}
          title=${`${preset.name} · ${preset.color}`}
          onClick=${() => onSeed(preset.color)}
        >
          <${Icon} name="check" />
        </button>`)}
      </div>
      <label class="seed-custom" title="选择自定义主题色">
        <input type="color" value=${seed} aria-label="自定义主题色" onInput=${(event) => onSeed(event.currentTarget.value)} />
        <${Icon} name="palette" />
        <span>${isPreset(seed) ? "自定义…" : "自定义"}</span>
        ${!isPreset(seed) ? html`<span class="seed-custom-dot" style=${`--seed-current:${seed}`}></span>` : null}
      </label>
      <button type="button" class="md-button text small" onClick=${onReset}>恢复默认</button>
    </div>
    <p class="seed-status" aria-live="polite">当前：${seedLabel(seed)}</p>
  </div>`;
}

export function ProfileMenu({ profiles, activeId, onSelect, onManage }) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef(null);
  const buttonRef = useRef(null);
  const menuRef = useRef(null);
  const active = profiles.find((profile) => profile.id === activeId) || profiles[0];

  const close = (restoreFocus = true) => {
    setOpen(false);
    if (restoreFocus) buttonRef.current?.focus();
  };

  useEffect(() => {
    if (!open) return undefined;
    const update = () => {
      const button = buttonRef.current;
      const menu = menuRef.current;
      if (!button || !menu) return;
      const rect = button.getBoundingClientRect();
      const width = Math.min(320, window.innerWidth - 16);
      const height = menu.offsetHeight;
      menu.style.width = `${width}px`;
      menu.style.left = `${Math.max(8, Math.min(rect.right - width, window.innerWidth - width - 8))}px`;
      const below = rect.bottom + 4;
      menu.style.top = `${below + height > window.innerHeight - 8 ? Math.max(8, rect.top - height - 4) : below}px`;
    };
    update();
    const onPointerDown = (event) => {
      if (!wrapRef.current?.contains(event.target)) close(false);
    };
    const onKeyDown = (event) => {
      if (event.key === "Escape") {
        event.preventDefault();
        close();
      }
    };
    const onFocusIn = (event) => {
      if (!wrapRef.current?.contains(event.target)) close(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("focusin", onFocusIn);
    window.addEventListener("resize", update);
    window.addEventListener("scroll", update, true);
    const timer = setTimeout(() => {
      const target = menuRef.current?.querySelector(".profile-item.active") || menuRef.current?.querySelector("button");
      target?.focus();
    }, 0);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("focusin", onFocusIn);
      window.removeEventListener("resize", update);
      window.removeEventListener("scroll", update, true);
    };
  }, [open]);

  return html`<div class="profile-switch" ref=${wrapRef}>
    <button
      ref=${buttonRef}
      type="button"
      class="env-button"
      aria-haspopup="menu"
      aria-expanded=${open ? "true" : "false"}
      aria-label=${`快速切换环境：${active ? active.name : "默认环境"}`}
      title=${`快速切换环境：${active ? active.name : "默认环境"}`}
      onClick=${() => (open ? close() : setOpen(true))}
    >
      <${Icon} name="swap" />
      <span class="env-name">${active ? active.name : "默认环境"}</span>
    </button>
    ${open && html`<div
      ref=${menuRef}
      class="profile-menu"
      role="menu"
      aria-label="快速切换环境"
      onKeyDown=${(event) => {
        const items = [...(menuRef.current?.querySelectorAll(".profile-item, .profile-manage") || [])];
        const index = items.indexOf(document.activeElement);
        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          event.preventDefault();
          const delta = event.key === "ArrowDown" ? 1 : -1;
          const next = (index + delta + items.length) % items.length;
          items[next]?.focus();
        } else if (event.key === "Home" || event.key === "End") {
          event.preventDefault();
          items[event.key === "Home" ? 0 : items.length - 1]?.focus();
        }
      }}
    >
      <div class="profile-menu-head">快速切换环境</div>
      ${profiles.length === 0
        ? html`<div class="profile-empty">还没有保存的环境</div>`
        : profiles.map((profile) => html`<button
            key=${profile.id}
            type="button"
            role="menuitem"
            class=${"profile-item" + (profile.id === (active && active.id) ? " active" : "")}
            aria-current=${profile.id === (active && active.id) ? "true" : undefined}
            onClick=${() => { onSelect(profile.id); close(); }}
          >
            <span class="profile-item-body">
              <strong>${profile.name}</strong>
              <small>${hostOf(profile.api)} · ${baseName(profile.configPath)}</small>
            </span>
            ${profile.id === (active && active.id) ? html`<${Icon} name="check" />` : null}
          </button>`)}
      <div class="profile-menu-foot">
        <button type="button" role="menuitem" class="profile-manage" onClick=${() => { onManage(); close(); }}>管理环境…</button>
      </div>
    </div>`}
  </div>`;
}

export function State({ kind = "empty", icon = "monitoring", title, body, actions }) {
  return html`<div class=${"state state--" + kind}>
    <${Icon} name=${icon} className="state-icon" />
    ${title ? html`<p class="state-title">${title}</p>` : null}
    ${body ? html`<p class="state-body">${body}</p>` : null}
    ${actions ? html`<div class="state-actions">${actions}</div>` : null}
  </div>`;
}

export function LoadingRows({ label = "正在加载…" }) {
  return html`<div class="skeleton-list" role="status" aria-label=${label} aria-busy="true">
    <div class="skeleton-row"></div>
    <div class="skeleton-row"></div>
    <div class="skeleton-row"></div>
    <span class="sr-only">${label}</span>
  </div>`;
}

export function ScoreRing({ value, bestName }) {
  const ratio = Math.max(0, Math.min(1, Number(value) || 0));
  const offset = 276.46 * (1 - ratio);
  return html`<div class="hero-score">
    <div class="score-kicker">推荐节点</div>
    <div class="score-ring">
      <svg viewBox="0 0 100 100" aria-hidden="true">
        <circle class="ring-track" cx="50" cy="50" r="44"></circle>
        <circle class="ring-value" cx="50" cy="50" r="44" style=${`stroke-dashoffset:${offset}`}></circle>
      </svg>
      <div class="score-text"><span>${Math.round(ratio * 100)}</span><small>评分</small></div>
    </div>
    <div class="best-name" title=${bestName}>${bestName}</div>
  </div>`;
}

export function HeroPending({ members, rounds, concurrency, lastTestAt }) {
  return html`<div class="hero-pending">
    <div class="pending-kicker"><span class="pending-dot"></span>待测速</div>
    <div class="pending-figure"><strong>${members || "—"}</strong><span>个节点</span></div>
    <div class="pending-meta">${rounds} 轮 · 并发 ${concurrency}</div>
    <div class="pending-last">${lastTestAt ? `上次 ${lastTestAt}` : "尚未测速"}</div>
  </div>`;
}
