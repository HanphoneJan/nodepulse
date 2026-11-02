import {
  themeFromSourceColor,
  argbFromHex,
  hexFromArgb,
} from "./material-color-utilities.js";

const DEFAULT_SEED = "#0b57d0";
const STORAGE_KEY = "nodepulse.seed.v1";

const PRESETS = [
  { name: "蓝", color: "#0b57d0" },
  { name: "紫", color: "#6750a4" },
  { name: "青", color: "#006a6a" },
  { name: "绿", color: "#4a6741" },
  { name: "橙", color: "#8a5a00" },
  { name: "红", color: "#b3261e" },
  { name: "粉", color: "#7d5260" },
];

const ROLE_MAP = {
  primary: "--md-primary",
  onPrimary: "--md-on-primary",
  primaryContainer: "--md-primary-container",
  onPrimaryContainer: "--md-on-primary-container",
  secondary: "--md-secondary",
  onSecondary: "--md-on-secondary",
  secondaryContainer: "--md-secondary-container",
  onSecondaryContainer: "--md-on-secondary-container",
  surface: "--md-surface",
  surfaceDim: "--md-surface-dim",
  surfaceContainerLow: "--md-surface-container-low",
  surfaceContainer: "--md-surface-container",
  surfaceContainerHigh: "--md-surface-container-high",
  surfaceContainerHighest: "--md-surface-container-highest",
  onSurface: "--md-on-surface",
  onSurfaceVariant: "--md-on-surface-variant",
  outline: "--md-outline",
  outlineVariant: "--md-outline-variant",
  error: "--md-error",
  onError: "--md-on-error",
};

function schemeVars(scheme) {
  return Object.entries(ROLE_MAP)
    .filter(([role]) => typeof scheme[role] === "number")
    .map(([role, cssVar]) => `${cssVar}: ${hexFromArgb(scheme[role])};`)
    .join(" ");
}

function normalizeSeed(seed) {
  return /^#[0-9a-f]{6}$/i.test(seed || "") ? seed.toLowerCase() : DEFAULT_SEED;
}

function apply(seed) {
  const normalized = normalizeSeed(seed);
  let theme;
  try {
    theme = themeFromSourceColor(argbFromHex(normalized));
  } catch (_) {
    theme = themeFromSourceColor(argbFromHex(DEFAULT_SEED));
  }
  const css = [
    `html[data-theme="dark"] { ${schemeVars(theme.schemes.dark)} }`,
    `html[data-theme="light"] { ${schemeVars(theme.schemes.light)} }`,
  ].join("\n");
  let style = document.getElementById("seed-theme");
  if (!style) {
    style = document.createElement("style");
    style.id = "seed-theme";
    document.head.appendChild(style);
  }
  style.textContent = css;
  document.documentElement.dataset.seed = normalized;

  // Keep the browser/WebView chrome in sync with the current scheme.
  const isDark = document.documentElement.dataset.theme === "dark";
  const scheme = isDark ? theme.schemes.dark : theme.schemes.light;
  let meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) {
    meta = document.createElement("meta");
    meta.name = "theme-color";
    document.head.appendChild(meta);
  }
  meta.content = hexFromArgb(scheme.primary);
  return normalized;
}

window.NPSeedTheme = { apply, DEFAULT_SEED, STORAGE_KEY, PRESETS };
