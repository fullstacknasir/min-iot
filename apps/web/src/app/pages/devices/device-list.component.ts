import { Component, inject, signal, computed } from '@angular/core';
import { Router } from '@angular/router';
import { injectQuery } from '@tanstack/angular-query-experimental';
import { lastValueFrom } from 'rxjs';
import { DeviceService } from '../../core/services/device.service';
import { AuthService } from '../../core/services/auth.service';
import { Device, DeviceStatus } from '../../core/models/device.model';
import {
  STATUS_ONLINE,
  STATUS_OFFLINE,
  STATUS_DISABLED,
  STATUS_REVOKED,
  DEFAULT_REFRESH_INTERVAL_MS,
  DEFAULT_GRID_PAGE_LIMIT,
  GRID_PAGE_SIZE_OPTIONS,
} from '../../core/constants/app.constants';

@Component({
  selector: 'app-device-list',
  standalone: false,
  templateUrl: './device-list.component.html',
})
export class DeviceListComponent {
  private readonly deviceService = inject(DeviceService);
  private readonly router = inject(Router);
  public readonly authService = inject(AuthService);

  public readonly searchQuery = signal<string>('');
  public readonly filterStatus = signal<string>('all');

  /**
   * TanStack Query for caching and auto-polling the fleet of IoT devices.
   */
  public readonly devicesQuery = injectQuery(() => ({
    queryKey: ['devices'],
    queryFn: () => lastValueFrom(this.deviceService.getDevices()),
    refetchInterval: DEFAULT_REFRESH_INTERVAL_MS,
  }));

  public readonly allDevices = computed(() => this.devicesQuery.data() ?? []);

  public readonly totalCount = computed(() => this.allDevices().length);
  public readonly onlineCount = computed(
    () => this.allDevices().filter((d) => d.status === STATUS_ONLINE).length
  );
  public readonly offlineCount = computed(
    () => this.allDevices().filter((d) => d.status === STATUS_OFFLINE).length
  );
  public readonly disabledCount = computed(
    () => this.allDevices().filter((d) => d.status === STATUS_DISABLED || d.status === STATUS_REVOKED).length
  );

  public readonly currentPage = signal<number>(1);
  public readonly pageSize = signal<number>(DEFAULT_GRID_PAGE_LIMIT);
  public readonly pageSizeOptions = GRID_PAGE_SIZE_OPTIONS;

  public readonly filteredDevices = computed(() => {
    const list = this.allDevices();
    const query = this.searchQuery().trim().toLowerCase();
    const status = this.filterStatus();

    return list.filter((device) => {
      const matchesSearch =
        !query ||
        device.name.toLowerCase().includes(query) ||
        device.id.toLowerCase().includes(query) ||
        device.device_type.toLowerCase().includes(query);

      const matchesStatus = status === 'all' || device.status === status;
      return matchesSearch && matchesStatus;
    });
  });

  public readonly totalFiltered = computed(() => this.filteredDevices().length);
  public readonly totalPages = computed(() =>
    Math.max(1, Math.ceil(this.totalFiltered() / this.pageSize()))
  );

  public readonly paginatedDevices = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredDevices().slice(start, start + this.pageSize());
  });

  public setFilter(status: string): void {
    this.filterStatus.set(status);
    this.currentPage.set(1);
  }

  public onSearchInput(event: Event): void {
    const target = event.target as HTMLInputElement;
    this.searchQuery.set(target?.value ?? '');
    this.currentPage.set(1);
  }

  public onPageChange(newPage: number): void {
    this.currentPage.set(newPage);
  }

  public onPageSizeChange(newSize: number): void {
    this.pageSize.set(newSize);
    this.currentPage.set(1);
  }

  public navigateToDevice(deviceId: string): void {
    if (!deviceId) return;
    this.router.navigate(['/devices', deviceId]);
  }

  public getStatusBadgeClass(status: DeviceStatus): string {
    switch (status) {
      case STATUS_ONLINE:
        return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30';
      case STATUS_OFFLINE:
        return 'bg-slate-700/50 text-slate-400 border-slate-600/30';
      case STATUS_DISABLED:
        return 'bg-amber-500/10 text-amber-400 border-amber-500/30';
      case STATUS_REVOKED:
        return 'bg-red-500/10 text-red-400 border-red-500/30';
      default:
        return 'bg-slate-700/50 text-slate-400 border-slate-600/30';
    }
  }
}
