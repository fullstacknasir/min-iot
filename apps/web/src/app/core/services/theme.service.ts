import { Injectable, signal, computed } from '@angular/core';

export type AppTheme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'miniot_theme';
export const THEME_DARK: AppTheme = 'dark';
export const THEME_LIGHT: AppTheme = 'light';
export const DARK_CSS_CLASS = 'dark';

@Injectable({
  providedIn: 'root',
})
export class ThemeService {
  public readonly currentTheme = signal<AppTheme>(this.readInitialTheme());
  public readonly isDarkMode = computed(() => this.currentTheme() === THEME_DARK);

  constructor() {
    this.applyThemeToDom(this.currentTheme());
  }

  /**
   * Toggles active theme between light and dark modes.
   */
  public toggleTheme(): void {
    const nextTheme = this.currentTheme() === THEME_DARK ? THEME_LIGHT : THEME_DARK;
    this.setTheme(nextTheme);
  }

  /**
   * Explicitly sets and persists the requested theme.
   */
  public setTheme(theme: AppTheme): void {
    this.currentTheme.set(theme);
    this.persistTheme(theme);
    this.applyThemeToDom(theme);
  }

  private readInitialTheme(): AppTheme {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return THEME_DARK;
    const stored = localStorage.getItem(THEME_STORAGE_KEY) as AppTheme | null;
    if (stored === THEME_DARK || stored === THEME_LIGHT) return stored;
    return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? THEME_DARK : THEME_LIGHT;
  }

  private persistTheme(theme: AppTheme): void {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(THEME_STORAGE_KEY, theme);
    } catch {
      // storage unavailable
    }
  }

  private applyThemeToDom(theme: AppTheme): void {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    if (theme === THEME_DARK) {
      root.classList.add(DARK_CSS_CLASS);
      root.setAttribute('data-theme', THEME_DARK);
    } else {
      root.classList.remove(DARK_CSS_CLASS);
      root.setAttribute('data-theme', THEME_LIGHT);
    }
  }
}
