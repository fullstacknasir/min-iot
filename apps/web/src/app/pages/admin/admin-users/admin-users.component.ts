import { Component, OnInit, inject, signal } from '@angular/core';
import { UserService } from '../../../core/services/user.service';
import { AuthService } from '../../../core/services/auth.service';
import { User, UserRole, CreateOrgUserPayload } from '../../../core/models/auth.model';
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
  PAGE_SIZE_OPTIONS,
} from '../../../core/constants/app.constants';

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

  public readonly currentPage = signal<number>(DEFAULT_PAGE);
  public readonly pageSize = signal<number>(DEFAULT_PAGE_LIMIT);
  public readonly totalCount = signal<number>(0);
  public readonly totalPages = signal<number>(1);
  public readonly pageSizeOptions = PAGE_SIZE_OPTIONS;

  public newUserEmail = '';
  public newUserPassword = '';
  public newUserRole: UserRole = 'viewer';
  public actionMessage = signal<string | null>(null);

  public ngOnInit(): void {
    this.loadUsers();
  }

  public loadUsers(page = this.currentPage(), limit = this.pageSize()): void {
    this.isLoading.set(true);
    this.userService.getPaginatedUsers({ page, limit }).subscribe({
      next: (res) => {
        this.users.set(res?.data ?? []);
        this.totalCount.set(res?.total ?? 0);
        this.currentPage.set(res?.page ?? page);
        this.pageSize.set(res?.limit ?? limit);
        this.totalPages.set(res?.total_pages ?? 1);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public onPageChange(newPage: number): void {
    this.loadUsers(newPage, this.pageSize());
  }

  public onPageSizeChange(newSize: number): void {
    this.pageSize.set(newSize);
    this.loadUsers(DEFAULT_PAGE, newSize);
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
