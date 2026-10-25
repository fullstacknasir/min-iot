import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { CustomWidgetRunnerComponent } from './custom-widget-runner.component';
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { TelemetryService } from '../../core/services/telemetry.service';
import { AuthService } from '../../core/services/auth.service';
import { MatIconModule } from '@angular/material/icon';
import { of } from 'rxjs';

describe('CustomWidgetRunnerComponent', () => {
  let component: CustomWidgetRunnerComponent;
  let fixture: ComponentFixture<CustomWidgetRunnerComponent>;
  let customWidgetService: CustomWidgetService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [CustomWidgetRunnerComponent],
      imports: [MatIconModule],
      providers: [
        CustomWidgetService,
        TelemetryService,
        AuthService,
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CustomWidgetRunnerComponent);
    component = fixture.componentInstance;
    customWidgetService = TestBed.inject(CustomWidgetService);
  });

  it('should initialize and execute ThingsBoard custom widget lifecycle', () => {
    let initCalled = false;
    let dataUpdatedCalled = false;

    component.liveCodeOverride = {
      template_html: '<div id="test-metric">0</div>',
      style_css: '#test-metric { font-size: 20px; }',
      controller_js: `
        self.onInit = function() {
          const el = self.ctx.$container.querySelector('#test-metric');
          if (el) el.textContent = 'ready';
        };
        self.onDataUpdated = function(telemetry) {
          const el = self.ctx.$container.querySelector('#test-metric');
          if (el) el.textContent = 'updated';
        };
      `,
    };

    fixture.detectChanges();

    const host = component.widgetHost.nativeElement;
    const metricEl = host.querySelector('#test-metric');
    expect(metricEl).toBeTruthy();
    expect(metricEl?.textContent).toBe('ready');

    // Simulate incoming telemetry data
    component.tbOnDataUpdated({
      device_id: 'dev-1',
      metrics: { temperature: 24.5 },
      timestamp: new Date().toISOString(),
    });

    expect(metricEl?.textContent).toBe('updated');
  });

  it('should handle runtime compilation errors gracefully', () => {
    component.liveCodeOverride = {
      template_html: '<div>Error Test</div>',
      style_css: '',
      controller_js: 'syntax error in javascript $$$###',
    };

    fixture.detectChanges();
    expect(component.executionError()).toContain('[ThingsBoard Lifecycle compile]');
  });

  it('should load widget by id from CustomWidgetService', () => {
    const mockWidget = {
      id: 'cw-123',
      organization_id: 'org-1',
      name: 'Custom Flow',
      description: '',
      widget_type: 'telemetry',
      template_html: '<span id="flow">flow</span>',
      style_css: '',
      controller_js: 'self.onInit = function() {};',
      settings_schema: {},
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    vi.spyOn(customWidgetService, 'getCustomWidget').mockReturnValue(of(mockWidget));

    component.widgetId = 'cw-123';
    component.tbOnInit();

    expect(customWidgetService.getCustomWidget).toHaveBeenCalledWith('cw-123');
    expect(component.activeWidget()?.name).toBe('Custom Flow');
  });
});
