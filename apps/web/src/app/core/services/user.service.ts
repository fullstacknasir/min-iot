import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import { User, CreateOrgUserPayload, UserRole } from '../models/auth.model';

@Injectable({
  providedIn: 'root',
})
export class UserService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = `${DEFAULT_API_BASE_URL}/users`;

  public getUsers(): Observable<User[]> {
    return this.httpClient
      .get<{ data: User[]; total: number }>(this.baseUrl)
      .pipe(map((res) => res?.data ?? []));
  }

  public createUser(payload: CreateOrgUserPayload): Observable<User> {
    return this.httpClient
      .post<{ data: User }>(this.baseUrl, payload)
      .pipe(map((res) => res.data));
  }

  public updateUserRole(userId: string, role: UserRole): Observable<void> {
    return this.httpClient.patch<void>(`${this.baseUrl}/${userId}/role`, { role });
  }

  public deleteUser(userId: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl}/${userId}`);
  }
}
