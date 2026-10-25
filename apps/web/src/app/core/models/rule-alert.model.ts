import { PaginationParams, PaginatedResponse } from './pagination.model';

export type RuleSeverity = 'critical' | 'warning' | 'info';
export type RuleCondition = '>' | '>=' | '<' | '<=' | '==' | '!=';
export type NotificationChannel = 'in_app' | 'email' | 'webhook' | 'all';
export type AlertStatus = 'active' | 'acknowledged' | 'resolved';

export interface Rule {
  id: string;
  organization_id: string;
  name: string;
  description: string;
  device_id?: string | null;
  device_name?: string | null;
  device_type?: string;
  metric_name: string;
  condition: RuleCondition;
  threshold: number;
  severity: RuleSeverity;
  cooldown_seconds: number;
  notification_channel: NotificationChannel;
  webhook_url: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateRulePayload {
  name: string;
  description?: string;
  device_id?: string | null;
  device_type?: string;
  metric_name: string;
  condition: RuleCondition;
  threshold: number;
  severity?: RuleSeverity;
  cooldown_seconds?: number;
  notification_channel?: NotificationChannel;
  webhook_url?: string;
  enabled?: boolean;
}

export interface UpdateRulePayload {
  name?: string;
  description?: string;
  device_id?: string | null;
  clear_device_id?: boolean;
  device_type?: string;
  metric_name?: string;
  condition?: RuleCondition;
  threshold?: number;
  severity?: RuleSeverity;
  cooldown_seconds?: number;
  notification_channel?: NotificationChannel;
  webhook_url?: string;
  enabled?: boolean;
}

export interface Alert {
  id: string;
  organization_id: string;
  rule_id?: string | null;
  rule_name?: string | null;
  device_id: string;
  device_name?: string | null;
  severity: RuleSeverity;
  status: AlertStatus;
  message: string;
  metric_name: string;
  metric_value: number;
  threshold_value: number;
  triggered_at: string;
  acknowledged_at?: string | null;
  acknowledged_by?: string | null;
  resolved_at?: string | null;
  resolved_by?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Notification {
  id: string;
  organization_id: string;
  alert_id?: string | null;
  title: string;
  message: string;
  severity: RuleSeverity;
  channel: string;
  read: boolean;
  created_at: string;
}

export interface RuleFilterParams extends PaginationParams {
  device_id?: string;
  enabled?: boolean;
}

export interface AlertFilterParams extends PaginationParams {
  status?: string;
  severity?: string;
  device_id?: string;
}

export interface NotificationFilterParams extends PaginationParams {
  unread?: boolean;
}
