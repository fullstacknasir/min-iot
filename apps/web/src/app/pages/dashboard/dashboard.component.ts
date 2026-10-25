import { Component, OnInit, inject, signal } from '@angular/core';
import { DashboardService } from '../../core/services/dashboard.service';
import { DeviceService } from '../../core/services/device.service';
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { AuthService } from '../../core/services/auth.service';
import { Dashboard, WidgetConfig, WidgetType } from '../../core/models/dashboard.model';
import { CustomWidget } from '../../core/models/custom-widget.model';
import { Device } from '../../core/models/device.model';

const FEEDBACK_TIMEOUT_MS = 3500;
const DEFAULT_METRIC = 'temperature';
const DEFAULT_MIN = 0;
const DEFAULT_MAX = 100;
const MAX_COL_SPAN = 3;
const MIN_COL_SPAN = 1;

@Component({
  selector: 'app-dashboard',
  standalone: false,
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit {
  private readonly dashboardService = inject(DashboardService);
  private readonly deviceService = inject(DeviceService);
  private readonly customWidgetService = inject(CustomWidgetService);
  public readonly authService = inject(AuthService);

  public readonly dashboards = signal<Dashboard[]>([]);
  public readonly selectedDashboard = signal<Dashboard | null>(null);
  public readonly fleetDevices = signal<Device[]>([]);
  public readonly customWidgets = signal<CustomWidget[]>([]);
  public readonly isEditMode = signal<boolean>(false);
  public readonly isLoading = signal<boolean>(false);
  public readonly showAddWidgetModal = signal<boolean>(false);
  public readonly showCreateDashboardModal = signal<boolean>(false);
  public readonly actionMessage = signal<string | null>(null);

  // Widget Form
  public newWidgetType: WidgetType = 'gauge';
  public newWidgetCustomWidgetId = '';
  public newWidgetTitle = '';
  public newWidgetDeviceId = '';
  public newWidgetMetricName = DEFAULT_METRIC;
  public newWidgetColSpan = 1;
  public newWidgetMin = DEFAULT_MIN;
  public newWidgetMax = DEFAULT_MAX;
  public newWidgetUnit = '';

  // Dashboard Form
  public newDashboardTitle = '';
  public newDashboardDescription = '';
  public newDashboardIsDefault = false;

  public ngOnInit(): void {
    this.loadDashboards();
    this.loadFleetDevices();
    this.loadCustomWidgets();
  }

  public loadCustomWidgets(): void {
    this.customWidgetService.getCustomWidgets().subscribe({
      next: (widgets) => {
        this.customWidgets.set(widgets);
        if (widgets.length > 0 && !this.newWidgetCustomWidgetId) {
          this.newWidgetCustomWidgetId = widgets[0].id;
        }
      },
      error: (err) => console.debug('Custom widgets load handled:', err),
    });
  }

  public loadDashboards(): void {
    this.isLoading.set(true);
    this.dashboardService.getDashboards().subscribe({
      next: (list) => {
        this.dashboards.set(list);
        if (list.length > 0 && !this.selectedDashboard()) {
          const defaultDash = list.find((d) => d.is_default) ?? list[0];
          this.selectedDashboard.set(defaultDash);
        }
        this.isLoading.set(false);
      },
      error: () => this.isLoading.set(false),
    });
  }

  public loadFleetDevices(): void {
    this.deviceService.getDevices({ all: true }).subscribe({
      next: (devs) => {
        this.fleetDevices.set(devs);
        if (devs.length > 0 && !this.newWidgetDeviceId) {
          this.newWidgetDeviceId = devs[0].id;
        }
      },
      error: (err) => console.warn('Failed loading devices for canvas:', err),
    });
  }

  public selectDashboard(d: Dashboard): void {
    this.selectedDashboard.set(d);
    this.isEditMode.set(false);
  }

  public onDashboardSelectChange(id: string): void {
    const found = this.dashboards().find((d) => d.id === id);
    if (found) {
      this.selectDashboard(found);
    }
  }

  public toggleEditMode(): void {
    this.isEditMode.update((v) => !v);
  }

  public openCreateDashboardModal(): void {
    this.newDashboardTitle = '';
    this.newDashboardDescription = '';
    this.newDashboardIsDefault = this.dashboards().length === 0;
    this.showCreateDashboardModal.set(true);
  }

  public closeCreateDashboardModal(): void {
    this.showCreateDashboardModal.set(false);
  }

  public handleCreateDashboard(): void {
    if (!this.newDashboardTitle.trim()) return;

    this.dashboardService
      .createDashboard({
        title: this.newDashboardTitle.trim(),
        description: this.newDashboardDescription.trim(),
        is_default: this.newDashboardIsDefault,
        configuration: { widgets: [] },
      })
      .subscribe({
        next: (created) => {
          this.closeCreateDashboardModal();
          this.dashboards.update((list) => [created, ...list]);
          this.selectedDashboard.set(created);
          this.isEditMode.set(true);
          this.setActionFeedback('Dashboard created! Now add custom widgets to canvas.');
        },
        error: (err) => alert(err?.error?.error ?? 'Failed to create dashboard.'),
      });
  }

  public handleDeleteDashboard(dash: Dashboard, event?: MouseEvent): void {
    event?.stopPropagation();
    if (!confirm(`Delete dashboard "${dash.title}"?`)) return;

    this.dashboardService.deleteDashboard(dash.id).subscribe({
      next: () => {
        this.dashboards.update((list) => list.filter((d) => d.id !== dash.id));
        const remaining = this.dashboards();
        this.selectedDashboard.set(remaining.length > 0 ? remaining[0] : null);
        this.setActionFeedback('Dashboard deleted successfully.');
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to delete dashboard.'),
    });
  }

  public handleSetDefault(dash: Dashboard, event?: MouseEvent): void {
    event?.stopPropagation();
    this.dashboardService.setDefaultDashboard(dash.id).subscribe({
      next: () => {
        this.dashboards.update((list) =>
          list.map((d) => ({ ...d, is_default: d.id === dash.id }))
        );
        if (this.selectedDashboard()?.id === dash.id) {
          this.selectedDashboard.update((curr) => curr ? { ...curr, is_default: true } : null);
        }
        this.setActionFeedback(`"${dash.title}" set as default dashboard.`);
      },
      error: (err) => alert(err?.error?.error ?? 'Failed to set default dashboard.'),
    });
  }

  public openAddWidgetModal(): void {
    this.loadCustomWidgets();
    this.resetWidgetForm();
    this.showAddWidgetModal.set(true);
  }

  public closeAddWidgetModal(): void {
    this.showAddWidgetModal.set(false);
  }

  public handleAddWidget(): void {
    const current = this.selectedDashboard();
    if (!current || !this.newWidgetDeviceId) return;

    const targetDev = this.fleetDevices().find((d) => d.id === this.newWidgetDeviceId);
    const targetCustomWidget = this.customWidgets().find((cw) => cw.id === this.newWidgetCustomWidgetId);

    const defaultTitle = this.newWidgetType === 'custom'
      ? (targetCustomWidget?.name ?? 'Custom Widget')
      : `${targetDev?.name ?? this.newWidgetDeviceId} ${this.newWidgetType}`;

    const newWidget: WidgetConfig = {
      id: `w-${Date.now()}`,
      type: this.newWidgetType,
      title: this.newWidgetTitle.trim() || defaultTitle,
      device_id: this.newWidgetDeviceId,
      device_name: targetDev?.name ?? this.newWidgetDeviceId,
      metric_name: this.newWidgetMetricName.trim(),
      col_span: Number(this.newWidgetColSpan) || 1,
      row_span: 1,
      settings: {
        min: Number(this.newWidgetMin),
        max: Number(this.newWidgetMax),
        unit: this.newWidgetUnit.trim(),
        customWidgetId: this.newWidgetType === 'custom' ? this.newWidgetCustomWidgetId : undefined,
      },
    };

    const updatedWidgets = [...(current.configuration?.widgets ?? []), newWidget];
    this.updateCurrentDashboardWidgets(updatedWidgets, 'Widget added to canvas.');
    this.closeAddWidgetModal();
  }

  public handleRemoveWidget(widgetId: string): void {
    const current = this.selectedDashboard();
    if (!current) return;

    const filtered = (current.configuration?.widgets ?? []).filter((w) => w.id !== widgetId);
    this.updateCurrentDashboardWidgets(filtered, 'Widget removed from canvas.');
  }

  public handleMoveWidget(index: number, direction: 'left' | 'right'): void {
    const current = this.selectedDashboard();
    if (!current) return;

    const widgets = [...(current.configuration?.widgets ?? [])];
    const targetIdx = direction === 'left' ? index - 1 : index + 1;
    if (targetIdx < 0 || targetIdx >= widgets.length) return;

    const temp = widgets[index];
    widgets[index] = widgets[targetIdx];
    widgets[targetIdx] = temp;

    this.updateCurrentDashboardWidgets(widgets);
  }

  public handleToggleColSpan(widget: WidgetConfig): void {
    widget.col_span = widget.col_span >= MAX_COL_SPAN ? MIN_COL_SPAN : widget.col_span + 1;
    const current = this.selectedDashboard();
    if (!current) return;
    this.updateCurrentDashboardWidgets(current.configuration?.widgets ?? []);
  }

  public handleSaveLayout(): void {
    const current = this.selectedDashboard();
    if (!current) return;

    this.dashboardService
      .updateDashboard(current.id, {
        title: current.title,
        description: current.description,
        configuration: current.configuration,
      })
      .subscribe({
        next: () => {
          this.isEditMode.set(false);
          this.setActionFeedback('Dashboard layout saved successfully.');
        },
        error: (err) => alert(err?.error?.error ?? 'Failed saving layout.'),
      });
  }

  public getColSpanClass(span: number): string {
    switch (span) {
      case 2:
        return 'col-span-1 md:col-span-2';
      case 3:
        return 'col-span-1 md:col-span-2 lg:col-span-3';
      default:
        return 'col-span-1';
    }
  }

  private updateCurrentDashboardWidgets(widgets: WidgetConfig[], feedbackMsg?: string): void {
    const current = this.selectedDashboard();
    if (!current) return;

    const updatedDash = {
      ...current,
      configuration: { widgets },
    };
    this.selectedDashboard.set(updatedDash);
    this.dashboards.update((list) =>
      list.map((d) => (d.id === current.id ? updatedDash : d))
    );
    if (feedbackMsg) {
      this.setActionFeedback(feedbackMsg);
    }
  }

  private resetWidgetForm(): void {
    this.newWidgetType = 'gauge';
    this.newWidgetCustomWidgetId = this.customWidgets()[0]?.id ?? '';
    this.newWidgetTitle = '';
    this.newWidgetMetricName = DEFAULT_METRIC;
    this.newWidgetColSpan = 1;
    this.newWidgetMin = DEFAULT_MIN;
    this.newWidgetMax = DEFAULT_MAX;
    this.newWidgetUnit = '';
  }

  private setActionFeedback(msg: string): void {
    this.actionMessage.set(msg);
    setTimeout(() => this.actionMessage.set(null), FEEDBACK_TIMEOUT_MS);
  }
}
