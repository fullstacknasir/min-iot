import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { MatIconModule } from '@angular/material/icon';
import { of } from 'rxjs';
import { TelemetryLiveWidgetComponent } from './telemetry-live-widget.component';
import { TelemetryService } from '../../core/services/telemetry.service';
import { LatestTelemetryData } from '../../core/models/telemetry.model';

describe('TelemetryLiveWidgetComponent (ThingsBoard Custom Widget)', () => {
  let component: TelemetryLiveWidgetComponent;
  let fixture: ComponentFixture<TelemetryLiveWidgetComponent>;
  let telemetryServiceMock: {
    getLatestTelemetry: ReturnType<typeof vi.fn>;
    connectLiveStream: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    telemetryServiceMock = {
      getLatestTelemetry: vi.fn().mockReturnValue(of({
        device_id: 'dev-1',
        metrics: { temperature: 24.5, humidity: 60.0 },
      })),
      connectLiveStream: vi.fn().mockReturnValue(of({
        device_id: 'dev-1',
        metrics: { temperature: 25.0, humidity: 61.2 },
      })),
    };

    await TestBed.configureTestingModule({
      declarations: [TelemetryLiveWidgetComponent],
      imports: [MatIconModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: TelemetryService, useValue: telemetryServiceMock },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TelemetryLiveWidgetComponent);
    component = fixture.componentInstance;
    component.deviceId = 'dev-1';
  });

  it('should initialize and execute ThingsBoard widget lifecycle tbOnInit', () => {
    fixture.detectChanges();
    expect(telemetryServiceMock.getLatestTelemetry).toHaveBeenCalledWith('dev-1');
    expect(telemetryServiceMock.connectLiveStream).toHaveBeenCalledWith('dev-1');
    expect(component.isConnected()).toBe(true);
  });

  it('should update metrics when tbOnDataUpdated is called', () => {
    const sampleData: LatestTelemetryData = {
      device_id: 'dev-1',
      metrics: { voltage: 3.3, current: 0.5 },
      timestamp: new Date().toISOString(),
    };

    component.tbOnDataUpdated(sampleData);
    expect(component.metricList().length).toBe(2);
    expect(component.metricList()[0].name).toBe('voltage');
    expect(component.metricList()[0].value).toBe(3.3);
  });
});
