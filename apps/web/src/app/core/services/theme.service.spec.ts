import { TestBed } from '@angular/core/testing';
import { ThemeService, THEME_DARK, THEME_LIGHT } from './theme.service';

describe('ThemeService', () => {
  let service: ThemeService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ThemeService],
    });
    service = TestBed.inject(ThemeService);
  });

  it('should initialize with a theme', () => {
    expect(service.currentTheme()).toBeDefined();
    expect([THEME_DARK, THEME_LIGHT]).toContain(service.currentTheme());
  });

  it('should toggle theme from dark to light and vice versa', () => {
    service.setTheme(THEME_DARK);
    expect(service.isDarkMode()).toBe(true);

    service.toggleTheme();
    expect(service.currentTheme()).toBe(THEME_LIGHT);
    expect(service.isDarkMode()).toBe(false);

    service.toggleTheme();
    expect(service.currentTheme()).toBe(THEME_DARK);
    expect(service.isDarkMode()).toBe(true);
  });
});
