import {
  themeFromSourceColor,
  argbFromHex,
  hexFromArgb,
} from "../material-color-utilities.js";

export const DEFAULT_SEED = "#0b57d0";
export const THEME_KEY = "nodepulse.themeMode.v2";
export const SEED_KEY = "nodepulse.seed.v1";
export const SEED_CSS_KEY = "nodepulse.seedCss.v1";
export const THEME_MODES = ["system", "light", "dark"];

export const PRESETS = [
  { name: "蓝", color: "#0b57d0" },
  { name: "紫", color: "#6750a4" },
  { name: "青", color: "#006a6a" },
  { name: "绿", color: "#4a6741" },
  { name: "橙", color: "#8a5a00" },
  { name: "红", color: "#b3261e" },
  { name: "粉", color: "#7d5260" },
];

const SURFACE_TONES = {
  light: {
    surfaceDim: 87,
    surfaceBright: 98,
    surfaceContainerLowest: 100,
    surfaceContainerLow: 96,
    surfaceContainer: 94,
    surfaceContainerHigh: 92,
    surfaceContainerHighest: 90,
  },
  dark: {
    surfaceDim: 6,
    surfaceBright: 24,
    surfaceContainerLowest: 4,
    surfaceContainerLow: 10,
    surfaceContainer: 12,
    surfaceContainerHigh: 17,
    surfaceContainerHighest: 22,
  },
};

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

function schemeVars(theme, mode) {
  const dark = mode === "dark";
  const scheme = theme.schemes[mode];
  const neutral = theme.palettes.neutral;
  const neutralVariant = theme.palettes.neutralVariant;
  const tertiary = theme.palettes.tertiary;
  const error = theme.palettes.error;
  const vars = {};

  for (const [role, cssVar] of Object.entries(SCHEME_ROLES)) {
    if (typeof scheme[role] === "number") vars[cssVar] = hexFromArgb(scheme[role]);
  }
  for (const [role, tone] of Object.entries(SURFACE_TONES[mode])) {
    const cssVar = "--md-" + role.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
    vars[cssVar] = hexFromArgb(neutral.tone(tone));
  }
  vars["--md-surface-variant"] = hexFromArgb(neutralVariant.tone(dark ? 30 : 90));
  vars["--md-tertiary"] = hexFromArgb(tertiary.tone(dark ? 80 : 40));
  vars["--md-on-tertiary"] = hexFromArgb(tertiary.tone(dark ? 20 : 100));
  vars["--md-tertiary-container"] = hexFromArgb(tertiary.tone(dark ? 30 : 90));
  vars["--md-on-tertiary-container"] = hexFromArgb(tertiary.tone(dark ? 90 : 10));
  vars["--md-error-container"] = hexFromArgb(error.tone(dark ? 30 : 90));
  vars["--md-on-error-container"] = hexFromArgb(error.tone(dark ? 90 : 10));
  return vars;
}

export function applySeed(seed) {
  const normalized = normalizeSeed(seed);
  let theme;
  try {
    theme = themeFromSourceColor(argbFromHex(normalized));
  } catch (_) {
    theme = themeFromSourceColor(argbFromHex(DEFAULT_SEED));
  }

  const block = (mode) => {
    const vars = schemeVars(theme, mode);
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
  const scheme = dark ? theme.schemes.dark : theme.schemes.light;
  let meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) {
    meta = document.createElement("meta");
    meta.name = "theme-color";
    document.head.appendChild(meta);
  }
  meta.content = hexFromArgb(scheme.primary);
  return normalized;
}

export function currentSeed() {
  return normalizeSeed(storageGet(SEED_KEY) || DEFAULT_SEED);
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
