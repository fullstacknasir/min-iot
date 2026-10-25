import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { CustomWidgetService } from './custom-widget.service';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import { CustomWidget } from '../models/custom-widget.model';

const BASE_URL = `${DEFAULT_API_BASE_URL}/custom-widgets`;

describe('CustomWidgetService', () => {
  let service: CustomWidgetService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [CustomWidgetService, provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(CustomWidgetService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should fetch list of custom widgets', () => {
    const mockWidgets: CustomWidget[] = [
      {
        id: 'cw-1',
        organization_id: 'org-1',
        name: 'Battery Gauge',
        description: 'Custom battery card',
        widget_type: 'telemetry',
        template_html: '<div>Battery</div>',
        style_css: '.battery { color: red; }',
        controller_js: 'self.onInit = function() {};',
        settings_schema: {},
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

    service.getCustomWidgets().subscribe((widgets) => {
      expect(widgets.length).toBe(1);
      expect(widgets[0].id).toBe('cw-1');
      expect(widgets[0].name).toBe('Battery Gauge');
    });

    const req = httpTesting.expectOne(BASE_URL);
    expect(req.request.method).toBe('GET');
    req.flush({ data: mockWidgets, count: 1 });
  });

  it('should create a custom widget', () => {
    const payload = {
      name: 'New Custom Widget',
      template_html: '<span>Test</span>',
    };

    service.createCustomWidget(payload).subscribe((res) => {
      expect(res.id).toBe('cw-new');
      expect(res.name).toBe('New Custom Widget');
    });

    const req = httpTesting.expectOne(BASE_URL);
    expect(req.request.method).toBe('POST');
    req.flush({
      message: 'Created',
      data: {
        id: 'cw-new',
        organization_id: 'org-1',
        name: 'New Custom Widget',
        description: '',
        widget_type: 'telemetry',
        template_html: '<span>Test</span>',
        style_css: '',
        controller_js: '',
        settings_schema: {},
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    });
  });
});
