import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { MatIconModule } from '@angular/material/icon';
import { of } from 'rxjs';
import { WidgetEditorComponent } from './widget-editor.component';
import { CustomWidgetRunnerComponent } from '../../widgets/custom-widget-runner/custom-widget-runner.component';
import { MonacoEditorComponent } from '../../shared/components/monaco-editor/monaco-editor.component';
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { DeviceService } from '../../core/services/device.service';
import { AuthService } from '../../core/services/auth.service';

describe('WidgetEditorComponent', () => {
  let component: WidgetEditorComponent;
  let fixture: ComponentFixture<WidgetEditorComponent>;
  let customWidgetService: CustomWidgetService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [WidgetEditorComponent, CustomWidgetRunnerComponent, MonacoEditorComponent],
      imports: [FormsModule, MatIconModule],
      providers: [
        CustomWidgetService,
        DeviceService,
        AuthService,
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: Router,
          useValue: { navigate: vi.fn() },
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: () => null,
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(WidgetEditorComponent);
    component = fixture.componentInstance;
    customWidgetService = TestBed.inject(CustomWidgetService);
  });

  it('should initialize and load default ThingsBoard preset', () => {
    fixture.detectChanges();
    expect(component.widgetName).toBe('ThingsBoard Radial Gauge');
    expect(component.templateHtml).toContain('tb-gauge-chart');
    expect(component.controllerJs).toContain('self.onInit');
  });

  it('should switch active editor tabs', () => {
    component.setActiveTab('css');
    expect(component.activeTab()).toBe('css');

    component.setActiveTab('javascript');
    expect(component.activeTab()).toBe('javascript');
  });

  it('should trigger simulated telemetry and dispatch log', () => {
    component.simulatedValue = 42.5;
    component.selectedMetricName = 'pressure';
    component.triggerSimulatedTelemetry();

    expect(component.simulatedData()).toBeTruthy();
    expect(component.simulatedData()?.metrics['pressure']).toBe(42.5);
    expect(component.consoleLogs().length).toBeGreaterThan(0);
  });

  it('should create new custom widget via service', () => {
    const mockCreated = {
      id: 'cw-saved-1',
      organization_id: 'org-1',
      name: 'Test Gauge',
      description: 'Desc',
      widget_type: 'telemetry',
      template_html: '<div>Test</div>',
      style_css: '',
      controller_js: 'self.onInit = function() {};',
      settings_schema: {},
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    vi.spyOn(customWidgetService, 'createCustomWidget').mockReturnValue(of(mockCreated));

    component.widgetName = 'Test Gauge';
    component.handleSaveWidget();

    expect(customWidgetService.createCustomWidget).toHaveBeenCalled();
    expect(component.widgetId).toBe('cw-saved-1');
  });
});
