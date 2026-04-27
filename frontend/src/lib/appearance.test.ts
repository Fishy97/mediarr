import { describe, expect, test } from 'vitest';
import { applyAppearanceSettings, nextThemePreference, resolveTheme } from './appearance';

describe('appearance helpers', () => {
  test('resolves system theme against the current browser preference', () => {
    expect(resolveTheme('system', true)).toBe('light');
    expect(resolveTheme('system', false)).toBe('dark');
    expect(resolveTheme('dark', true)).toBe('dark');
    expect(resolveTheme('light', false)).toBe('light');
  });

  test('applies theme and custom css to the document shell', () => {
    const doc = document.implementation.createHTMLDocument('Mediarr');

    applyAppearanceSettings(doc, { theme: 'light', customCss: '.panel { border-radius: 6px; }' }, false);

    expect(doc.documentElement.dataset.theme).toBe('light');
    expect(doc.documentElement.dataset.themePreference).toBe('light');
    expect(doc.getElementById('mediarr-custom-css')?.textContent).toBe('.panel { border-radius: 6px; }');

    applyAppearanceSettings(doc, { theme: 'system', customCss: '' }, true);

    expect(doc.documentElement.dataset.theme).toBe('light');
    expect(doc.documentElement.dataset.themePreference).toBe('system');
    expect(doc.getElementById('mediarr-custom-css')?.textContent).toBe('');
  });

  test('quick toggle persists the opposite explicit theme', () => {
    expect(nextThemePreference('dark', false)).toBe('light');
    expect(nextThemePreference('light', true)).toBe('dark');
    expect(nextThemePreference('system', false)).toBe('light');
    expect(nextThemePreference('system', true)).toBe('dark');
  });
});
