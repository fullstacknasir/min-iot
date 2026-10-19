import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, map, of } from 'rxjs';
import { DEFAULT_API_BASE_URL, HEALTH_CHECK_URL } from '../constants/app.constants';
import { Device, LatestTelemetryData, SystemHealth, TelemetryPoint } from '../models/device.model';

@Injectable({
  providedIn: 'root',
})
export class ApiService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = DEFAULT_API_BASE_URL;

  /**
   * Fetches the full fleet of registered IoT devices.
   */
  public getDevices(): Observable<Device[]> {
    return this.httpClient.get<{ data: Device[] }>(`${this.baseUrl}/devices`).pipe(
      map((response) => response?.data ?? []),
      catchError((error) => this.handleError<Device[]>('getDevices', [], error))
    );
  }

  /**
   * Retrieves the latest real-time cached telemetry from Redis.
   */
  public getLatestTelemetry(deviceId: string): Observable<LatestTelemetryData | null> {
    if (!deviceId) return of(null);

    return this.httpClient.get<{ data: LatestTelemetryData }>(`${this.baseUrl}/devices/${deviceId}/telemetry/latest`).pipe(
      map((response) => response?.data ?? null),
      catchError((error) => this.handleError<LatestTelemetryData | null>('getLatestTelemetry', null, error))
    );
  }

  /**
   * Queries historical telemetry entries stored in TimescaleDB hypertable.
   */
  public getTelemetryHistory(deviceId: string, limit: number = 20): Observable<TelemetryPoint[]> {
    if (!deviceId) return of([]);

    return this.httpClient.get<{ data: TelemetryPoint[] }>(`${this.baseUrl}/devices/${deviceId}/telemetry/history?limit=${limit}`).pipe(
      map((response) => response?.data ?? []),
      catchError((error) => this.handleError<TelemetryPoint[]>('getTelemetryHistory', [], error))
    );
  }

  /**
   * Checks the health and connection status of TimescaleDB and Redis.
   */
  public checkHealth(): Observable<SystemHealth> {
    return this.httpClient.get<SystemHealth>(HEALTH_CHECK_URL).pipe(
      catchError(() => of({ status: 'unreachable', timestamp: new Date().toISOString(), services: {} }))
    );
  }

  /**
   * Simulates sending a new telemetry reading directly to the API.
   */
  public sendTelemetry(deviceId: string, metrics: Record<string, number>): Observable<boolean> {
    if (!deviceId || !metrics) return of(false);

    return this.httpClient.post(`${this.baseUrl}/telemetry`, { device_id: deviceId, metrics }).pipe(
      map(() => true),
      catchError(() => of(false))
    );
  }

  private handleError<T>(operation: string, fallback: T, error: unknown): Observable<T> {
    console.warn(`[ApiService] Operation ${operation} failed:`, error);
    return of(fallback);
  }
}
