import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { Router } from '@angular/router';
import { AppModule } from './app.module';
import { AppComponent } from './app.component';
import { ThemeService, THEME_LIGHT, THEME_DARK } from './core/services/theme.service';

describe('AppComponent', () => {
  let router: Router;
  let themeService: ThemeService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideNoopAnimations(),
      ],
    }).compileComponents();

    router = TestBed.inject(Router);
    themeService = TestBed.inject(ThemeService);
    vi.spyOn(router, 'navigate').mockImplementation(() => Promise.resolve(true));
  });

  it('should create the minIoT application instance', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it('should toggle sidebar state between collapsed and expanded', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;

    expect(app.isSidebarCollapsed()).toBe(false);
    app.toggleSidebar();
    expect(app.isSidebarCollapsed()).toBe(true);
    app.toggleSidebar();
    expect(app.isSidebarCollapsed()).toBe(false);
  });

  it('should toggle user menu and close on escape', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;

    expect(app.isUserMenuOpen()).toBe(false);
    app.toggleUserMenu();
    expect(app.isUserMenuOpen()).toBe(true);

    app.handleEscape();
    expect(app.isUserMenuOpen()).toBe(false);
  });

  it('should toggle theme using ThemeService', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;

    themeService.setTheme(THEME_DARK);
    app.toggleTheme();
    expect(themeService.currentTheme()).toBe(THEME_LIGHT);
  });

  it('should navigate to profile from submenu', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;

    app.toggleUserMenu();
    app.navigateToProfile();

    expect(app.isUserMenuOpen()).toBe(false);
    expect(router.navigate).toHaveBeenCalledWith(['/profile']);
  });
});
