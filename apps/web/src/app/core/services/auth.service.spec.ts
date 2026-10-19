import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Router } from '@angular/router';
import { AuthService } from './auth.service';
import { DEFAULT_API_BASE_URL, ROLE_ADMIN } from '../constants/app.constants';
import { AuthResponse } from '../models/auth.model';

describe('AuthService', () => {
  let service: AuthService;
  let httpTesting: HttpTestingController;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    routerSpy = { navigate: vi.fn() };

    TestBed.configureTestingModule({
      providers: [
        AuthService,
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
      ],
    });

    service = TestBed.inject(AuthService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should authenticate user and store token session', () => {
    const mockResponse: AuthResponse = {
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      expires_in: 900,
      user: {
        id: 'usr-1',
        organization_id: 'org-1',
        email: 'admin@miniot.io',
        role: ROLE_ADMIN,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    };

    service.login({ email: 'admin@miniot.io', password: 'Password123!' }).subscribe((res) => {
      expect(res.access_token).toBe('mock-access-token');
      expect(service.isAuthenticated()).toBe(true);
      expect(service.isAdmin()).toBe(true);
    });

    const req = httpTesting.expectOne(`${DEFAULT_API_BASE_URL}/auth/login`);
    expect(req.request.method).toBe('POST');
    req.flush(mockResponse);
  });
});
