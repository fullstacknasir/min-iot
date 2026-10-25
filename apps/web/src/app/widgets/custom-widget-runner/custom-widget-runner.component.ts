import {
  Component,
  Input,
  Output,
  EventEmitter,
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
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { AuthService } from '../../core/services/auth.service';
import { LatestTelemetryData } from '../../core/models/telemetry.model';
import {
  CustomWidget,
  ThingsBoardWidgetContext,
  ThingsBoardWidgetInstance,
} from '../../core/models/custom-widget.model';
import { FORMAT_DATETIME_DISPLAY } from '../../core/constants/app.constants';

const SCOPE_PREFIX = 'tb-custom-';
const DEFAULT_METRIC_KEY = 'temperature';

export interface RunnerLogEvent {
  level: 'log' | 'warn' | 'error';
  message: string;
  timestamp: string;
}

@Component({
  selector: 'app-custom-widget-runner',
  standalone: false,
  templateUrl: './custom-widget-runner.component.html',
})
export class CustomWidgetRunnerComponent implements OnInit, OnDestroy, OnChanges {
  @Input() public widgetId?: string;
  @Input() public customWidget?: CustomWidget;
  @Input() public deviceId?: string;
  @Input() public metricName: string = DEFAULT_METRIC_KEY;
  @Input() public customSettings?: Record<string, unknown>;
  @Input() public simulatedData?: LatestTelemetryData | null;
  @Input() public liveCodeOverride?: {
    template_html: string;
    style_css: string;
    controller_js: string;
  };

  @Output() public readonly runnerLog = new EventEmitter<RunnerLogEvent>();

  @ViewChild('widgetHost', { static: true }) public widgetHost!: ElementRef<HTMLDivElement>;

  private readonly telemetryService = inject(TelemetryService);
  private readonly customWidgetService = inject(CustomWidgetService);
  private readonly authService = inject(AuthService);

  private liveSubscription?: Subscription;
  private resizeObserver?: ResizeObserver;
  private widgetInstance?: ThingsBoardWidgetInstance;
  private styleElement?: HTMLStyleElement;
  private readonly uniqueScopeId = `${SCOPE_PREFIX}${Math.random().toString(36).substring(2, 9)}`;

  public readonly activeWidget = signal<CustomWidget | null>(null);
  public readonly isLoading = signal<boolean>(false);
  public readonly executionError = signal<string | null>(null);
  public readonly isConnected = signal<boolean>(false);
  public readonly lastReceivedTime = signal<string | null>(null);

  public ngOnInit(): void {
    this.tbOnInit();
  }

  public ngOnChanges(changes: SimpleChanges): void {
    const codeChanged = changes['liveCodeOverride'] && !changes['liveCodeOverride']?.isFirstChange();
    const widgetChanged = changes['widgetId'] || changes['customWidget'];
    const deviceChanged = changes['deviceId'] && !changes['deviceId']?.isFirstChange();

    if (codeChanged || widgetChanged || deviceChanged) {
      this.tbOnDestroy();
      this.tbOnInit();
      return;
    }

    if (changes['simulatedData'] && this.simulatedData) {
      this.tbOnDataUpdated(this.simulatedData);
    }
  }

  public ngOnDestroy(): void {
    this.tbOnDestroy();
  }

  @HostListener('window:resize')
  public handleWindowResize(): void {
    this.tbOnResize();
  }

  /**
   * ThingsBoard Widget Lifecycle: onInit
   */
  public tbOnInit(): void {
    this.executionError.set(null);
    if (this.liveCodeOverride) {
      this.initializeLiveRuntime(this.liveCodeOverride);
      return;
    }

    if (this.customWidget) {
      this.activeWidget.set(this.customWidget);
      this.initializeRuntime(this.customWidget);
      return;
    }

    if (this.widgetId) {
      this.loadWidgetFromApi(this.widgetId);
      return;
    }
  }

  /**
   * ThingsBoard Widget Lifecycle: onDataUpdated
   */
  public tbOnDataUpdated(telemetry: LatestTelemetryData): void {
    if (!telemetry || !this.widgetInstance) return;

    this.widgetInstance.ctx.data = telemetry;
    const ts = telemetry.received_at ?? telemetry.timestamp ?? new Date().toISOString();
    this.lastReceivedTime.set(moment(ts).format(FORMAT_DATETIME_DISPLAY));

    try {
      this.widgetInstance.onDataUpdated?.(telemetry);
    } catch (err: unknown) {
      this.handleRuntimeError('onDataUpdated', err);
    }
  }

  /**
   * ThingsBoard Widget Lifecycle: onResize
   */
  public tbOnResize(): void {
    if (!this.widgetInstance?.onResize) return;
    try {
      this.widgetInstance.onResize();
    } catch (err: unknown) {
      this.handleRuntimeError('onResize', err);
    }
  }

  /**
   * ThingsBoard Widget Lifecycle: onDestroy
   */
  public tbOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.liveSubscription?.unsubscribe();

    if (this.widgetInstance?.onDestroy) {
      try {
        this.widgetInstance.onDestroy();
      } catch (err: unknown) {
        console.debug('Custom widget onDestroy error handled:', err);
      }
    }

    this.widgetInstance = undefined;
    this.cleanupScopedStyles();
    this.clearHostContainer();
    this.isConnected.set(false);
  }

  /**
   * Fetches widget definition from database by ID
   */
  private loadWidgetFromApi(id: string): void {
    this.isLoading.set(true);
    this.customWidgetService.getCustomWidget(id).subscribe({
      next: (widget) => {
        this.isLoading.set(false);
        this.activeWidget.set(widget);
        this.initializeRuntime(widget);
      },
      error: (err) => {
        this.isLoading.set(false);
        this.executionError.set(err?.error?.error ?? 'Failed loading custom widget definition.');
      },
    });
  }

  /**
   * Mounts widget template, injects styles, and initializes controller
   */
  private initializeRuntime(widget: CustomWidget): void {
    this.initializeLiveRuntime({
      template_html: widget.template_html,
      style_css: widget.style_css,
      controller_js: widget.controller_js,
    });
  }

  /**
   * Compiles and executes live code payload directly
   */
  private initializeLiveRuntime(code: {
    template_html: string;
    style_css: string;
    controller_js: string;
  }): void {
    if (!this.widgetHost?.nativeElement) return;

    this.clearHostContainer();
    this.injectScopedCss(code.style_css);
    this.injectHtmlTemplate(code.template_html);

    const context = this.createWidgetContext();
    this.widgetInstance = this.compileController(code.controller_js, context);

    this.setupResizeObserver();
    this.executeOnInit();
    this.connectTelemetryDataFeed();
  }

  /**
   * Constructs the ThingsBoard widget context (self.ctx)
   */
  private createWidgetContext(): ThingsBoardWidgetContext {
    const profile = this.authService.currentUser();
    const effectiveSettings = {
      ...(this.activeWidget()?.settings_schema ?? {}),
      ...(this.customSettings ?? {}),
    };

    return {
      $container: this.widgetHost.nativeElement,
      data: this.simulatedData ?? null,
      datasources: this.deviceId ? [{ deviceId: this.deviceId, metricName: this.metricName }] : [],
      settings: effectiveSettings,
      moment,
      echarts,
      currentUser: profile ? { id: profile.id, email: profile.email, role: profile.role } : undefined,
    };
  }

  /**
   * Compiles user-provided JavaScript with isolated execution scope
   */
  private compileController(
    controllerCode: string,
    context: ThingsBoardWidgetContext
  ): ThingsBoardWidgetInstance {
    const instance: ThingsBoardWidgetInstance = {
      ctx: context,
    };

    if (!controllerCode || !controllerCode.trim()) return instance;

    const wrappedExecution = `
      "use strict";
      return function(self, ctx, moment, echarts) {
        ${controllerCode}
      };
    `;

    try {
      const compiledFn = new Function(wrappedExecution)();
      compiledFn(instance, context, moment, echarts);
      this.emitLog('log', 'Widget controller compiled and attached successfully.');
    } catch (err: unknown) {
      this.handleRuntimeError('compile', err);
    }

    return instance;
  }

  /**
   * Safely invokes self.onInit()
   */
  private executeOnInit(): void {
    if (!this.widgetInstance?.onInit) return;
    try {
      this.widgetInstance.onInit();
      this.emitLog('log', 'Widget onInit executed.');
    } catch (err: unknown) {
      this.handleRuntimeError('onInit', err);
    }
  }

  /**
   * Connects SSE stream or fetches initial telemetry if deviceId is present
   */
  private connectTelemetryDataFeed(): void {
    if (this.simulatedData) {
      this.tbOnDataUpdated(this.simulatedData);
      return;
    }
    if (!this.deviceId) return;

    this.telemetryService.getLatestTelemetry(this.deviceId).subscribe({
      next: (telemetry) => {
        if (telemetry) this.tbOnDataUpdated(telemetry);
      },
      error: (err) => console.debug('Initial telemetry fetch skipped:', err),
    });

    this.liveSubscription = this.telemetryService.connectLiveStream(this.deviceId).subscribe({
      next: (telemetry) => {
        this.isConnected.set(true);
        this.tbOnDataUpdated(telemetry);
      },
      error: () => this.isConnected.set(false),
    });
  }

  /**
   * Renders HTML template inside the scoped container
   */
  private injectHtmlTemplate(html: string): void {
    const container = this.widgetHost.nativeElement;
    container.classList.add(this.uniqueScopeId);
    container.innerHTML = html || '';
  }

  /**
   * Scopes user CSS to the unique widget instance
   */
  private injectScopedCss(css: string): void {
    this.cleanupScopedStyles();
    if (!css || !css.trim()) return;

    this.styleElement = document.createElement('style');
    this.styleElement.setAttribute('data-widget-scope', this.uniqueScopeId);
    // Scope selectors to only match within this widget container
    const scopedCss = `.${this.uniqueScopeId} { ${css} }`;
    this.styleElement.textContent = scopedCss;
    document.head.appendChild(this.styleElement);
  }

  /**
   * Removes injected stylesheet from document head
   */
  private cleanupScopedStyles(): void {
    if (this.styleElement?.parentNode) {
      this.styleElement.parentNode.removeChild(this.styleElement);
      this.styleElement = undefined;
    }
  }

  /**
   * Clears inner container content
   */
  private clearHostContainer(): void {
    if (this.widgetHost?.nativeElement) {
      this.widgetHost.nativeElement.innerHTML = '';
    }
  }

  /**
   * Observes container element bounding changes for ThingsBoard onResize()
   */
  private setupResizeObserver(): void {
    if (typeof ResizeObserver === 'undefined' || !this.widgetHost?.nativeElement) return;
    this.resizeObserver = new ResizeObserver(() => this.tbOnResize());
    this.resizeObserver.observe(this.widgetHost.nativeElement);
  }

  /**
   * Handles and records runtime errors gracefully
   */
  private handleRuntimeError(phase: string, err: unknown): void {
    const message = err instanceof Error ? err.message : String(err);
    const formatted = `[ThingsBoard Lifecycle ${phase}] ${message}`;
    this.executionError.set(formatted);
    this.emitLog('error', formatted);
    console.error(formatted, err);
  }

  /**
   * Dispatches log events for parent panels or sandboxes
   */
  private emitLog(level: 'log' | 'warn' | 'error', message: string): void {
    this.runnerLog.emit({
      level,
      message,
      timestamp: moment().format('HH:mm:ss.SSS'),
    });
  }
}
