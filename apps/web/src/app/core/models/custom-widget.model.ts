import moment from 'moment';
import * as echarts from 'echarts';
import { LatestTelemetryData } from './telemetry.model';

export interface CustomWidget {
  id: string;
  organization_id: string;
  name: string;
  description: string;
  widget_type: string;
  template_html: string;
  style_css: string;
  controller_js: string;
  settings_schema: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomWidgetPayload {
  name: string;
  description?: string;
  widget_type?: string;
  template_html?: string;
  style_css?: string;
  controller_js?: string;
  settings_schema?: Record<string, unknown>;
}

export interface UpdateCustomWidgetPayload {
  name?: string;
  description?: string;
  widget_type?: string;
  template_html?: string;
  style_css?: string;
  controller_js?: string;
  settings_schema?: Record<string, unknown>;
}

/**
 * ThingsBoard custom widget execution context passed to self.ctx
 */
export interface ThingsBoardWidgetContext {
  $container: HTMLElement;
  data: LatestTelemetryData | null;
  datasources: Array<{
    deviceId: string;
    metricName?: string;
  }>;
  settings: Record<string, unknown>;
  moment: typeof moment;
  echarts: typeof echarts;
  currentUser?: {
    id: string;
    email: string;
    role: string;
  };
}

export interface ThingsBoardWidgetInstance {
  ctx: ThingsBoardWidgetContext;
  onInit?: () => void;
  onDataUpdated?: (telemetry?: LatestTelemetryData) => void;
  onResize?: () => void;
  onDestroy?: () => void;
}
