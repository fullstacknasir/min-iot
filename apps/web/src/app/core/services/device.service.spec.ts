import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { DeviceService } from './device.service';
import { DEFAULT_API_BASE_URL, STATUS_ONLINE } from '../constants/app.constants';
import { Device } from '../models/device.model';

describe('DeviceService', () => {
  let service: DeviceService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [DeviceService, provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(DeviceService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should fetch list of devices', () => {
    const mockDevices: Device[] = [
      {
        id: 'dev-1',
        organization_id: 'org-1',
        name: 'Sensor 1',
        device_type: 'sensor',
        status: STATUS_ONLINE,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

    service.getDevices().subscribe((devices) => {
      expect(devices.length).toBe(1);
      expect(devices[0].id).toBe('dev-1');
    });

    const req = httpTesting.expectOne(`${DEFAULT_API_BASE_URL}/devices`);
    expect(req.request.method).toBe('GET');
    req.flush({ data: mockDevices, total: 1 });
  });

  it('should provision device credentials', () => {
    service.provisionDevice('dev-1').subscribe((res) => {
      expect(res.token).toBe('secret-dev-token');
    });

    const req = httpTesting.expectOne(`${DEFAULT_API_BASE_URL}/devices/dev-1/credentials`);
    expect(req.request.method).toBe('POST');
    req.flush({
      device_id: 'dev-1',
      token: 'secret-dev-token',
      credential_id: 'cred-1',
      status: 'active',
    });
  });
});
