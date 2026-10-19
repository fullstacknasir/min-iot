import { Component, OnInit, inject, signal } from '@angular/core';
import { DeviceService } from '../../../core/services/device.service';
import { Device, ProvisionResponse, CreateDevicePayload } from '../../../core/models/device.model';

@Component({
  selector: 'app-admin-devices',
  standalone: false,
  templateUrl: './admin-devices.component.html',
})
export class AdminDevicesComponent implements OnInit {
  private readonly deviceService = inject(DeviceService);

  public readonly devices = signal<Device[]>([]);
  public readonly isLoading = signal<boolean>(false);
  public readonly showCreateModal = signal<boolean>(false);
  public readonly showProvisionModal = signal<boolean>(false);
  public readonly provisionResult = signal<ProvisionResponse | null>(null);

  public newDeviceId = '';
  public newDeviceName = '';
  public newDeviceType = 'sensor';
  public actionMessage = signal<string | null>(null);

  public ngOnInit(): void {
    this.loadFleet();
  }

  public loadFleet(): void {
    this.isLoading.set(true);
    this.deviceService.getDevices().subscribe({
      next: (list) => {
        this.devices.set(list);
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public openCreateModal(): void {
    this.newDeviceId = '';
    this.newDeviceName = '';
    this.newDeviceType = 'sensor';
    this.showCreateModal.set(true);
  }

  public closeCreateModal(): void {
    this.showCreateModal.set(false);
  }

  public handleCreateDevice(): void {
    if (!this.newDeviceName) return;

    const payload: CreateDevicePayload = {
      id: this.newDeviceId.trim() || undefined,
      name: this.newDeviceName.trim(),
      device_type: this.newDeviceType.trim(),
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
    setTimeout(() => this.actionMessage.set(null), 4000);
  }
}
