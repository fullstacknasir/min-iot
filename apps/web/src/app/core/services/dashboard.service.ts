import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import { Dashboard, CreateDashboardPayload, UpdateDashboardPayload } from '../models/dashboard.model';

@Injectable({
  providedIn: 'root',
})
export class DashboardService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = `${DEFAULT_API_BASE_URL}/dashboards`;

  public getDashboards(): Observable<Dashboard[]> {
    return this.httpClient
      .get<{ data: Dashboard[]; count: number }>(this.baseUrl)
      .pipe(map((res) => res.data ?? []));
  }

  public getDashboard(id: string): Observable<Dashboard> {
    return this.httpClient
      .get<{ data: Dashboard }>(`${this.baseUrl}/${id}`)
      .pipe(map((res) => res.data));
  }

  public createDashboard(payload: CreateDashboardPayload): Observable<Dashboard> {
    return this.httpClient
      .post<{ message: string; data: Dashboard }>(this.baseUrl, payload)
      .pipe(map((res) => res.data));
  }

  public updateDashboard(id: string, payload: UpdateDashboardPayload): Observable<Dashboard> {
    return this.httpClient
      .put<{ message: string; data: Dashboard }>(`${this.baseUrl}/${id}`, payload)
      .pipe(map((res) => res.data));
  }

  public deleteDashboard(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl}/${id}`);
  }

  public setDefaultDashboard(id: string): Observable<void> {
    return this.httpClient.post<void>(`${this.baseUrl}/${id}/default`, {});
  }
}
