export interface TelemetryPoint {
  time: string;
  device_id: string;
  metric_name: string;
  metric_value: number;
  string_value?: string;
  received_at?: string;
}

export interface AggregatePoint {
  bucket: string;
  device_id: string;
  metric_name: string;
  avg_value: number;
  min_value: number;
  max_value: number;
  count_value: number;
}

export interface LatestTelemetryData {
  device_id: string;
  timestamp?: string;
  received_at?: string;
  metrics: Record<string, number>;
  updated_at?: string;
}

export interface LiveTelemetryEvent {
  device_id: string;
  timestamp: string;
  metrics: Record<string, number>;
}

export interface SystemHealth {
  status: string;
  timestamp: string;
  services: Record<string, string>;
}

export interface TimeRangeOption {
  label: string;
  value: string;
  bucket: string;
  durationMinutes: number;
}
