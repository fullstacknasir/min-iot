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
import moment from 'moment';
import * as echarts from 'echarts';
import { TelemetryService } from '../../core/services/telemetry.service';
import { TelemetryPoint, TimeRangeOption } from '../../core/models/telemetry.model';
import {
  TIME_RANGE_1H,
  TIME_RANGE_6H,
  TIME_RANGE_24H,
  TIME_RANGE_7D,
  TIME_RANGE_30D,
  FORMAT_DATETIME_DISPLAY,
} from '../../core/constants/app.constants';

const TIME_RANGES: TimeRangeOption[] = [
  { label: 'Last 1 Hour', value: TIME_RANGE_1H, bucket: '1m', durationMinutes: 60 },
  { label: 'Last 6 Hours', value: TIME_RANGE_6H, bucket: '5m', durationMinutes: 360 },
  { label: 'Last 24 Hours', value: TIME_RANGE_24H, bucket: '15m', durationMinutes: 1440 },
  { label: 'Last 7 Days', value: TIME_RANGE_7D, bucket: '1h', durationMinutes: 10080 },
  { label: 'Last 30 Days', value: TIME_RANGE_30D, bucket: '6h', durationMinutes: 43200 },
];

const METRIC_COLORS = ['#06b6d4', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#3b82f6'];

@Component({
  selector: 'app-telemetry-chart-widget',
  standalone: false,
  templateUrl: './telemetry-chart-widget.component.html',
})
export class TelemetryChartWidgetComponent implements OnInit, OnDestroy, OnChanges {
  @Input({ required: true }) public deviceId!: string;
  @ViewChild('chartContainer', { static: true }) public chartContainer!: ElementRef<HTMLDivElement>;

  private readonly telemetryService = inject(TelemetryService);
  private echartsInstance?: echarts.ECharts;
  private resizeObserver?: ResizeObserver;

  public readonly timeRanges = TIME_RANGES;
  public readonly selectedRange = signal<string>(TIME_RANGE_24H);
  public readonly isLoading = signal<boolean>(false);
  public readonly availableMetrics = signal<string[]>([]);
  public readonly selectedMetric = signal<string>('all');

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
    if (!this.chartContainer?.nativeElement) return;
    this.initChartInstance();
    this.setupResizeObserver();
    this.fetchHistoricalData();
  }

  /**
   * ThingsBoard Widget Lifecycle: onDataUpdated
   */
  public tbOnDataUpdated(points: TelemetryPoint[]): void {
    if (!this.echartsInstance) return;
    const seriesData = this.groupPointsByMetric(points);
    this.updateMetricsList(Object.keys(seriesData));
    const option = this.buildChartOption(seriesData);
    this.echartsInstance.setOption(option, true);
  }

  /**
   * ThingsBoard Widget Lifecycle: onResize
   */
  public tbOnResize(): void {
    if (this.echartsInstance && !this.echartsInstance.isDisposed()) {
      this.echartsInstance.resize();
    }
  }

  /**
   * ThingsBoard Widget Lifecycle: onDestroy
   */
  public tbOnDestroy(): void {
    this.resizeObserver?.disconnect();
    if (this.echartsInstance && !this.echartsInstance.isDisposed()) {
      this.echartsInstance.dispose();
    }
  }

  public setTimeRange(rangeValue: string): void {
    this.selectedRange.set(rangeValue);
    this.fetchHistoricalData();
  }

  public setMetricFilter(metric: string): void {
    this.selectedMetric.set(metric);
    this.fetchHistoricalData();
  }

  private initChartInstance(): void {
    this.echartsInstance = echarts.init(this.chartContainer.nativeElement, undefined, {
      renderer: 'canvas',
    });
  }

  private setupResizeObserver(): void {
    if (typeof ResizeObserver === 'undefined') return;
    this.resizeObserver = new ResizeObserver(() => this.tbOnResize());
    this.resizeObserver.observe(this.chartContainer.nativeElement);
  }

  /**
   * Queries historical telemetry points using momentjs calculated time-range.
   */
  private fetchHistoricalData(): void {
    if (!this.deviceId) return;
    this.isLoading.set(true);

    const rangeConfig = this.timeRanges.find((r) => r.value === this.selectedRange()) ?? this.timeRanges[2];
    const from = moment().subtract(rangeConfig.durationMinutes, 'minutes').toISOString();
    const to = moment().toISOString();
    const metricParam = this.selectedMetric() !== 'all' ? this.selectedMetric() : undefined;

    this.telemetryService
      .getHistory(this.deviceId, { from, to, metric: metricParam, limit: 500, order: 'ASC' })
      .subscribe({
        next: (points) => {
          this.isLoading.set(false);
          this.tbOnDataUpdated(points);
        },
        error: () => this.isLoading.set(false),
      });
  }

  private groupPointsByMetric(points: TelemetryPoint[]): Record<string, [number, number][]> {
    const groups: Record<string, [number, number][]> = {};
    points.forEach((pt) => {
      if (!groups[pt.metric_name]) groups[pt.metric_name] = [];
      const timestampMs = moment(pt.time).valueOf();
      groups[pt.metric_name].push([timestampMs, pt.metric_value]);
    });
    return groups;
  }

  private updateMetricsList(metrics: string[]): void {
    const distinct = Array.from(new Set([...this.availableMetrics(), ...metrics]));
    this.availableMetrics.set(distinct);
  }

  /**
   * Builds Apache ECharts option specification for multi-metric visualization.
   */
  private buildChartOption(seriesData: Record<string, [number, number][]>): echarts.EChartsOption {
    const metricNames = Object.keys(seriesData);
    const series = metricNames.map((name, index) => {
      const color = METRIC_COLORS[index % METRIC_COLORS.length];
      return {
        name,
        type: 'line' as const,
        smooth: true,
        showSymbol: false,
        data: seriesData[name],
        itemStyle: { color },
        lineStyle: { width: 2.5, color },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: `${color}40` },
            { offset: 1, color: `${color}00` },
          ]),
        },
      };
    });

    return {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#0f172a',
        borderColor: '#334155',
        textStyle: { color: '#f8fafc' },
        formatter: (params: unknown) => this.formatTooltip(params),
      },
      legend: {
        data: metricNames,
        textStyle: { color: '#94a3b8' },
        top: 0,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        top: '14%',
        containLabel: true,
      },
      xAxis: {
        type: 'time',
        axisLine: { lineStyle: { color: '#334155' } },
        axisLabel: {
          color: '#94a3b8',
          formatter: (value: number) => moment(value).format('HH:mm'),
        },
        splitLine: { show: false },
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: '#334155' } },
        axisLabel: { color: '#94a3b8' },
        splitLine: { lineStyle: { color: '#1e293b' } },
      },
      series,
    };
  }

  private formatTooltip(params: unknown): string {
    if (!Array.isArray(params) || params.length === 0) return '';
    const firstPoint = params[0];
    const timeFormatted = moment(firstPoint.value[0]).format(FORMAT_DATETIME_DISPLAY);
    let html = `<div style="font-weight:600;margin-bottom:4px;color:#38bdf8">${timeFormatted}</div>`;
    params.forEach((item: { seriesName: string; value: [number, number]; color: string }) => {
      html += `<div style="display:flex;align-items:center;gap:6px;font-size:12px">
        <span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${item.color}"></span>
        <span style="color:#94a3b8">${item.seriesName}:</span>
        <span style="font-weight:600;color:#ffffff">${item.value[1]}</span>
      </div>`;
    });
    return html;
  }
}
