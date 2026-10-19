import { Component, HostListener, inject, signal } from '@angular/core';
import { AuthService } from './core/services/auth.service';

@Component({
  selector: 'app-root',
  standalone: false,
  templateUrl: './app.component.html',
})
export class AppComponent {
  public readonly authService = inject(AuthService);
  public readonly windowInnerWidth = signal<number>(window.innerWidth);

  @HostListener('window:resize')
  public handleWindowResize(): void {
    this.windowInnerWidth.set(window.innerWidth);
  }

  public handleLogout(): void {
    this.authService.logout();
  }
}
