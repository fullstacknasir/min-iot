import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, catchError, throwError, of, map } from 'rxjs';
import {
  DEFAULT_API_BASE_URL,
  STORAGE_ACCESS_TOKEN_KEY,
  STORAGE_REFRESH_TOKEN_KEY,
  STORAGE_USER_KEY,
  STORAGE_ORG_KEY,
  ROLE_ADMIN,
  ROLE_OPERATOR,
} from '../constants/app.constants';
import {
  AuthResponse,
  LoginPayload,
  RefreshTokenPayload,
  User,
  Organization,
  UpdateProfilePayload,
  ProfileResponse,
} from '../models/auth.model';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly httpClient = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly authBaseUrl = `${DEFAULT_API_BASE_URL}/auth`;

  public readonly currentUser = signal<User | null>(this.readInitialUser());
  public readonly currentOrg = signal<Organization | null>(this.readInitialOrg());
  public readonly accessToken = signal<string | null>(this.getStorageItem(STORAGE_ACCESS_TOKEN_KEY));

  public readonly isAuthenticated = computed(() => !!this.accessToken());
  public readonly isAdmin = computed(() => this.currentUser()?.role === ROLE_ADMIN);
  public readonly isOperatorOrAdmin = computed(() => {
    const role = this.currentUser()?.role;
    return role === ROLE_ADMIN || role === ROLE_OPERATOR;
  });

  /**
   * Authenticates user against API and stores token pair.
   */
  public login(credentials: LoginPayload): Observable<AuthResponse> {
    return this.httpClient.post<AuthResponse>(`${this.authBaseUrl}/login`, credentials).pipe(
      tap((response) => this.persistSession(response))
    );
  }

  /**
   * Refreshes access token using stored refresh token with rotation.
   */
  public refreshTokens(): Observable<string | null> {
    const refreshToken = this.getStorageItem(STORAGE_REFRESH_TOKEN_KEY);
    if (!refreshToken) {
      this.clearSessionAndRedirect();
      return of(null);
    }

    const payload: RefreshTokenPayload = { refresh_token: refreshToken };
    return this.httpClient.post<AuthResponse>(`${this.authBaseUrl}/refresh`, payload).pipe(
      map((response) => {
        this.persistSession(response);
        return response.access_token;
      }),
      catchError((error) => {
        this.clearSessionAndRedirect();
        return throwError(() => error);
      })
    );
  }

  /**
   * Revokes server session and cleans client storage.
   */
  public logout(): void {
    const refreshToken = this.getStorageItem(STORAGE_REFRESH_TOKEN_KEY);
    if (refreshToken) {
      this.httpClient.post(`${this.authBaseUrl}/logout`, { refresh_token: refreshToken }).subscribe({
        error: (err) => console.warn('Logout API warning:', err),
      });
    }
    this.clearSessionAndRedirect();
  }

  /**
   * Fetches current authenticated user profile from server.
   */
  public getProfile(): Observable<ProfileResponse> {
    return this.httpClient.get<ProfileResponse>(`${this.authBaseUrl}/me`);
  }

  /**
   * Updates user profile attributes and synchronizes local state.
   */
  public updateProfile(payload: UpdateProfilePayload): Observable<ProfileResponse> {
    return this.httpClient.put<ProfileResponse>(`${this.authBaseUrl}/me`, payload).pipe(
      tap((res) => {
        if (res?.user) {
          this.setStorageItem(STORAGE_USER_KEY, JSON.stringify(res.user));
          this.currentUser.set(res.user);
        }
        if (res?.organization) {
          this.setStorageItem(STORAGE_ORG_KEY, JSON.stringify(res.organization));
          this.currentOrg.set(res.organization);
        }
      })
    );
  }

  public getRawAccessToken(): string | null {
    return this.accessToken() ?? this.getStorageItem(STORAGE_ACCESS_TOKEN_KEY);
  }

  private persistSession(response: AuthResponse): void {
    this.setStorageItem(STORAGE_ACCESS_TOKEN_KEY, response.access_token);
    this.setStorageItem(STORAGE_REFRESH_TOKEN_KEY, response.refresh_token);
    if (response.user) {
      this.setStorageItem(STORAGE_USER_KEY, JSON.stringify(response.user));
      this.currentUser.set(response.user);
    }
    if (response.organization) {
      this.setStorageItem(STORAGE_ORG_KEY, JSON.stringify(response.organization));
      this.currentOrg.set(response.organization);
    }
    this.accessToken.set(response.access_token);
  }

  private clearSessionAndRedirect(): void {
    this.removeStorageItem(STORAGE_ACCESS_TOKEN_KEY);
    this.removeStorageItem(STORAGE_REFRESH_TOKEN_KEY);
    this.removeStorageItem(STORAGE_USER_KEY);
    this.removeStorageItem(STORAGE_ORG_KEY);
    this.accessToken.set(null);
    this.currentUser.set(null);
    this.currentOrg.set(null);
    this.router.navigate(['/login']);
  }

  private readInitialUser(): User | null {
    const raw = this.getStorageItem(STORAGE_USER_KEY);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as User;
    } catch {
      return null;
    }
  }

  private readInitialOrg(): Organization | null {
    const raw = this.getStorageItem(STORAGE_ORG_KEY);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as Organization;
    } catch {
      return null;
    }
  }

  private getStorageItem(key: string): string | null {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return null;
    try {
      return localStorage.getItem(key);
    } catch {
      return null;
    }
  }

  private setStorageItem(key: string, val: string): void {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(key, val);
    } catch {
      // storage unavailable
    }
  }

  private removeStorageItem(key: string): void {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return;
    try {
      localStorage.removeItem(key);
    } catch {
      // storage unavailable
    }
  }
}
