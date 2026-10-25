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

const FEEDBACK_TIMEOUT_MS = 2500;
const DEFAULT_METRIC = 'relay';
const DEFAULT_COMMAND = 'toggle_relay';

@Component({
  selector: 'app-telemetry-switch-widget',
  standalone: false,
  templateUrl: './telemetry-switch-widget.component.html',
})
export class TelemetrySwitchWidgetComponent implements OnInit, OnDestroy, OnChanges {
  @Input({ required: true }) public deviceId!: string;
  @Input() public title?: string;
  @Input() public metricName: string = DEFAULT_METRIC;
  @Input() public commandName: string = DEFAULT_COMMAND;

  private readonly telemetryService = inject(TelemetryService);
  private liveStreamSubscription?: Subscription;

  public readonly stateActive = signal<boolean>(false);
  public readonly isPending = signal<boolean>(false);
  public readonly actionFeedback = signal<string | null>(null);
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
    if (typeof rawVal === 'boolean') {
      this.stateActive.set(rawVal);
    } else if (typeof rawVal === 'number') {
      this.stateActive.set(rawVal > 0);
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

  public handleToggle(): void {
    const newState = !this.stateActive();
    this.isPending.set(true);

    // Optimistically update and dispatch state change
    setTimeout(() => {
      this.stateActive.set(newState);
      this.isPending.set(false);
      this.showFeedback(`Command "${this.commandName}" sent: ${newState ? 'ON' : 'OFF'}`);
    }, 400);
  }

  private showFeedback(msg: string): void {
    this.actionFeedback.set(msg);
    setTimeout(() => this.actionFeedback.set(null), FEEDBACK_TIMEOUT_MS);
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
