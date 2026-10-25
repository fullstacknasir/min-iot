export type WidgetType = 'gauge' | 'chart' | 'live_feed' | 'digital_card' | 'switch' | 'custom';

export interface WidgetConfig {
  id: string;
  type: WidgetType;
  title: string;
  device_id: string;
  device_name?: string;
  metric_name?: string;
  col_span: number;
  row_span: number;
  settings?: Record<string, unknown>;
}

export interface DashboardConfiguration {
  widgets: WidgetConfig[];
}

export interface Dashboard {
  id: string;
  organization_id: string;
  title: string;
  description: string;
  is_default: boolean;
  configuration: DashboardConfiguration;
  created_at: string;
  updated_at: string;
}

export interface CreateDashboardPayload {
  title: string;
  description?: string;
  is_default?: boolean;
  configuration?: DashboardConfiguration;
}

export interface UpdateDashboardPayload {
  title?: string;
  description?: string;
  is_default?: boolean;
  configuration?: DashboardConfiguration;
}
