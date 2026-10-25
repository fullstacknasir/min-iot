import {
  Component,
  Input,
  OnInit,
  OnDestroy,
  OnChanges,
  SimpleChanges,
  ElementRef,
  ViewChild,
  HostListener,
  inject,
  signal,
} from '@angular/core';
import { Subscription } from 'rxjs';
import moment from 'moment';
import * as echarts from 'echarts';
import { TelemetryService } from '../../core/services/telemetry.service';
import { LatestTelemetryData } from '../../core/models/telemetry.model';
import { ThemeService } from '../../core/services/theme.service';
import { FORMAT_DATETIME_DISPLAY } from '../../core/constants/app.constants';

const DEFAULT_MIN = 0;
const DEFAULT_MAX = 100;
const DEFAULT_METRIC = 'temperature';

const DEFAULT_METRIC_UNITS: Record<string, string> = {
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

@Component({
  selector: 'app-telemetry-gauge-widget',
  standalone: false,
  templateUrl: './telemetry-gauge-widget.component.html',
})
export class TelemetryGaugeWidgetComponent implements OnInit, OnDestroy, OnChanges {
  @Input({ required: true }) public deviceId!: string;
  @Input() public metricName: string = DEFAULT_METRIC;
  @Input() public title?: string;
  @Input() public min: number = DEFAULT_MIN;
  @Input() public max: number = DEFAULT_MAX;
  @Input() public unit?: string;

  @ViewChild('gaugeContainer', { static: true }) public gaugeContainer!: ElementRef<HTMLDivElement>;

  private readonly telemetryService = inject(TelemetryService);
  private readonly themeService = inject(ThemeService);

  private liveStreamSubscription?: Subscription;
  private gaugeInstance?: echarts.ECharts;
  private resizeObserver?: ResizeObserver;

  public readonly currentValue = signal<number | null>(null);
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
    if (!this.gaugeContainer?.nativeElement || !this.deviceId) return;
    this.initGaugeInstance();
    this.setupResizeObserver();
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
      const rounded = Number(rawVal.toFixed(1));
      this.currentValue.set(rounded);
      this.updateGaugeVisual(rounded);
    }
    const ts = data.received_at ?? data.timestamp ?? new Date().toISOString();
    this.lastReceivedTime.set(moment(ts).format(FORMAT_DATETIME_DISPLAY));
  }

  /**
   * ThingsBoard Widget Lifecycle: onResize
   */
  public tbOnResize(): void {
    this.gaugeInstance?.resize();
  }

  /**
   * ThingsBoard Widget Lifecycle: onDestroy
   */
  public tbOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.liveStreamSubscription?.unsubscribe();
    if (this.gaugeInstance && !this.gaugeInstance.isDisposed()) {
      try {
        this.gaugeInstance.dispose();
      } catch (err) {
        console.debug('Gauge dispose handled:', err);
      }
    }
    this.gaugeInstance = undefined;
    this.isConnected.set(false);
  }

  public getResolvedUnit(): string {
    return this.unit || DEFAULT_METRIC_UNITS[this.metricName.toLowerCase()] || '';
  }

  private initGaugeInstance(): void {
    if (typeof window === 'undefined' || !this.gaugeContainer?.nativeElement) return;
    try {
      const isDark = this.themeService.isDarkMode();
      this.gaugeInstance = echarts.init(this.gaugeContainer.nativeElement, isDark ? 'dark' : undefined);
      this.updateGaugeVisual(this.currentValue() ?? this.min);
    } catch (err) {
      console.debug('Canvas context unavailable in current environment:', err);
    }
  }

  private updateGaugeVisual(val: number): void {
    if (!this.gaugeInstance) return;
    const isDark = this.themeService.isDarkMode();
    const effectiveUnit = this.getResolvedUnit();

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      series: [
        {
          type: 'gauge',
          center: ['50%', '60%'],
          radius: '90%',
          startAngle: 200,
          endAngle: -20,
          min: this.min,
          max: this.max,
          splitNumber: 5,
          progress: {
            show: true,
            width: 14,
            roundCap: true,
            itemStyle: {
              color: '#06b6d4',
            },
          },
          pointer: {
            show: true,
            length: '65%',
            width: 4,
            itemStyle: {
              color: isDark ? '#f8fafc' : '#0f172a',
            },
          },
          axisLine: {
            roundCap: true,
            lineStyle: {
              width: 14,
              color: [[1, isDark ? '#1e293b' : '#e2e8f0']],
            },
          },
          axisTick: {
            show: false,
          },
          splitLine: {
            length: 8,
            lineStyle: {
              width: 1,
              color: isDark ? '#475569' : '#cbd5e1',
            },
          },
          axisLabel: {
            distance: 20,
            color: isDark ? '#94a3b8' : '#64748b',
            fontSize: 10,
          },
          title: {
            show: false,
          },
          detail: {
            valueAnimation: true,
            fontSize: 20,
            fontWeight: 'bold',
            offsetCenter: [0, '25%'],
            color: isDark ? '#f8fafc' : '#0f172a',
            formatter: (v: number) => `${v.toFixed(1)}${effectiveUnit}`,
          },
          data: [{ value: val }],
        },
      ],
    };

    try {
      this.gaugeInstance.setOption(option, true);
    } catch (err) {
      console.debug('Failed updating gauge visual:', err);
    }
  }

  private setupResizeObserver(): void {
    if (typeof ResizeObserver === 'undefined') return;
    this.resizeObserver = new ResizeObserver(() => this.tbOnResize());
    this.resizeObserver.observe(this.gaugeContainer.nativeElement);
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
