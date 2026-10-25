import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { AppModule } from '../../app.module';
import { RulesComponent } from './rules.component';
import { RuleService } from '../../core/services/rule.service';
import { DeviceService } from '../../core/services/device.service';
import { Rule } from '../../core/models/rule-alert.model';

describe('RulesComponent', () => {
  let ruleService: RuleService;
  let deviceService: DeviceService;

  const mockRule: Rule = {
    id: 'rule-1',
    organization_id: 'org-1',
    name: 'Overheat Detection',
    description: 'Alert when engine temp exceeds 80C',
    metric_name: 'temperature',
    condition: '>',
    threshold: 80.0,
    severity: 'critical',
    cooldown_seconds: 120,
    notification_channel: 'in_app',
    webhook_url: '',
    enabled: true,
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

    ruleService = TestBed.inject(RuleService);
    deviceService = TestBed.inject(DeviceService);
  });

  it('should initialize and load rules list and devices', () => {
    vi.spyOn(ruleService, 'getPaginatedRules').mockReturnValue(
      of({
        data: [mockRule],
        page: 1,
        limit: 10,
        total: 1,
        total_pages: 1,
      })
    );
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));

    const fixture = TestBed.createComponent(RulesComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    expect(component.rules().length).toBe(1);
    expect(component.rules()[0].name).toBe('Overheat Detection');
    expect(component.totalCount()).toBe(1);
  });

  it('should toggle rule enabled status', () => {
    vi.spyOn(ruleService, 'getPaginatedRules').mockReturnValue(
      of({
        data: [mockRule],
        page: 1,
        limit: 10,
        total: 1,
        total_pages: 1,
      })
    );
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));
    vi.spyOn(ruleService, 'toggleRule').mockReturnValue(
      of({ id: 'rule-1', enabled: false })
    );

    const fixture = TestBed.createComponent(RulesComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    const targetRule = { ...mockRule, enabled: true };
    component.handleToggle(targetRule);
    expect(ruleService.toggleRule).toHaveBeenCalledWith('rule-1');
    expect(targetRule.enabled).toBe(false);
  });

  it('should create a new rule when valid form is submitted', () => {
    vi.spyOn(ruleService, 'getPaginatedRules').mockReturnValue(
      of({
        data: [],
        page: 1,
        limit: 10,
        total: 0,
        total_pages: 0,
      })
    );
    vi.spyOn(deviceService, 'getDevices').mockReturnValue(of([]));
    vi.spyOn(ruleService, 'createRule').mockReturnValue(of(mockRule));

    const fixture = TestBed.createComponent(RulesComponent);
    const component = fixture.componentInstance;
    fixture.detectChanges();

    component.openCreateModal();
    component.formName = 'New Temp Rule';
    component.formMetricName = 'temperature';
    component.formCondition = '>=';
    component.formThreshold = 90;
    component.formSeverity = 'critical';

    component.handleSaveRule();
    expect(ruleService.createRule).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'New Temp Rule',
        metric_name: 'temperature',
        condition: '>=',
        threshold: 90,
      })
    );
    expect(component.showModal()).toBe(false);
  });
});
