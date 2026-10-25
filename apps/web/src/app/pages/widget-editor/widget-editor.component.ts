import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import moment from 'moment';
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { DeviceService } from '../../core/services/device.service';
import { AuthService } from '../../core/services/auth.service';
import { Device } from '../../core/models/device.model';
import { LatestTelemetryData } from '../../core/models/telemetry.model';
import { RunnerLogEvent } from '../../widgets/custom-widget-runner/custom-widget-runner.component';
import { WIDGET_PRESETS, WidgetPreset } from '../../core/constants/widget-presets.constants';

type EditorTab = 'html' | 'css' | 'javascript' | 'settings';

const FEEDBACK_DURATION_MS = 4000;
const DEFAULT_SIMULATED_METRIC_KEY = 'temperature';
const DEFAULT_SIMULATED_VALUE = 25.4;

@Component({
  selector: 'app-widget-editor',
  standalone: false,
  templateUrl: './widget-editor.component.html',
})
export class WidgetEditorComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly customWidgetService = inject(CustomWidgetService);
  private readonly deviceService = inject(DeviceService);
  public readonly authService = inject(AuthService);

  public readonly presets: WidgetPreset[] = WIDGET_PRESETS;
  public readonly activeTab = signal<EditorTab>('html');
  public readonly isSaving = signal<boolean>(false);
  public readonly isLoading = signal<boolean>(false);
  public readonly actionMessage = signal<string | null>(null);
  public readonly errorMessage = signal<string | null>(null);
  public readonly showDocs = signal<boolean>(false);
  public readonly showConsole = signal<boolean>(true);
  public readonly consoleLogs = signal<RunnerLogEvent[]>([]);

  // Devices for preview
  public readonly fleetDevices = signal<Device[]>([]);
  public selectedDeviceId = '';
  public selectedMetricName = DEFAULT_SIMULATED_METRIC_KEY;

  // Simulation mode
  public simulatedValue = DEFAULT_SIMULATED_VALUE;
  public simulatedData = signal<LatestTelemetryData | null>(null);

  // Widget Metadata
  public widgetId: string | null = null;
  public widgetName = 'My Custom ThingsBoard Widget';
  public widgetDescription = 'Custom ThingsBoard IoT visual component';
  public widgetType = 'telemetry';

  // Code Buffers
  public templateHtml = '';
  public styleCss = '';
  public controllerJs = '';
  public settingsJson = '{\n  "title": "My Metric",\n  "unit": "°C"\n}';

  // Live execution override signal
  public liveCode = signal<{
    template_html: string;
    style_css: string;
    controller_js: string;
  }>({
    template_html: '',
    style_css: '',
    controller_js: '',
  });

  public ngOnInit(): void {
    this.loadFleetDevices();
    this.checkRouteForWidgetId();
  }

  public setActiveTab(tab: EditorTab): void {
    this.activeTab.set(tab);
  }

  public toggleDocs(): void {
    this.showDocs.update((v) => !v);
  }

  public toggleConsole(): void {
    this.showConsole.update((v) => !v);
  }

  public clearConsole(): void {
    this.consoleLogs.set([]);
  }

  public onRunnerLog(event: RunnerLogEvent): void {
    this.consoleLogs.update((logs) => [event, ...logs].slice(0, 50));
  }

  public handleCodeChange(): void {
    this.liveCode.set({
      template_html: this.templateHtml,
      style_css: this.styleCss,
      controller_js: this.controllerJs,
    });
  }

  public loadPreset(presetId: string): void {
    const found = this.presets.find((p) => p.id === presetId);
    if (!found) return;

    this.widgetName = found.name;
    this.widgetDescription = found.description;
    this.widgetType = found.widget_type;
    this.templateHtml = found.template_html;
    this.styleCss = found.style_css;
    this.controllerJs = found.controller_js;
    this.settingsJson = JSON.stringify(found.settings_schema, null, 2);

    this.handleCodeChange();
    this.setSuccessMessage(`Preset "${found.name}" loaded into editor.`);
  }

  public triggerSimulatedTelemetry(): void {
    const mockTelemetry: LatestTelemetryData = {
      device_id: this.selectedDeviceId || 'simulated-device',
      metrics: {
        [this.selectedMetricName]: Number(this.simulatedValue),
        battery: 85,
        rssi: -62,
        voltage: 3.3,
      },
      timestamp: new Date().toISOString(),
      received_at: new Date().toISOString(),
    };

    this.simulatedData.set(mockTelemetry);
    this.onRunnerLog({
      level: 'log',
      message: `Dispatched simulated telemetry: ${this.selectedMetricName} = ${this.simulatedValue}`,
      timestamp: moment().format('HH:mm:ss.SSS'),
    });
  }

  public handleSaveWidget(): void {
    if (!this.widgetName.trim()) {
      this.setErrorMessage('Widget name cannot be empty.');
      return;
    }

    let parsedSettings = {};
    try {
      if (this.settingsJson.trim()) {
        parsedSettings = JSON.parse(this.settingsJson);
      }
    } catch {
      this.setErrorMessage('Invalid JSON format in Settings Schema.');
      return;
    }

    this.isSaving.set(true);
    const payload = {
      name: this.widgetName.trim(),
      description: this.widgetDescription.trim(),
      widget_type: this.widgetType,
      template_html: this.templateHtml,
      style_css: this.styleCss,
      controller_js: this.controllerJs,
      settings_schema: parsedSettings,
    };

    if (this.widgetId) {
      this.customWidgetService.updateCustomWidget(this.widgetId, payload).subscribe({
        next: () => {
          this.isSaving.set(false);
          this.setSuccessMessage('Custom widget updated successfully!');
        },
        error: (err) => {
          this.isSaving.set(false);
          this.setErrorMessage(err?.error?.error ?? 'Failed to update custom widget.');
        },
      });
      return;
    }

    this.customWidgetService.createCustomWidget(payload).subscribe({
      next: (created) => {
        this.isSaving.set(false);
        this.widgetId = created.id;
        this.setSuccessMessage('Custom widget created and registered!');
        this.router.navigate(['/widgets/editor', created.id], { replaceUrl: true });
      },
      error: (err) => {
        this.isSaving.set(false);
        this.setErrorMessage(err?.error?.error ?? 'Failed creating custom widget.');
      },
    });
  }

  public navigateToLibrary(): void {
    this.router.navigate(['/widgets']);
  }

  private checkRouteForWidgetId(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.widgetId = id;
      this.loadWidgetData(id);
      return;
    }
    // Load default preset on new
    this.loadPreset('radial_gauge');
  }

  private loadWidgetData(id: string): void {
    this.isLoading.set(true);
    this.customWidgetService.getCustomWidget(id).subscribe({
      next: (w) => {
        this.isLoading.set(false);
        this.widgetName = w.name;
        this.widgetDescription = w.description;
        this.widgetType = w.widget_type;
        this.templateHtml = w.template_html;
        this.styleCss = w.style_css;
        this.controllerJs = w.controller_js;
        this.settingsJson = JSON.stringify(w.settings_schema ?? {}, null, 2);
        this.handleCodeChange();
      },
      error: (err) => {
        this.isLoading.set(false);
        this.setErrorMessage(err?.error?.error ?? 'Failed loading custom widget.');
      },
    });
  }

  private loadFleetDevices(): void {
    this.deviceService.getDevices({ all: true }).subscribe({
      next: (devices) => {
        this.fleetDevices.set(devices);
        if (devices.length > 0 && !this.selectedDeviceId) {
          this.selectedDeviceId = devices[0].id;
        }
      },
      error: (err) => console.debug('Preview devices load handled:', err),
    });
  }

  private setSuccessMessage(msg: string): void {
    this.errorMessage.set(null);
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), FEEDBACK_DURATION_MS);
  }

  private setErrorMessage(msg: string): void {
    this.actionMessage.set(null);
    this.errorMessage.set(msg);
    setTimeout(() => this.errorMessage.set(null), FEEDBACK_DURATION_MS);
  }
}
