import { Component, OnInit, inject, signal } from '@angular/core';
import { UserService } from '../../../core/services/user.service';
import { AuthService } from '../../../core/services/auth.service';
import { User, UserRole, CreateOrgUserPayload } from '../../../core/models/auth.model';

@Component({
  selector: 'app-admin-users',
  standalone: false,
  templateUrl: './admin-users.component.html',
})
export class AdminUsersComponent implements OnInit {
  private readonly userService = inject(UserService);
  public readonly authService = inject(AuthService);

  public readonly users = signal<User[]>([]);
  public readonly isLoading = signal<boolean>(false);
  public readonly showCreateModal = signal<boolean>(false);

  public newUserEmail = '';
  public newUserPassword = '';
  public newUserRole: UserRole = 'viewer';
  public actionMessage = signal<string | null>(null);

  public ngOnInit(): void {
    this.loadUsers();
  }

  public loadUsers(): void {
    this.isLoading.set(true);
    this.userService.getUsers().subscribe({
      next: (list) => {
        this.users.set(list);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public openCreateModal(): void {
    this.newUserEmail = '';
    this.newUserPassword = '';
    this.newUserRole = 'viewer';
    this.showCreateModal.set(true);
  }

  public closeCreateModal(): void {
    this.showCreateModal.set(false);
  }

  public handleCreateUser(): void {
    if (!this.newUserEmail || !this.newUserPassword) return;

    const payload: CreateOrgUserPayload = {
      email: this.newUserEmail.trim(),
      password: this.newUserPassword,
      role: this.newUserRole,
    };

    this.userService.createUser(payload).subscribe({
      next: () => {
        this.closeCreateModal();
        this.loadUsers();
        this.setActionFeedback('New user invited successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to create user.'),
    });
  }

  public handleRoleChange(user: User, newRole: string): void {
    if (user.role === newRole) return;

    this.userService.updateUserRole(user.id, newRole as UserRole).subscribe({
      next: () => {
        this.loadUsers();
        this.setActionFeedback(`Updated role for ${user.email} to ${newRole}.`);
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to update role.'),
    });
  }

  public handleDeleteUser(user: User): void {
    if (!confirm(`Are you sure you want to remove user ${user.email}?`)) return;

    this.userService.deleteUser(user.id).subscribe({
      next: () => {
        this.loadUsers();
        this.setActionFeedback('User removed.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to delete user.'),
    });
  }

  private setActionFeedback(msg: string): void {
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), 4000);
  }
}
