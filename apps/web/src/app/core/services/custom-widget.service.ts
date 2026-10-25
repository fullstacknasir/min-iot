import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import {
  CustomWidget,
  CreateCustomWidgetPayload,
  UpdateCustomWidgetPayload,
} from '../models/custom-widget.model';

const CUSTOM_WIDGETS_ENDPOINT = `${DEFAULT_API_BASE_URL}/custom-widgets`;

@Injectable({
  providedIn: 'root',
})
export class CustomWidgetService {
  private readonly httpClient = inject(HttpClient);

  /**
   * Retrieves all custom widget definitions for the organization.
   */
  public getCustomWidgets(): Observable<CustomWidget[]> {
    return this.httpClient
      .get<{ data: CustomWidget[]; count: number }>(CUSTOM_WIDGETS_ENDPOINT)
      .pipe(map((res) => res?.data ?? []));
  }

  /**
   * Retrieves a single custom widget definition by ID.
   */
  public getCustomWidget(id: string): Observable<CustomWidget> {
    return this.httpClient
      .get<{ data: CustomWidget }>(`${CUSTOM_WIDGETS_ENDPOINT}/${id}`)
      .pipe(map((res) => res.data));
  }

  /**
   * Creates and registers a new custom widget definition.
   */
  public createCustomWidget(payload: CreateCustomWidgetPayload): Observable<CustomWidget> {
    return this.httpClient
      .post<{ message: string; data: CustomWidget }>(CUSTOM_WIDGETS_ENDPOINT, payload)
      .pipe(map((res) => res.data));
  }

  /**
   * Updates an existing custom widget definition.
   */
  public updateCustomWidget(id: string, payload: UpdateCustomWidgetPayload): Observable<CustomWidget> {
    return this.httpClient
      .put<{ message: string; data: CustomWidget }>(`${CUSTOM_WIDGETS_ENDPOINT}/${id}`, payload)
      .pipe(map((res) => res.data));
  }

  /**
   * Permanently deletes a custom widget definition.
   */
  public deleteCustomWidget(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${CUSTOM_WIDGETS_ENDPOINT}/${id}`);
  }
}
