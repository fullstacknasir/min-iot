import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { AppModule } from '../../app.module';
import { TelemetryGaugeWidgetComponent } from './telemetry-gauge-widget.component';
import { TelemetryService } from '../../core/services/telemetry.service';
import { LatestTelemetryData } from '../../core/models/telemetry.model';

describe('TelemetryGaugeWidgetComponent', () => {
  let telemetryService: TelemetryService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideNoopAnimations(),
      ],
    }).compileComponents();

    telemetryService = TestBed.inject(TelemetryService);
  });

  it('should initialize and process data updates via ThingsBoard lifecycle', () => {
    const mockData: LatestTelemetryData = {
      device_id: 'dev-gauge-1',
      metrics: {
        pressure: 75.5,
      },
      received_at: new Date().toISOString(),
    };

    vi.spyOn(telemetryService, 'getLatestTelemetry').mockReturnValue(of(mockData));
    vi.spyOn(telemetryService, 'connectLiveStream').mockReturnValue(of(mockData));

    const fixture = TestBed.createComponent(TelemetryGaugeWidgetComponent);
    const component = fixture.componentInstance;
    component.deviceId = 'dev-gauge-1';
    component.metricName = 'pressure';
    component.min = 0;
    component.max = 100;

    fixture.detectChanges();

    expect(component.currentValue()).toBe(75.5);
    expect(component.isConnected()).toBe(true);
    expect(component.getResolvedUnit()).toBe('hPa');
  });

  it('should handle tbOnResize without errors', () => {
    const fixture = TestBed.createComponent(TelemetryGaugeWidgetComponent);
    const component = fixture.componentInstance;
    component.deviceId = 'dev-gauge-1';
    component.metricName = 'temperature';

    fixture.detectChanges();
    expect(() => component.tbOnResize()).not.toThrow();
  });
});
