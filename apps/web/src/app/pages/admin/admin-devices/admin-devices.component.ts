import { Component, OnInit, inject, signal } from '@angular/core';
import { DeviceService } from '../../../core/services/device.service';
import { UserService } from '../../../core/services/user.service';
import { Device, ProvisionResponse, CreateDevicePayload } from '../../../core/models/device.model';
import { User } from '../../../core/models/auth.model';
import {
  DEFAULT_PAGE,
  DEFAULT_PAGE_LIMIT,
  PAGE_SIZE_OPTIONS,
} from '../../../core/constants/app.constants';

const FEEDBACK_TIMEOUT_MS = 4000;

@Component({
  selector: 'app-admin-devices',
  standalone: false,
  templateUrl: './admin-devices.component.html',
})
export class AdminDevicesComponent implements OnInit {
  private readonly deviceService = inject(DeviceService);
  private readonly userService = inject(UserService);

  public readonly devices = signal<Device[]>([]);
  public readonly orgUsers = signal<User[]>([]);
  public readonly isLoading = signal<boolean>(false);
  public readonly showCreateModal = signal<boolean>(false);
  public readonly showProvisionModal = signal<boolean>(false);
  public readonly provisionResult = signal<ProvisionResponse | null>(null);
  public readonly actionMessage = signal<string | null>(null);

  public readonly currentPage = signal<number>(DEFAULT_PAGE);
  public readonly pageSize = signal<number>(DEFAULT_PAGE_LIMIT);
  public readonly totalCount = signal<number>(0);
  public readonly totalPages = signal<number>(1);
  public readonly pageSizeOptions = PAGE_SIZE_OPTIONS;

  public newDeviceId = '';
  public newDeviceName = '';
  public newDeviceType = 'sensor';
  public newDeviceAssignedUserId = '';

  public ngOnInit(): void {
    this.loadFleet();
    this.loadOrgUsers();
  }

  public loadFleet(page = this.currentPage(), limit = this.pageSize()): void {
    this.isLoading.set(true);
    this.deviceService.getPaginatedDevices({ page, limit, all: true }).subscribe({
      next: (res) => {
        this.devices.set(res?.data ?? []);
        this.totalCount.set(res?.total ?? 0);
        this.currentPage.set(res?.page ?? page);
        this.pageSize.set(res?.limit ?? limit);
        this.totalPages.set(res?.total_pages ?? 1);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public loadOrgUsers(): void {
    this.userService.getUsers({ all: true }).subscribe({
      next: (users) => this.orgUsers.set(users),
      error: (err) => console.warn('Failed to load organization users:', err),
    });
  }

  public onPageChange(newPage: number): void {
    this.loadFleet(newPage, this.pageSize());
  }

  public onPageSizeChange(newSize: number): void {
    this.pageSize.set(newSize);
    this.loadFleet(DEFAULT_PAGE, newSize);
  }

  public openCreateModal(): void {
    this.newDeviceId = '';
    this.newDeviceName = '';
    this.newDeviceType = 'sensor';
    this.newDeviceAssignedUserId = '';
    this.showCreateModal.set(true);
  }

  public closeCreateModal(): void {
    this.showCreateModal.set(false);
  }

  public handleCreateDevice(): void {
    if (!this.newDeviceName.trim()) return;

    const payload: CreateDevicePayload = {
      id: this.newDeviceId.trim() || undefined,
      name: this.newDeviceName.trim(),
      device_type: this.newDeviceType.trim(),
      assigned_user_id: this.newDeviceAssignedUserId || undefined,
    };

    this.deviceService.createDevice(payload).subscribe({
      next: () => {
        this.closeCreateModal();
        this.loadFleet();
        this.setActionFeedback('Device registered successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to register device.'),
    });
  }

  /**
   * Assigns or unassigns device to a specific organization user.
   */
  public handleAssignUser(device: Device, newUserId: string): void {
    const isUnassigning = !newUserId || newUserId === '';
    const updatePayload = isUnassigning
      ? { clear_assigned_user: true }
      : { assigned_user_id: newUserId };

    this.deviceService.updateDevice(device.id, updatePayload).subscribe({
      next: () => {
        this.loadFleet();
        this.setActionFeedback(isUnassigning ? 'Device unassigned.' : 'User assigned to device.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to update device assignment.'),
    });
  }

  public provisionDevice(deviceId: string): void {
    this.deviceService.provisionDevice(deviceId).subscribe({
      next: (res) => {
        this.provisionResult.set(res);
        this.showProvisionModal.set(true);
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to provision device.'),
    });
  }

  public enableDevice(deviceId: string): void {
    this.deviceService.enableDevice(deviceId).subscribe({
      next: () => {
        this.loadFleet();
        this.setActionFeedback('Device enabled.');
      },
    });
  }

  public disableDevice(deviceId: string): void {
    this.deviceService.disableDevice(deviceId).subscribe({
      next: () => {
        this.loadFleet();
        this.setActionFeedback('Device disabled.');
      },
    });
  }

  public revokeDevice(deviceId: string): void {
    if (!confirm('Are you sure you want to revoke this device? Its MQTT credentials will be permanently invalidated.')) {
      return;
    }
    this.deviceService.revokeDevice(deviceId).subscribe({
      next: () => {
        this.loadFleet();
        this.setActionFeedback('Device credentials revoked.');
      },
    });
  }

  public deleteDevice(deviceId: string): void {
    if (!confirm(`Permanently delete device ${deviceId}?`)) return;

    this.deviceService.deleteDevice(deviceId).subscribe({
      next: () => {
        this.loadFleet();
        this.setActionFeedback('Device deleted.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to delete device.'),
    });
  }

  public copyTokenToClipboard(token: string): void {
    navigator.clipboard.writeText(token);
    alert('Credential secret copied to clipboard!');
  }

  private setActionFeedback(msg: string): void {
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), FEEDBACK_TIMEOUT_MS);
  }
}
