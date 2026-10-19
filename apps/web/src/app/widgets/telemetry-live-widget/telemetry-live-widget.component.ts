import {
  Component,
  Input,
  OnInit,
  OnDestroy,
  OnChanges,
  SimpleChanges,
  HostListener,
  inject,
  signal,
  computed,
} from '@angular/core';
import { Subscription } from 'rxjs';
import moment from 'moment';
import { TelemetryService } from '../../core/services/telemetry.service';
import { LatestTelemetryData } from '../../core/models/telemetry.model';
import { FORMAT_DATETIME_DISPLAY } from '../../core/constants/app.constants';

const DEFAULT_METRIC_UNIT = '';
const METRIC_UNITS: Record<string, string> = {
  temperature: '°C',
  temp: '°C',
  humidity: '%',
  pressure: 'hPa',
  voltage: 'V',
  current: 'A',
  power: 'W',
  battery: '%',
};

const METRIC_ICONS: Record<string, string> = {
  temperature: 'thermostat',
  temp: 'thermostat',
  humidity: 'water_drop',
  pressure: 'speed',
  voltage: 'bolt',
  current: 'electric_meter',
  power: 'flash_on',
  battery: 'battery_std',
};

@Component({
  selector: 'app-telemetry-live-widget',
  standalone: false,
  templateUrl: './telemetry-live-widget.component.html',
})
export class TelemetryLiveWidgetComponent implements OnInit, OnDestroy, OnChanges {
  @Input({ required: true }) public deviceId!: string;

  private readonly telemetryService = inject(TelemetryService);
  private liveStreamSubscription?: Subscription;

  public readonly latestData = signal<LatestTelemetryData | null>(null);
  public readonly isConnected = signal<boolean>(false);
  public readonly lastReceivedTime = signal<string | null>(null);

  public readonly metricList = computed(() => {
    const metrics = this.latestData()?.metrics ?? {};
    return Object.entries(metrics).map(([name, value]) => ({
      name,
      value: typeof value === 'number' ? Number(value.toFixed(2)) : value,
      unit: METRIC_UNITS[name.toLowerCase()] ?? DEFAULT_METRIC_UNIT,
      icon: METRIC_ICONS[name.toLowerCase()] ?? 'sensors',
    }));
  });

  public ngOnInit(): void {
    this.tbOnInit();
  }

  public ngOnChanges(changes: SimpleChanges): void {
    if (changes['deviceId'] && !changes['deviceId'].isFirstChange()) {
      this.tbOnDestroy();
      this.tbOnInit();
    }
  }

  public ngOnDestroy(): void {
    this.tbOnDestroy();
  }

  @HostListener('window:resize')
  public handleResize(): void {
    this.tbOnResize();
  }

  /**
   * ThingsBoard Widget Lifecycle: onInit
   */
  public tbOnInit(): void {
    if (!this.deviceId) return;
    this.telemetryService.getLatestTelemetry(this.deviceId).subscribe((data) => {
      if (data) this.tbOnDataUpdated(data);
    });
    this.subscribeLiveStream();
  }

  /**
   * ThingsBoard Widget Lifecycle: onDataUpdated
   */
  public tbOnDataUpdated(data: LatestTelemetryData): void {
    if (!data) return;
    this.latestData.set(data);
    const ts = data.received_at ?? data.timestamp ?? new Date().toISOString();
    this.lastReceivedTime.set(moment(ts).format(FORMAT_DATETIME_DISPLAY));
  }

  /**
   * ThingsBoard Widget Lifecycle: onResize
   */
  public tbOnResize(): void {
    // Thingsboard widget responsive reflow handling
  }

  /**
   * ThingsBoard Widget Lifecycle: onDestroy
   */
  public tbOnDestroy(): void {
    this.liveStreamSubscription?.unsubscribe();
    this.isConnected.set(false);
  }

  private subscribeLiveStream(): void {
    this.liveStreamSubscription?.unsubscribe();
    this.liveStreamSubscription = this.telemetryService
      .connectLiveStream(this.deviceId)
      .subscribe({
        next: (telemetry) => {
          this.isConnected.set(true);
          this.tbOnDataUpdated(telemetry);
        },
        error: () => this.isConnected.set(false),
      });
  }
}
