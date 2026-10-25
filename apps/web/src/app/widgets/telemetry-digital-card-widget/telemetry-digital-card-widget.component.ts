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
} from '@angular/core';
import { Subscription } from 'rxjs';
import moment from 'moment';
import { TelemetryService } from '../../core/services/telemetry.service';
import { LatestTelemetryData } from '../../core/models/telemetry.model';
import { FORMAT_DATETIME_DISPLAY } from '../../core/constants/app.constants';

const DEFAULT_METRIC = 'temperature';

const DEFAULT_UNITS: Record<string, string> = {
  temperature: '°C',
  temp: '°C',
  humidity: '%',
  pressure: 'hPa',
  voltage: 'V',
  current: 'A',
  power: 'W',
  battery: '%',
  rssi: 'dBm',
};

const DEFAULT_ICONS: Record<string, string> = {
  temperature: 'thermostat',
  temp: 'thermostat',
  humidity: 'water_drop',
  pressure: 'speed',
  voltage: 'bolt',
  current: 'electric_meter',
  power: 'flash_on',
  battery: 'battery_std',
  rssi: 'wifi',
};

@Component({
  selector: 'app-telemetry-digital-card-widget',
  standalone: false,
  templateUrl: './telemetry-digital-card-widget.component.html',
})
export class TelemetryDigitalCardWidgetComponent implements OnInit, OnDestroy, OnChanges {
  @Input({ required: true }) public deviceId!: string;
  @Input() public metricName: string = DEFAULT_METRIC;
  @Input() public title?: string;
  @Input() public unit?: string;
  @Input() public icon?: string;

  private readonly telemetryService = inject(TelemetryService);
  private liveStreamSubscription?: Subscription;

  public readonly currentValue = signal<string | number>('--');
  public readonly lastReceivedTime = signal<string | null>(null);
  public readonly isConnected = signal<boolean>(false);

  public ngOnInit(): void {
    this.tbOnInit();
  }

  public ngOnChanges(changes: SimpleChanges): void {
    if ((changes['deviceId'] || changes['metricName']) && !changes['deviceId']?.isFirstChange()) {
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
    this.fetchLatestTelemetry();
    this.subscribeLiveStream();
  }

  /**
   * ThingsBoard Widget Lifecycle: onDataUpdated
   */
  public tbOnDataUpdated(data: LatestTelemetryData): void {
    if (!data?.metrics) return;
    const rawVal = data.metrics[this.metricName];
    if (typeof rawVal === 'number' && !isNaN(rawVal)) {
      this.currentValue.set(Number(rawVal.toFixed(2)));
    } else if (rawVal !== undefined && rawVal !== null) {
      this.currentValue.set(String(rawVal));
    }
    const ts = data.received_at ?? data.timestamp ?? new Date().toISOString();
    this.lastReceivedTime.set(moment(ts).format(FORMAT_DATETIME_DISPLAY));
  }

  /**
   * ThingsBoard Widget Lifecycle: onResize
   */
  public tbOnResize(): void {
    // ThingsBoard responsive reflow hook
  }

  /**
   * ThingsBoard Widget Lifecycle: onDestroy
   */
  public tbOnDestroy(): void {
    this.liveStreamSubscription?.unsubscribe();
    this.isConnected.set(false);
  }

  public getResolvedUnit(): string {
    return this.unit || DEFAULT_UNITS[this.metricName.toLowerCase()] || '';
  }

  public getResolvedIcon(): string {
    return this.icon || DEFAULT_ICONS[this.metricName.toLowerCase()] || 'sensors';
  }

  private fetchLatestTelemetry(): void {
    this.telemetryService.getLatestTelemetry(this.deviceId).subscribe((data) => {
      if (data) this.tbOnDataUpdated(data);
    });
  }

  private subscribeLiveStream(): void {
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
