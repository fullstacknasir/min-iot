import { Injectable, inject } from '@angular/core';
import {
  HttpInterceptor,
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpErrorResponse,
} from '@angular/common/http';
import { Observable, throwError, BehaviorSubject } from 'rxjs';
import { catchError, switchMap, filter, take } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';

const AUTH_URL_MARKER = '/api/v1/auth/';

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  private readonly authService = inject(AuthService);
  private isRefreshing = false;
  private readonly refreshTokenSubject = new BehaviorSubject<string | null>(null);

  public intercept(request: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    const token = this.authService.getRawAccessToken();
    const authorizedReq = this.attachToken(request, token);

    return next.handle(authorizedReq).pipe(
      catchError((error: unknown) => {
        if (error instanceof HttpErrorResponse && error.status === 401 && !request.url.includes(AUTH_URL_MARKER)) {
          return this.handle401Error(request, next);
        }
        return throwError(() => error);
      })
    );
  }

  /**
   * Refreshes access token and replays the original failed HTTP request.
   */
  private handle401Error(request: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    if (!this.isRefreshing) {
      this.isRefreshing = true;
      this.refreshTokenSubject.next(null);

      return this.authService.refreshTokens().pipe(
        switchMap((newToken: string | null) => {
          this.isRefreshing = false;
          if (!newToken) return throwError(() => new Error('Refresh failed'));
          this.refreshTokenSubject.next(newToken);
          return next.handle(this.attachToken(request, newToken));
        }),
        catchError((refreshErr) => {
          this.isRefreshing = false;
          return throwError(() => refreshErr);
        })
      );
    }

    return this.refreshTokenSubject.pipe(
      filter((token) => token !== null),
      take(1),
      switchMap((token) => next.handle(this.attachToken(request, token)))
    );
  }

  private attachToken(request: HttpRequest<unknown>, token: string | null): HttpRequest<unknown> {
    if (!token || request.headers.has('Authorization')) return request;
    return request.clone({
      setHeaders: { Authorization: `Bearer ${token}` },
    });
  }
}
