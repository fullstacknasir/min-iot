import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

const DEFAULT_DEMO_EMAIL = 'admin@miniot.io';
const DEFAULT_DEMO_PASS = 'AdminPassword123!';

@Component({
  selector: 'app-login',
  standalone: false,
  templateUrl: './login.component.html',
})
export class LoginComponent {
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  public email = '';
  public password = '';
  public readonly isSubmitting = signal<boolean>(false);
  public readonly errorMessage = signal<string | null>(null);

  public onSubmit(): void {
    if (!this.email || !this.password) {
      this.errorMessage.set('Please enter both email and password.');
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    this.authService.login({ email: this.email, password: this.password }).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.router.navigate(['/devices']);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        this.handleAuthError(err);
      },
    });
  }

  public fillDemoCredentials(): void {
    this.email = DEFAULT_DEMO_EMAIL;
    this.password = DEFAULT_DEMO_PASS;
    this.errorMessage.set(null);
  }

  private handleAuthError(err: { status?: number; error?: { error?: string } }): void {
    if (err?.status === 429) {
      this.errorMessage.set('Too many login attempts. Please wait 15 minutes before retrying.');
      return;
    }
    const apiError = err?.error?.error ?? 'Invalid email or password. Please try again.';
    this.errorMessage.set(apiError);
  }
}
