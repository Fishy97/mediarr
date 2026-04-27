import type { AppearanceSettings } from '../types';

export type ResolvedTheme = 'dark' | 'light';

export function resolveTheme(theme: AppearanceSettings['theme'], prefersLight: boolean): ResolvedTheme {
  if (theme === 'light' || theme === 'dark') {
    return theme;
  }
  return prefersLight ? 'light' : 'dark';
}

export function applyAppearanceSettings(documentRef: Document, appearance: AppearanceSettings, prefersLight: boolean): void {
  const resolved = resolveTheme(appearance.theme, prefersLight);
  documentRef.documentElement.dataset.theme = resolved;
  documentRef.documentElement.dataset.themePreference = appearance.theme;
  let style = documentRef.getElementById('mediarr-custom-css');
  if (!style) {
    style = documentRef.createElement('style');
    style.id = 'mediarr-custom-css';
    documentRef.head.appendChild(style);
  }
  style.textContent = appearance.customCss;
}
