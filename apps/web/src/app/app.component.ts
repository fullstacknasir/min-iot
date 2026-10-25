import { Component, ElementRef, HostListener, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { AuthService } from './core/services/auth.service';
import { ThemeService } from './core/services/theme.service';
import { NotificationService } from './core/services/notification.service';
import { Notification as AppNotification } from './core/models/rule-alert.model';

const MOBILE_BREAKPOINT_PX = 768;
const MAX_RECENT_NOTIFICATIONS = 5;

@Component({
  selector: 'app-root',
  standalone: false,
  templateUrl: './app.component.html',
})
export class AppComponent implements OnInit, OnDestroy {
  public readonly authService = inject(AuthService);
  public readonly themeService = inject(ThemeService);
  private readonly notificationService = inject(NotificationService);
  private readonly router = inject(Router);
  private readonly elementRef = inject(ElementRef);

  public readonly isSidebarCollapsed = signal<boolean>(false);
  public readonly isMobileSidebarOpen = signal<boolean>(false);
  public readonly isUserMenuOpen = signal<boolean>(false);
  public readonly isNotificationMenuOpen = signal<boolean>(false);
  public readonly windowInnerWidth = signal<number>(typeof window !== 'undefined' ? window.innerWidth : 1200);

  public readonly unreadCount = signal<number>(0);
  public readonly recentNotifications = signal<AppNotification[]>([]);

  private liveNotificationSub?: Subscription;

  public ngOnInit(): void {
    if (this.authService.isAuthenticated()) {
      this.initNotificationFeed();
    }
  }

  public ngOnDestroy(): void {
    this.liveNotificationSub?.unsubscribe();
  }

  @HostListener('window:resize')
  public handleWindowResize(): void {
    if (typeof window === 'undefined') return;
    this.windowInnerWidth.set(window.innerWidth);
    if (window.innerWidth >= MOBILE_BREAKPOINT_PX && this.isMobileSidebarOpen()) {
      this.isMobileSidebarOpen.set(false);
    }
  }

  @HostListener('document:click', ['$event'])
  public handleDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement | null;
    this.closeMenusOnOutsideClick(target);
  }

  @HostListener('document:keydown.escape')
  public handleEscape(): void {
    this.isUserMenuOpen.set(false);
    this.isNotificationMenuOpen.set(false);
    this.isMobileSidebarOpen.set(false);
  }

  public toggleSidebar(): void {
    this.isSidebarCollapsed.update((val) => !val);
  }

  public toggleMobileSidebar(): void {
    this.isMobileSidebarOpen.update((val) => !val);
  }

  public closeMobileSidebar(): void {
    this.isMobileSidebarOpen.set(false);
  }

  public toggleUserMenu(event?: MouseEvent): void {
    event?.stopPropagation();
    this.isNotificationMenuOpen.set(false);
    this.isUserMenuOpen.update((val) => !val);
  }

  public closeUserMenu(): void {
    this.isUserMenuOpen.set(false);
  }

  public toggleNotificationMenu(event?: MouseEvent): void {
    event?.stopPropagation();
    this.isUserMenuOpen.set(false);
    const opening = !this.isNotificationMenuOpen();
    this.isNotificationMenuOpen.set(opening);
    if (opening) {
      this.refreshNotifications();
    }
  }

  public toggleTheme(): void {
    this.themeService.toggleTheme();
  }

  public navigateToProfile(): void {
    this.isUserMenuOpen.set(false);
    this.closeMobileSidebar();
    this.router.navigate(['/profile']);
  }

  public navigateToAlerts(): void {
    this.isNotificationMenuOpen.set(false);
    this.closeMobileSidebar();
    this.router.navigate(['/alerts']);
  }

  public handleLogout(): void {
    this.isUserMenuOpen.set(false);
    this.isNotificationMenuOpen.set(false);
    this.closeMobileSidebar();
    this.liveNotificationSub?.unsubscribe();
    this.authService.logout();
  }

  public markNotificationAsRead(item: AppNotification, event?: MouseEvent): void {
    event?.stopPropagation();
    if (item.read) return;

    this.notificationService.markAsRead(item.id).subscribe({
      next: () => {
        item.read = true;
        this.unreadCount.update((c) => Math.max(0, c - 1));
      },
      error: (err) => console.warn('Failed to mark notification read:', err),
    });
  }

  public markAllNotificationsAsRead(): void {
    this.notificationService.markAllAsRead().subscribe({
      next: () => {
        this.unreadCount.set(0);
        this.recentNotifications.update((list) => list.map((n) => ({ ...n, read: true })));
      },
      error: (err) => console.warn('Failed to mark all notifications read:', err),
    });
  }

  /**
   * Initializes real-time SSE listener and initial fetch for notifications.
   */
  private initNotificationFeed(): void {
    this.refreshNotifications();
    this.liveNotificationSub = this.notificationService.subscribeLiveNotifications().subscribe({
      next: (event) => {
        if (!event?.notification) return;
        this.unreadCount.update((c) => c + 1);
        this.recentNotifications.update((list) => [event.notification!, ...list.slice(0, MAX_RECENT_NOTIFICATIONS - 1)]);
      },
      error: (err) => console.debug('Live notifications connection ended:', err),
    });
  }

  private refreshNotifications(): void {
    this.notificationService.getUnreadCount().subscribe({
      next: (count) => this.unreadCount.set(count),
      error: (err) => console.warn('Failed to fetch unread count:', err),
    });

    this.notificationService.getNotifications({ limit: MAX_RECENT_NOTIFICATIONS }).subscribe({
      next: (items) => this.recentNotifications.set(items),
      error: (err) => console.warn('Failed to fetch recent notifications:', err),
    });
  }

  private closeMenusOnOutsideClick(target: HTMLElement | null): void {
    if (this.isUserMenuOpen()) {
      const insideMenu = target ? this.elementRef.nativeElement.querySelector('#user-menu-dropdown')?.contains(target) : false;
      const insideTrigger = target ? this.elementRef.nativeElement.querySelector('#user-menu-trigger')?.contains(target) : false;
      if (!insideMenu && !insideTrigger) this.isUserMenuOpen.set(false);
    }

    if (this.isNotificationMenuOpen()) {
      const insideNotifMenu = target ? this.elementRef.nativeElement.querySelector('#notification-dropdown')?.contains(target) : false;
      const insideNotifTrigger = target ? this.elementRef.nativeElement.querySelector('#notification-trigger')?.contains(target) : false;
      if (!insideNotifMenu && !insideNotifTrigger) this.isNotificationMenuOpen.set(false);
    }
  }
}
