import { Injectable, inject, NgZone } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { DEFAULT_API_BASE_URL, LIVE_TELEMETRY_STREAM_URL } from '../constants/app.constants';
import { TelemetryPoint, AggregatePoint, LatestTelemetryData } from '../models/telemetry.model';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root',
})
export class TelemetryService {
  private readonly httpClient = inject(HttpClient);
  private readonly authService = inject(AuthService);
  private readonly ngZone = inject(NgZone);

  public getLatestTelemetry(deviceId: string): Observable<LatestTelemetryData | null> {
    const url = `${DEFAULT_API_BASE_URL}/devices/${deviceId}/telemetry/latest`;
    return this.httpClient
      .get<{ data: LatestTelemetryData }>(url)
      .pipe(map((res) => res?.data ?? null));
  }

  public getHistory(
    deviceId: string,
    params?: { from?: string; to?: string; metric?: string; limit?: number; order?: string }
  ): Observable<TelemetryPoint[]> {
    const url = `${DEFAULT_API_BASE_URL}/devices/${deviceId}/telemetry/history`;
    let httpParams = new HttpParams();
    if (params?.from) httpParams = httpParams.set('from', params.from);
    if (params?.to) httpParams = httpParams.set('to', params.to);
    if (params?.metric) httpParams = httpParams.set('metric', params.metric);
    if (params?.limit) httpParams = httpParams.set('limit', params.limit.toString());
    if (params?.order) httpParams = httpParams.set('order', params.order);

    return this.httpClient
      .get<{ data: TelemetryPoint[]; count: number }>(url, { params: httpParams })
      .pipe(map((res) => res?.data ?? []));
  }

  public getAggregates(
    deviceId: string,
    bucket: string,
    params?: { from?: string; to?: string; metric?: string; limit?: number }
  ): Observable<AggregatePoint[]> {
    const url = `${DEFAULT_API_BASE_URL}/devices/${deviceId}/telemetry/aggregates`;
    let httpParams = new HttpParams().set('bucket', bucket);
    if (params?.from) httpParams = httpParams.set('from', params.from);
    if (params?.to) httpParams = httpParams.set('to', params.to);
    if (params?.metric) httpParams = httpParams.set('metric', params.metric);
    if (params?.limit) httpParams = httpParams.set('limit', params.limit.toString());

    return this.httpClient
      .get<{ aggregates: AggregatePoint[]; count: number }>(url, { params: httpParams })
      .pipe(map((res) => res?.aggregates ?? []));
  }

  /**
   * Connects to Server-Sent Events stream for real-time telemetry updates.
   */
  public connectLiveStream(deviceId: string): Observable<LatestTelemetryData> {
    return new Observable<LatestTelemetryData>((observer) => {
      const token = this.authService.getRawAccessToken() ?? '';
      const streamUrl = `${LIVE_TELEMETRY_STREAM_URL}/${deviceId}/telemetry/live?token=${encodeURIComponent(token)}`;
      const eventSource = new EventSource(streamUrl);

      const handlePayload = (e: MessageEvent) => {
        this.ngZone.run(() => {
          try {
            const parsed = JSON.parse(e.data);
            observer.next(parsed?.data ?? parsed);
          } catch (err) {
            console.warn('[SSE] Failed parsing telemetry payload', err);
          }
        });
      };

      eventSource.addEventListener('initial', handlePayload);
      eventSource.addEventListener('telemetry', handlePayload);
      eventSource.onerror = (err) => {
        this.ngZone.run(() => console.warn('[SSE] EventSource connection warning', err));
      };

      return () => eventSource.close();
    });
  }

  public sendSimulatedTelemetry(deviceId: string, metrics: Record<string, number>): Observable<boolean> {
    return this.httpClient
      .post(`${DEFAULT_API_BASE_URL}/telemetry`, { device_id: deviceId, metrics })
      .pipe(
        map(() => true)
      );
  }
}
