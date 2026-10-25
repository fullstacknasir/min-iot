import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL, STORAGE_ACCESS_TOKEN_KEY } from '../constants/app.constants';
import { PaginatedResponse } from '../models/pagination.model';
import { Notification, NotificationFilterParams, Alert } from '../models/rule-alert.model';

export interface LiveNotificationEvent {
  alert?: Alert;
  notification?: Notification;
}

@Injectable({
  providedIn: 'root',
})
export class NotificationService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = `${DEFAULT_API_BASE_URL}/notifications`;

  public getNotifications(options?: NotificationFilterParams): Observable<Notification[]> {
    const query = this.buildQueryParams(options);
    return this.httpClient
      .get<PaginatedResponse<Notification>>(`${this.baseUrl}${query}`)
      .pipe(map((res) => res?.data ?? []));
  }

  public getPaginatedNotifications(options?: NotificationFilterParams): Observable<PaginatedResponse<Notification>> {
    const query = this.buildQueryParams(options);
    return this.httpClient.get<PaginatedResponse<Notification>>(`${this.baseUrl}${query}`);
  }

  public getUnreadCount(): Observable<number> {
    return this.httpClient
      .get<{ unread_count: number }>(`${this.baseUrl}/unread-count`)
      .pipe(map((res) => res?.unread_count ?? 0));
  }

  public markAsRead(notificationId: string): Observable<void> {
    return this.httpClient.patch<void>(`${this.baseUrl}/${notificationId}/read`, {});
  }

  public markAllAsRead(): Observable<void> {
    return this.httpClient.post<void>(`${this.baseUrl}/read-all`, {});
  }

  /**
   * Connects to Server-Sent Events stream for live notification and alert events.
   */
  public subscribeLiveNotifications(): Observable<LiveNotificationEvent> {
    return new Observable<LiveNotificationEvent>((observer) => {
      if (typeof window === 'undefined' || typeof EventSource === 'undefined') {
        return () => {};
      }
      const token = (typeof localStorage !== 'undefined' && localStorage ? localStorage.getItem(STORAGE_ACCESS_TOKEN_KEY) : null) ?? '';
      const sseUrl = `${this.baseUrl}/live?token=${encodeURIComponent(token)}`;
      const eventSource = new EventSource(sseUrl);

      eventSource.addEventListener('notification', (event: MessageEvent) => {
        try {
          const parsed = JSON.parse(event.data);
          observer.next(parsed);
        } catch (err) {
          console.warn('[SSE] Failed to parse live notification event', err);
        }
      });

      eventSource.onerror = (err) => {
        // SSE auto-reconnects on disconnection
        console.debug('[SSE] Notifications stream reconnecting...', err);
      };

      return () => eventSource.close();
    });
  }

  private buildQueryParams(options?: NotificationFilterParams): string {
    if (!options) return '';
    const queryParts: string[] = [];
    if (options.all) queryParts.push('all=true');
    if (options.page) queryParts.push(`page=${options.page}`);
    if (options.limit) queryParts.push(`limit=${options.limit}`);
    if (options.unread) queryParts.push('unread=true');
    return queryParts.length > 0 ? `?${queryParts.join('&')}` : '';
  }
}
