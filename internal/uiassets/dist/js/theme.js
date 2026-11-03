import {
  Hct,
  MaterialDynamicColors,
  SchemeContent,
  TonalPalette,
  argbFromHex,
  hexFromArgb,
} from "../material-color-utilities.js";

// FlClash's default accent and presets (lib/common/constant.dart).
export const DEFAULT_SEED = "#007aff";
export const THEME_KEY = "nodepulse.themeMode.v2";
export const SEED_KEY = "nodepulse.seed.v2";
export const SEED_CSS_KEY = "nodepulse.seedCss.v2";
export const THEME_MODES = ["system", "light", "dark"];

export const PRESETS = [
  { name: "蓝", color: "#007aff" },
  { name: "紫", color: "#cb30e0" },
  { name: "粉", color: "#ff2d55" },
  { name: "红", color: "#ff383c" },
  { name: "橙", color: "#ff8d28" },
  { name: "黄", color: "#ffcc00" },
  { name: "绿", color: "#34c759" },
  { name: "灰", color: "#8e8e93" },
];

const LEGACY_DEFAULT = "#0b57d0";
const SURFACE_CHROMA = 3;
const PAPER_CHROMA = 1.5;

const SCHEME_ROLES = {
  primary: "--md-primary",
  onPrimary: "--md-on-primary",
  primaryContainer: "--md-primary-container",
  onPrimaryContainer: "--md-on-primary-container",
  secondary: "--md-secondary",
  onSecondary: "--md-on-secondary",
  secondaryContainer: "--md-secondary-container",
  onSecondaryContainer: "--md-on-secondary-container",
  surface: "--md-surface",
  onSurface: "--md-on-surface",
  onSurfaceVariant: "--md-on-surface-variant",
  outline: "--md-outline",
  outlineVariant: "--md-outline-variant",
  error: "--md-error",
  onError: "--md-on-error",
  inverseSurface: "--md-inverse-surface",
  inverseOnSurface: "--md-inverse-on-surface",
  inversePrimary: "--md-inverse-primary",
};

function storageGet(key) {
  try { return localStorage.getItem(key); } catch (_) { return null; }
}
function storageSet(key, value) {
  try { localStorage.setItem(key, value); } catch (_) {}
}

export function normalizeSeed(seed) {
  return /^#[0-9a-f]{6}$/i.test(seed || "") ? seed.toLowerCase() : DEFAULT_SEED;
}

export function isPreset(seed) {
  return PRESETS.some((p) => p.color === seed);
}

export function seedLabel(seed) {
  const preset = PRESETS.find((p) => p.color === seed);
  return preset ? `${preset.name} · ${seed}` : `自定义 · ${seed}`;
}

export function seedInk(seed) {
  const hex = normalizeSeed(seed).slice(1);
  const rgb = [0, 2, 4].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255);
  const lum = rgb
    .map((v) => (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)))
    .reduce((sum, v, i) => sum + v * [0.2126, 0.7152, 0.0722][i], 0);
  return lum > 0.5 ? "#1b1b1f" : "#ffffff";
}

const clamp = (value, min, max) => Math.min(max, Math.max(min, value));

// Port of FlClash's seededColorScheme: SchemeContent with low-chroma neutral
// surfaces, plus an accent clamped into a readable tone range.
function flclashVars(seed, mode) {
  const dark = mode === "dark";
  const source = Hct.fromInt(argbFromHex(seed));
  const scheme = new SchemeContent(source, dark, 0);
  const vars = {};

  const role = (color) => hexFromArgb(color.getArgb(scheme));
  const neutral = (color, maxChroma, darkTone) => {
    const palette = color.palette(scheme);
    const tone = darkTone == null ? color.getTone(scheme) : darkTone;
    return hexFromArgb(Hct.from(palette.hue, Math.min(palette.chroma, maxChroma), tone).toInt());
  };

  for (const [name, cssVar] of Object.entries(SCHEME_ROLES)) {
    const color = MaterialDynamicColors[name];
    if (color) vars[cssVar] = role(color);
  }

  const tone = dark
    ? clamp(source.tone + 3, 50, 75)
    : clamp(source.tone, 35, 60);
  vars["--md-primary"] = tone === source.tone
    ? seed
    : hexFromArgb(Hct.from(source.hue, source.chroma, tone).toInt());
  vars["--md-on-primary"] = tone <= 60
    ? "#ffffff"
    : hexFromArgb(TonalPalette.fromHueAndChroma(source.hue, source.chroma).tone(10));
  const container = TonalPalette.fromHueAndChroma(
    scheme.primaryPalette.hue,
    Math.min(scheme.primaryPalette.chroma, 48),
  );
  vars["--md-primary-container"] = hexFromArgb(container.tone(dark ? 30 : 90));
  vars["--md-on-primary-container"] = hexFromArgb(container.tone(dark ? 90 : 30));

  vars["--md-surface"] = neutral(MaterialDynamicColors.surface, PAPER_CHROMA);
  vars["--md-surface-dim"] = neutral(MaterialDynamicColors.surfaceDim, SURFACE_CHROMA);
  vars["--md-surface-bright"] = neutral(MaterialDynamicColors.surfaceBright, PAPER_CHROMA);
  vars["--md-surface-container-lowest"] = neutral(MaterialDynamicColors.surfaceContainerLowest, SURFACE_CHROMA);
  vars["--md-surface-container-low"] = neutral(MaterialDynamicColors.surfaceContainerLow, SURFACE_CHROMA);
  vars["--md-surface-container"] = neutral(MaterialDynamicColors.surfaceContainer, SURFACE_CHROMA, dark ? 15 : null);
  vars["--md-surface-container-high"] = neutral(MaterialDynamicColors.surfaceContainerHigh, SURFACE_CHROMA, dark ? 21 : null);
  vars["--md-surface-container-highest"] = neutral(MaterialDynamicColors.surfaceContainerHighest, SURFACE_CHROMA, dark ? 27 : null);

  if (dark) {
    const onSurface = MaterialDynamicColors.onSurface.palette(scheme);
    vars["--md-on-surface"] = hexFromArgb(onSurface.tone(97));
  }
  const outlineVariant = MaterialDynamicColors.outlineVariant.palette(scheme);
  vars["--md-outline-variant"] = hexFromArgb(
    Hct.from(outlineVariant.hue, Math.min(outlineVariant.chroma, 4), MaterialDynamicColors.outlineVariant.getTone(scheme)).toInt(),
  );

  vars["--md-tertiary"] = role(MaterialDynamicColors.tertiary);
  vars["--md-on-tertiary"] = role(MaterialDynamicColors.onTertiary);
  vars["--md-tertiary-container"] = role(MaterialDynamicColors.tertiaryContainer);
  vars["--md-on-tertiary-container"] = role(MaterialDynamicColors.onTertiaryContainer);
  vars["--md-error-container"] = role(MaterialDynamicColors.errorContainer);
  vars["--md-on-error-container"] = role(MaterialDynamicColors.onErrorContainer);
  vars["--md-surface-variant"] = neutral(MaterialDynamicColors.surfaceVariant, SURFACE_CHROMA);

  const success = Hct.fromInt(0xff34c759);
  const warning = Hct.fromInt(0xffff9500);
  vars["--md-success"] = hexFromArgb(Hct.from(success.hue, success.chroma, dark ? 80 : 40).toInt());
  vars["--md-warning"] = hexFromArgb(Hct.from(warning.hue, warning.chroma, dark ? 80 : 40).toInt());
  return vars;
}

export function applySeed(seed) {
  const normalized = normalizeSeed(seed);
  const block = (mode) => {
    const vars = flclashVars(normalized, mode);
    const body = Object.entries(vars)
      .map(([name, value]) => `${name}:${value};`)
      .join("");
    return `html[data-theme="${mode}"]{${body}}`;
  };
  const css = `${block("dark")}\n${block("light")}`;

  let style = document.getElementById("seed-theme");
  if (!style) {
    style = document.createElement("style");
    style.id = "seed-theme";
    document.head.appendChild(style);
  }
  style.textContent = css;
  document.documentElement.dataset.seed = normalized;
  storageSet(SEED_KEY, normalized);
  storageSet(SEED_CSS_KEY, css);

  const dark = document.documentElement.dataset.theme === "dark";
  const vars = flclashVars(normalized, dark ? "dark" : "light");
  let meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) {
    meta = document.createElement("meta");
    meta.name = "theme-color";
    document.head.appendChild(meta);
  }
  meta.content = vars["--md-primary"];
  return normalized;
}

export function currentSeed() {
  const stored = storageGet(SEED_KEY) || storageGet("nodepulse.seed.v1");
  if (!stored || stored === LEGACY_DEFAULT) return DEFAULT_SEED;
  return normalizeSeed(stored);
}

export function currentMode() {
  const mode = storageGet(THEME_KEY) || "system";
  return THEME_MODES.includes(mode) ? mode : "system";
}

function systemDark() {
  return !(window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches);
}

export function effectiveDark(mode) {
  return mode === "dark" || (mode === "system" && systemDark());
}

export function applyMode(mode) {
  const dark = effectiveDark(mode);
  document.documentElement.dataset.theme = dark ? "dark" : "light";
  document.documentElement.dataset.themeMode = mode;
  return dark;
}

export function setMode(mode) {
  const next = THEME_MODES.includes(mode) ? mode : "system";
  storageSet(THEME_KEY, next);
  applyMode(next);
  return next;
}

export function nextMode(mode) {
  return mode === "system" ? "light" : mode === "light" ? "dark" : "system";
}

export function modeLabel(mode) {
  return mode === "system" ? "跟随系统" : mode === "light" ? "亮色" : "暗色";
}

export function bootTheme(onSystemChange) {
  const mode = currentMode();
  applyMode(mode);
  const seed = applySeed(currentSeed());
  const mq = window.matchMedia && window.matchMedia("(prefers-color-scheme: light)");
  if (mq) {
    const handler = () => {
      if (currentMode() === "system") {
        applyMode("system");
        applySeed(currentSeed());
        if (onSystemChange) onSystemChange();
      }
    };
    if (mq.addEventListener) mq.addEventListener("change", handler);
    else if (mq.addListener) mq.addListener(handler);
  }
  return { mode, seed };
}
