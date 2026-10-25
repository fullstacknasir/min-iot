import { Component, OnInit, inject, signal } from '@angular/core';
import moment from 'moment';
import { AuthService } from '../../core/services/auth.service';
import { User, Organization } from '../../core/models/auth.model';
import { FORMAT_DATETIME_DISPLAY } from '../../core/constants/app.constants';

const MIN_PASSWORD_LENGTH = 8;
const FEEDBACK_TIMEOUT_MS = 4000;

@Component({
  selector: 'app-profile',
  standalone: false,
  templateUrl: './profile.component.html',
})
export class ProfileComponent implements OnInit {
  public readonly authService = inject(AuthService);

  public readonly user = signal<User | null>(null);
  public readonly organization = signal<Organization | null>(null);
  public readonly isSavingProfile = signal<boolean>(false);
  public readonly isChangingPassword = signal<boolean>(false);
  public readonly successMessage = signal<string | null>(null);
  public readonly errorMessage = signal<string | null>(null);

  public editName = '';
  public editEmail = '';
  public currentPassword = '';
  public newPassword = '';
  public confirmPassword = '';

  public ngOnInit(): void {
    this.loadProfile();
  }

  /**
   * Loads current user profile and populates editable form fields.
   */
  public loadProfile(): void {
    const cachedUser = this.authService.currentUser();
    if (cachedUser) this.populateForm(cachedUser);

    this.authService.getProfile().subscribe({
      next: (res) => {
        if (res.user) {
          this.user.set(res.user);
          this.populateForm(res.user);
        }
        if (res.organization) this.organization.set(res.organization);
      },
      error: () => {
        // Fallback to cached auth context if API transient error
        this.user.set(this.authService.currentUser());
        this.organization.set(this.authService.currentOrg());
      },
    });
  }

  /**
   * Submits profile updates (name and email).
   */
  public handleSaveProfile(): void {
    if (!this.editEmail.trim()) {
      this.showFeedback('Email address is required.', false);
      return;
    }

    this.isSavingProfile.set(true);
    this.authService
      .updateProfile({
        name: this.editName.trim(),
        email: this.editEmail.trim(),
      })
      .subscribe({
        next: (res) => {
          this.isSavingProfile.set(false);
          this.user.set(res.user);
          this.showFeedback('Profile updated successfully.', true);
        },
        error: (err) => {
          this.isSavingProfile.set(false);
          this.showFeedback(err?.error?.error ?? 'Failed to update profile.', false);
        },
      });
  }

  /**
   * Submits password change request.
   */
  public handleChangePassword(): void {
    if (!this.validatePasswordInputs()) return;

    this.isChangingPassword.set(true);
    this.authService
      .updateProfile({
        current_password: this.currentPassword,
        new_password: this.newPassword,
      })
      .subscribe({
        next: () => {
          this.isChangingPassword.set(false);
          this.resetPasswordFields();
          this.showFeedback('Password changed successfully.', true);
        },
        error: (err) => {
          this.isChangingPassword.set(false);
          this.showFeedback(err?.error?.error ?? 'Failed to update password.', false);
        },
      });
  }

  public formatMemberSince(isoDate?: string): string {
    if (!isoDate) return 'N/A';
    return moment(isoDate).format(FORMAT_DATETIME_DISPLAY);
  }

  private populateForm(userData: User): void {
    this.editName = userData.name ?? '';
    this.editEmail = userData.email ?? '';
  }

  private validatePasswordInputs(): boolean {
    if (!this.currentPassword) {
      this.showFeedback('Current password is required.', false);
      return false;
    }
    if (this.newPassword.length < MIN_PASSWORD_LENGTH) {
      this.showFeedback(`New password must be at least ${MIN_PASSWORD_LENGTH} characters.`, false);
      return false;
    }
    if (this.newPassword !== this.confirmPassword) {
      this.showFeedback('New password and confirmation do not match.', false);
      return false;
    }
    return true;
  }

  private resetPasswordFields(): void {
    this.currentPassword = '';
    this.newPassword = '';
    this.confirmPassword = '';
  }

  private showFeedback(message: string, isSuccess: boolean): void {
    if (isSuccess) {
      this.successMessage.set(message);
      this.errorMessage.set(null);
    } else {
      this.errorMessage.set(message);
      this.successMessage.set(null);
    }
    setTimeout(() => {
      this.successMessage.set(null);
      this.errorMessage.set(null);
    }, FEEDBACK_TIMEOUT_MS);
  }
}
