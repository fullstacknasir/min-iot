import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { AppModule } from '../../app.module';
import { AlertsComponent } from './alerts.component';
import { AlertService } from '../../core/services/alert.service';
import { Alert } from '../../core/models/rule-alert.model';

describe('AlertsComponent', () => {
  let alertService: AlertService;

  const mockAlert: Alert = {
    id: 'alert-1',
    organization_id: 'org-1',
    rule_id: 'rule-1',
    rule_name: 'High Heat',
    device_id: 'dev-1',
    device_name: 'Temp Sensor 1',
    metric_name: 'temperature',
    metric_value: 42.5,
    threshold_value: 40.0,
    severity: 'critical',
    status: 'active',
    message: 'Temperature exceeded 40.0',
    triggered_at: new Date().toISOString(),
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

    alertService = TestBed.inject(AlertService);
  });

  it('should initialize and load alerts table', () => {
    vi.spyOn(alertService, 'getPaginatedAlerts').mockReturnValue(
      of({
        data: [mockAlert],
        page: 1,
        limit: 10,
        total: 1,
        total_pages: 1,
      })
    );

    const fixture = TestBed.createComponent(AlertsComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    expect(component.alerts().length).toBe(1);
    expect(component.alerts()[0].id).toBe('alert-1');
    expect(component.totalCount()).toBe(1);
  });

  it('should acknowledge an active alert', () => {
    const activeAlert: Alert = { ...mockAlert, status: 'active' };
    vi.spyOn(alertService, 'getPaginatedAlerts').mockReturnValue(
      of({
        data: [activeAlert],
        page: 1,
        limit: 10,
        total: 1,
        total_pages: 1,
      })
    );
    vi.spyOn(alertService, 'acknowledgeAlert').mockReturnValue(of(undefined));

    const fixture = TestBed.createComponent(AlertsComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.handleAcknowledge(activeAlert);
    expect(alertService.acknowledgeAlert).toHaveBeenCalledWith('alert-1');
    expect(activeAlert.status).toBe('acknowledged');
  });

  it('should resolve an alert', () => {
    const activeAlert: Alert = { ...mockAlert, status: 'active' };
    vi.spyOn(alertService, 'getPaginatedAlerts').mockReturnValue(
      of({
        data: [activeAlert],
        page: 1,
        limit: 10,
        total: 1,
        total_pages: 1,
      })
    );
    vi.spyOn(alertService, 'resolveAlert').mockReturnValue(of(undefined));

    const fixture = TestBed.createComponent(AlertsComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.handleResolve(activeAlert);
    expect(alertService.resolveAlert).toHaveBeenCalledWith('alert-1');
    expect(activeAlert.status).toBe('resolved');
  });

  it('should filter alerts by status', () => {
    const spy = vi.spyOn(alertService, 'getPaginatedAlerts').mockReturnValue(
      of({
        data: [],
        page: 1,
        limit: 10,
        total: 0,
        total_pages: 0,
      })
    );

    const fixture = TestBed.createComponent(AlertsComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.setFilterStatus('resolved');
    expect(component.filterStatus()).toBe('resolved');
    expect(spy).toHaveBeenCalledWith(expect.objectContaining({ status: 'resolved' }));
  });
});
