import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { AppModule } from '../../app.module';
import { DashboardComponent } from './dashboard.component';
import { DashboardService } from '../../core/services/dashboard.service';
import { DeviceService } from '../../core/services/device.service';
import { Dashboard } from '../../core/models/dashboard.model';

describe('DashboardComponent', () => {
  let dashboardService: DashboardService;
  let deviceService: DeviceService;

  const mockDashboard: Dashboard = {
    id: 'dash-1',
    organization_id: 'org-1',
    title: 'Operations Canvas',
    description: 'Main monitoring layout',
    is_default: true,
    configuration: {
      widgets: [
        {
          id: 'w-1',
          type: 'gauge',
          title: 'Boiler Pressure',
          device_id: 'dev-1',
          metric_name: 'pressure',
          col_span: 1,
          row_span: 1,
        },
      ],
    },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideNoopAnimations(),
      ],
    }).compileComponents();

    dashboardService = TestBed.inject(DashboardService);
    deviceService = TestBed.inject(DeviceService);
  });

  it('should initialize and load dashboards on canvas', () => {
    vi.spyOn(dashboardService, 'getDashboards').mockReturnValue(of([mockDashboard]));
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));

    const fixture = TestBed.createComponent(DashboardComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    expect(component.dashboards().length).toBe(1);
    expect(component.selectedDashboard()?.id).toBe('dash-1');
    expect(component.selectedDashboard()?.configuration.widgets.length).toBe(1);
  });

  it('should toggle canvas edit mode', () => {
    vi.spyOn(dashboardService, 'getDashboards').mockReturnValue(of([mockDashboard]));
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));

    const fixture = TestBed.createComponent(DashboardComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    expect(component.isEditMode()).toBe(false);
    component.toggleEditMode();
    expect(component.isEditMode()).toBe(true);
  });

  it('should add a new widget to the selected dashboard configuration', () => {
    vi.spyOn(dashboardService, 'getDashboards').mockReturnValue(of([mockDashboard]));
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));

    const fixture = TestBed.createComponent(DashboardComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.openAddWidgetModal();
    component.newWidgetDeviceId = 'dev-2';
    component.newWidgetType = 'digital_card';
    component.newWidgetMetricName = 'temperature';
    component.newWidgetTitle = 'Temperature KPI';

    component.handleAddWidget();

    const current = component.selectedDashboard();
    expect(current?.configuration.widgets.length).toBe(2);
    expect(current?.configuration.widgets[1].type).toBe('digital_card');
  });

  it('should remove a widget from canvas', () => {
    vi.spyOn(dashboardService, 'getDashboards').mockReturnValue(of([mockDashboard]));
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));

    const fixture = TestBed.createComponent(DashboardComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.handleRemoveWidget('w-1');
    expect(component.selectedDashboard()?.configuration.widgets.length).toBe(0);
  });

  it('should add a custom ThingsBoard widget to the canvas', () => {
    vi.spyOn(dashboardService, 'getDashboards').mockReturnValue(of([mockDashboard]));
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));

    const fixture = TestBed.createComponent(DashboardComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.openAddWidgetModal();
    component.newWidgetDeviceId = 'dev-custom-1';
    component.newWidgetType = 'custom';
    component.newWidgetCustomWidgetId = 'cw-battery-1';
    component.newWidgetTitle = 'Custom Battery Widget';

    component.handleAddWidget();

    const current = component.selectedDashboard();
    expect(current?.configuration.widgets.length).toBe(2);
    expect(current?.configuration.widgets[1].type).toBe('custom');
    expect(current?.configuration.widgets[1].settings?.['customWidgetId']).toBe('cw-battery-1');
  });
});
