import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import { PaginatedResponse } from '../models/pagination.model';
import { Alert, AlertFilterParams } from '../models/rule-alert.model';

@Injectable({
  providedIn: 'root',
})
export class AlertService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = `${DEFAULT_API_BASE_URL}/alerts`;

  public getAlerts(options?: AlertFilterParams): Observable<Alert[]> {
    const query = this.buildQueryParams(options);
    return this.httpClient
      .get<PaginatedResponse<Alert>>(`${this.baseUrl}${query}`)
      .pipe(map((res) => res?.data ?? []));
  }

  public getPaginatedAlerts(options?: AlertFilterParams): Observable<PaginatedResponse<Alert>> {
    const query = this.buildQueryParams(options);
    return this.httpClient.get<PaginatedResponse<Alert>>(`${this.baseUrl}${query}`);
  }

  public getAlert(alertId: string): Observable<Alert> {
    return this.httpClient
      .get<{ data: Alert }>(`${this.baseUrl}/${alertId}`)
      .pipe(map((res) => res.data));
  }

  public acknowledgeAlert(alertId: string): Observable<void> {
    return this.httpClient.post<void>(`${this.baseUrl}/${alertId}/acknowledge`, {});
  }

  public resolveAlert(alertId: string): Observable<void> {
    return this.httpClient.post<void>(`${this.baseUrl}/${alertId}/resolve`, {});
  }

  private buildQueryParams(options?: AlertFilterParams): string {
    if (!options) return '';
    const queryParts: string[] = [];
    if (options.all) queryParts.push('all=true');
    if (options.page) queryParts.push(`page=${options.page}`);
    if (options.limit) queryParts.push(`limit=${options.limit}`);
    if (options.status && options.status !== 'all') queryParts.push(`status=${encodeURIComponent(options.status)}`);
    if (options.severity && options.severity !== 'all') queryParts.push(`severity=${encodeURIComponent(options.severity)}`);
    if (options.device_id) queryParts.push(`device_id=${encodeURIComponent(options.device_id)}`);
    return queryParts.length > 0 ? `?${queryParts.join('&')}` : '';
  }
}
