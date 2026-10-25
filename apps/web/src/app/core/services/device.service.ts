import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import {
  Device,
  CreateDevicePayload,
  UpdateDevicePayload,
  ProvisionResponse,
  DeviceCredential,
  DeviceStatusActionResponse,
  PaginationParams,
  PaginatedResponse,
} from '../models/device.model';

@Injectable({
  providedIn: 'root',
})
export class DeviceService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = `${DEFAULT_API_BASE_URL}/devices`;

  public getDevices(options?: PaginationParams): Observable<Device[]> {
    const query = this.buildQueryParams(options);
    return this.httpClient
      .get<PaginatedResponse<Device>>(`${this.baseUrl}${query}`)
      .pipe(map((res) => res?.data ?? []));
  }

  public getPaginatedDevices(options?: PaginationParams): Observable<PaginatedResponse<Device>> {
    const query = this.buildQueryParams(options);
    return this.httpClient.get<PaginatedResponse<Device>>(`${this.baseUrl}${query}`);
  }

  public getDevice(deviceId: string): Observable<Device> {
    return this.httpClient
      .get<{ data: Device }>(`${this.baseUrl}/${deviceId}`)
      .pipe(map((res) => res.data));
  }

  public createDevice(payload: CreateDevicePayload): Observable<Device> {
    return this.httpClient
      .post<{ data: Device }>(this.baseUrl, payload)
      .pipe(map((res) => res.data));
  }

  public updateDevice(deviceId: string, payload: UpdateDevicePayload): Observable<Device> {
    return this.httpClient
      .put<{ data: Device }>(`${this.baseUrl}/${deviceId}`, payload)
      .pipe(map((res) => res.data));
  }

  public deleteDevice(deviceId: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl}/${deviceId}`);
  }

  public provisionDevice(deviceId: string): Observable<ProvisionResponse> {
    return this.httpClient.post<ProvisionResponse>(`${this.baseUrl}/${deviceId}/credentials`, {});
  }

  public getDeviceCredentials(deviceId: string): Observable<DeviceCredential> {
    return this.httpClient
      .get<{ data: DeviceCredential }>(`${this.baseUrl}/${deviceId}/credentials`)
      .pipe(map((res) => res.data));
  }

  public enableDevice(deviceId: string): Observable<DeviceStatusActionResponse> {
    return this.httpClient.post<DeviceStatusActionResponse>(`${this.baseUrl}/${deviceId}/enable`, {});
  }

  public disableDevice(deviceId: string): Observable<DeviceStatusActionResponse> {
    return this.httpClient.post<DeviceStatusActionResponse>(`${this.baseUrl}/${deviceId}/disable`, {});
  }

  public revokeDevice(deviceId: string): Observable<DeviceStatusActionResponse> {
    return this.httpClient.post<DeviceStatusActionResponse>(`${this.baseUrl}/${deviceId}/revoke`, {});
  }

  private buildQueryParams(options?: PaginationParams): string {
    if (!options) return '';
    const queryParts: string[] = [];
    if (options.all) queryParts.push('all=true');
    if (options.page) queryParts.push(`page=${options.page}`);
    if (options.limit) queryParts.push(`limit=${options.limit}`);
    return queryParts.length > 0 ? `?${queryParts.join('&')}` : '';
  }
}
