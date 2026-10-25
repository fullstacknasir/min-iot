import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { DEFAULT_API_BASE_URL } from '../constants/app.constants';
import { PaginatedResponse } from '../models/pagination.model';
import { Rule, CreateRulePayload, UpdateRulePayload, RuleFilterParams } from '../models/rule-alert.model';

@Injectable({
  providedIn: 'root',
})
export class RuleService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = `${DEFAULT_API_BASE_URL}/rules`;

  public getRules(options?: RuleFilterParams): Observable<Rule[]> {
    const query = this.buildQueryParams(options);
    return this.httpClient
      .get<PaginatedResponse<Rule>>(`${this.baseUrl}${query}`)
      .pipe(map((res) => res?.data ?? []));
  }

  public getPaginatedRules(options?: RuleFilterParams): Observable<PaginatedResponse<Rule>> {
    const query = this.buildQueryParams(options);
    return this.httpClient.get<PaginatedResponse<Rule>>(`${this.baseUrl}${query}`);
  }

  public getRule(ruleId: string): Observable<Rule> {
    return this.httpClient
      .get<{ data: Rule }>(`${this.baseUrl}/${ruleId}`)
      .pipe(map((res) => res.data));
  }

  public createRule(payload: CreateRulePayload): Observable<Rule> {
    return this.httpClient
      .post<{ data: Rule }>(this.baseUrl, payload)
      .pipe(map((res) => res.data));
  }

  public updateRule(ruleId: string, payload: UpdateRulePayload): Observable<Rule> {
    return this.httpClient
      .put<{ data: Rule }>(`${this.baseUrl}/${ruleId}`, payload)
      .pipe(map((res) => res.data));
  }

  public deleteRule(ruleId: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl}/${ruleId}`);
  }

  public toggleRule(ruleId: string): Observable<{ enabled: boolean }> {
    return this.httpClient.patch<{ enabled: boolean }>(`${this.baseUrl}/${ruleId}/toggle`, {});
  }

  private buildQueryParams(options?: RuleFilterParams): string {
    if (!options) return '';
    const queryParts: string[] = [];
    if (options.all) queryParts.push('all=true');
    if (options.page) queryParts.push(`page=${options.page}`);
    if (options.limit) queryParts.push(`limit=${options.limit}`);
    if (options.device_id) queryParts.push(`device_id=${encodeURIComponent(options.device_id)}`);
    if (options.enabled !== undefined) queryParts.push(`enabled=${options.enabled}`);
    return queryParts.length > 0 ? `?${queryParts.join('&')}` : '';
  }
}
