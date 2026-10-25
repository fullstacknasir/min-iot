import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { Router } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { of } from 'rxjs';
import { WidgetLibraryComponent } from './widget-library.component';
import { CustomWidgetRunnerComponent } from '../../widgets/custom-widget-runner/custom-widget-runner.component';
import { CustomWidgetService } from '../../core/services/custom-widget.service';
import { AuthService } from '../../core/services/auth.service';
import { CustomWidget } from '../../core/models/custom-widget.model';

describe('WidgetLibraryComponent', () => {
  let component: WidgetLibraryComponent;
  let fixture: ComponentFixture<WidgetLibraryComponent>;
  let customWidgetService: CustomWidgetService;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  const mockList: CustomWidget[] = [
    {
      id: 'cw-1',
      organization_id: 'org-1',
      name: 'Custom Battery',
      description: 'Battery widget',
      widget_type: 'telemetry',
      template_html: '<div>Battery</div>',
      style_css: '',
      controller_js: 'self.onInit = function() {};',
      settings_schema: {},
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  ];

  beforeEach(async () => {
    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      declarations: [WidgetLibraryComponent, CustomWidgetRunnerComponent],
      imports: [MatIconModule],
      providers: [
        CustomWidgetService,
        AuthService,
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(WidgetLibraryComponent);
    component = fixture.componentInstance;
    customWidgetService = TestBed.inject(CustomWidgetService);
  });

  it('should load custom widgets on init', () => {
    vi.spyOn(customWidgetService, 'getCustomWidgets').mockReturnValue(of(mockList));

    component.ngOnInit();

    expect(customWidgetService.getCustomWidgets).toHaveBeenCalled();
    expect(component.widgets().length).toBe(1);
    expect(component.widgets()[0].name).toBe('Custom Battery');
  });

  it('should navigate to editor when createNewWidget is called', () => {
    component.createNewWidget();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/widgets/editor']);
  });

  it('should clone widget and add to list', () => {
    const cloned = { ...mockList[0], id: 'cw-clone', name: 'Custom Battery (Copy)' };
    vi.spyOn(customWidgetService, 'createCustomWidget').mockReturnValue(of(cloned));

    component.widgets.set(mockList);
    component.cloneWidget(mockList[0]);

    expect(customWidgetService.createCustomWidget).toHaveBeenCalled();
    expect(component.widgets().length).toBe(2);
    expect(component.widgets()[0].name).toBe('Custom Battery (Copy)');
  });
});
