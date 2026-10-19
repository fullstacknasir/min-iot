import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { DeviceService } from '../../core/services/device.service';
import { TelemetryService } from '../../core/services/telemetry.service';
import { Device } from '../../core/models/device.model';

const SIMULATED_BASE_TEMP = 25.0;
const SIMULATED_BASE_HUMIDITY = 55.0;

@Component({
  selector: 'app-device-detail',
  standalone: false,
  templateUrl: './device-detail.component.html',
})
export class DeviceDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly deviceService = inject(DeviceService);
  private readonly telemetryService = inject(TelemetryService);

  public readonly deviceId = signal<string>('');
  public readonly device = signal<Device | null>(null);
  public readonly isLoading = signal<boolean>(true);
  public readonly isSimulating = signal<boolean>(false);

  public ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.router.navigate(['/devices']);
      return;
    }
    this.deviceId.set(id);
    this.loadDeviceDetails(id);
  }

  public navigateBack(): void {
    this.router.navigate(['/devices']);
  }

  /**
   * Generates a simulated telemetry packet to test live SSE streaming.
   */
  public simulateTelemetry(): void {
    const currentId = this.deviceId();
    if (!currentId) return;

    this.isSimulating.set(true);
    const metrics: Record<string, number> = {
      temperature: Number((SIMULATED_BASE_TEMP + Math.random() * 5).toFixed(2)),
      humidity: Number((SIMULATED_BASE_HUMIDITY + Math.random() * 10).toFixed(2)),
      voltage: Number((3.3 + Math.random() * 0.2).toFixed(2)),
    };

    this.telemetryService.sendSimulatedTelemetry(currentId, metrics).subscribe({
      next: () => this.isSimulating.set(false),
      error: () => this.isSimulating.set(false),
    });
  }

  private loadDeviceDetails(id: string): void {
    this.isLoading.set(true);
    this.deviceService.getDevice(id).subscribe({
      next: (dev) => {
        this.device.set(dev);
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
        this.router.navigate(['/devices']);
      },
    });
  }
}
