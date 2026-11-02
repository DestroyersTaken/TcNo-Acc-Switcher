import { writable } from "svelte/store";
import { DEFAULT_THEME_ID } from "./types";

export const currentThemeId = writable<string>(DEFAULT_THEME_ID);
export const currentThemeBgUrl = writable<string>("");
export const currentThemeAccentKey = writable<string>("");
export const currentThemeCustomAccentColor = writable<string>("");
export const currentWindowsThemeAccentColor = writable<string>("");

// Mirrors the active theme onto <html data-theme="...">. The modern default look
// (styles/modern.scss) is scoped to `[data-theme="default"]`, so every other
// theme keeps rendering exactly as it was written.
if (typeof document !== "undefined") {
  currentThemeId.subscribe((id) => {
    const root = document.documentElement;
    if (root?.dataset) root.dataset.theme = id || DEFAULT_THEME_ID;
  });
}
