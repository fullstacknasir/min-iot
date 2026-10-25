export interface WidgetPreset {
  id: string;
  name: string;
  description: string;
  widget_type: string;
  template_html: string;
  style_css: string;
  controller_js: string;
  settings_schema: Record<string, unknown>;
}

export const WIDGET_PRESETS: WidgetPreset[] = [
  {
    id: 'radial_gauge',
    name: 'ThingsBoard Radial Gauge',
    description: 'Dynamic circular gauge rendered using ECharts with live telemetry needle reflow',
    widget_type: 'telemetry',
    template_html: `<div class="flex flex-col items-center justify-center p-3 h-full">
  <div id="tb-gauge-chart" class="w-full h-44"></div>
  <div class="mt-1 text-center">
    <span id="metric-label" class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Temperature</span>
    <div id="metric-value" class="text-xl font-bold text-slate-900 dark:text-white">-- °C</div>
  </div>
</div>`,
    style_css: `#tb-gauge-chart {
  min-height: 160px;
}
#metric-value {
  transition: all 0.3s ease;
}`,
    controller_js: `var chartInstance;

self.onInit = function() {
  var container = self.ctx.$container.querySelector('#tb-gauge-chart');
  if (container && self.ctx.echarts) {
    try {
      chartInstance = self.ctx.echarts.init(container);
      renderGauge(24.5);
    } catch (err) {
      console.warn('ECharts initialization handled:', err);
    }
  }
};

self.onDataUpdated = function(telemetry) {
  var metrics = telemetry && telemetry.metrics ? telemetry.metrics : {};
  var val = metrics.temperature !== undefined ? metrics.temperature : metrics.temp;
  if (val === undefined) {
    var firstKey = Object.keys(metrics)[0];
    if (firstKey) val = metrics[firstKey];
  }
  if (val !== undefined && typeof val === 'number') {
    renderGauge(val);
    var valueEl = self.ctx.$container.querySelector('#metric-value');
    if (valueEl) valueEl.textContent = val.toFixed(1) + ' °C';
  }
};

self.onResize = function() {
  if (chartInstance) {
    try {
      chartInstance.resize();
    } catch (e) {}
  }
};

self.onDestroy = function() {
  if (chartInstance) {
    try {
      chartInstance.dispose();
    } catch (e) {}
  }
  chartInstance = undefined;
};

function renderGauge(val) {
  if (!chartInstance) return;
  try {
    chartInstance.setOption({
      backgroundColor: 'transparent',
      series: [{
        type: 'gauge',
        min: 0,
        max: 100,
        progress: { show: true, width: 10, itemStyle: { color: '#06b6d4' } },
        pointer: { show: true, length: '60%', width: 3 },
        axisLine: { lineStyle: { width: 10, color: [[1, '#334155']] } },
        splitLine: { length: 6, lineStyle: { width: 1, color: '#64748b' } },
        axisLabel: { distance: 12, fontSize: 9, color: '#94a3b8' },
        detail: { show: false },
        data: [{ value: val }]
      }]
    });
  } catch (err) {}
}`,
    settings_schema: {
      min: 0,
      max: 100,
      unit: '°C',
    },
  },
  {
    id: 'industrial_metric_card',
    name: 'Industrial Telemetry Metric Card',
    description: 'High-contrast gradient metric card with threshold alarms and Moment.js timestamps',
    widget_type: 'telemetry',
    template_html: `<div class="tb-metric-card p-5 rounded-2xl flex flex-col justify-between h-full bg-gradient-to-br from-slate-900 to-slate-800 text-white shadow-lg">
  <div class="flex items-center justify-between">
    <span id="card-title" class="text-xs font-semibold uppercase tracking-wider text-cyan-400">Pressure Telemetry</span>
    <span id="status-pill" class="px-2 py-0.5 text-[10px] font-bold rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">NORMAL</span>
  </div>
  <div class="my-4">
    <div class="flex items-baseline gap-2">
      <span id="metric-display" class="text-4xl font-extrabold tracking-tight">--</span>
      <span id="unit-display" class="text-sm font-medium text-slate-400">bar</span>
    </div>
    <p id="subtext-display" class="text-xs text-slate-400 mt-1">Real-time IoT Telemetry Stream</p>
  </div>
  <div class="pt-3 border-t border-slate-700/60 flex items-center justify-between text-[11px] text-slate-400">
    <span>Last telemetry:</span>
    <span id="timestamp-display" class="font-mono">Waiting...</span>
  </div>
</div>`,
    style_css: `.tb-metric-card {
  min-height: 180px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.tb-metric-card:hover {
  transform: translateY(-2px);
}`,
    controller_js: `self.onInit = function() {
  var title = self.ctx.settings.title || 'Pressure Telemetry';
  var unit = self.ctx.settings.unit || 'bar';
  var titleEl = self.ctx.$container.querySelector('#card-title');
  var unitEl = self.ctx.$container.querySelector('#unit-display');
  if (titleEl) titleEl.textContent = title;
  if (unitEl) unitEl.textContent = unit;
};

self.onDataUpdated = function(telemetry) {
  var metrics = telemetry && telemetry.metrics ? telemetry.metrics : {};
  var val = metrics.pressure !== undefined ? metrics.pressure : (metrics.temperature || 0);

  var displayEl = self.ctx.$container.querySelector('#metric-display');
  var pillEl = self.ctx.$container.querySelector('#status-pill');
  var timeEl = self.ctx.$container.querySelector('#timestamp-display');

  if (displayEl) displayEl.textContent = Number(val).toFixed(2);
  if (timeEl && self.ctx.moment) {
    var rawTs = telemetry.received_at || telemetry.timestamp;
    timeEl.textContent = self.ctx.moment(rawTs).format('HH:mm:ss');
  }

  var highThreshold = Number(self.ctx.settings.highThreshold) || 75;
  if (pillEl) {
    if (val > highThreshold) {
      pillEl.textContent = 'HIGH ALERT';
      pillEl.className = 'px-2 py-0.5 text-[10px] font-bold rounded-full bg-red-500/20 text-red-400 border border-red-500/30';
    } else {
      pillEl.textContent = 'NORMAL';
      pillEl.className = 'px-2 py-0.5 text-[10px] font-bold rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30';
    }
  }
};

self.onResize = function() {};
self.onDestroy = function() {};`,
    settings_schema: {
      title: 'Pressure Telemetry',
      unit: 'bar',
      highThreshold: 75,
    },
  },
  {
    id: 'battery_connectivity_monitor',
    name: 'Battery & Signal Health Monitor',
    description: 'Animated battery progress bar and RSSI wireless strength meter',
    widget_type: 'telemetry',
    template_html: `<div class="battery-widget p-5 bg-white dark:bg-slate-900 rounded-2xl flex flex-col justify-between h-full border border-slate-200 dark:border-slate-800">
  <div class="flex items-center justify-between">
    <span class="text-xs font-bold text-slate-700 dark:text-slate-300">Device Power & Link</span>
    <span id="link-quality" class="text-[11px] font-mono font-semibold text-cyan-600 dark:text-cyan-400">RSSI: -dBm</span>
  </div>
  <div class="my-3">
    <div class="flex items-center justify-between text-xs mb-1 font-semibold">
      <span class="text-slate-500">Battery Level</span>
      <span id="battery-pct" class="text-slate-800 dark:text-slate-200">--%</span>
    </div>
    <div class="w-full bg-slate-200 dark:bg-slate-800 h-3 rounded-full overflow-hidden p-0.5">
      <div id="battery-bar" class="h-full bg-emerald-500 rounded-full transition-all duration-300" style="width: 0%;"></div>
    </div>
  </div>
  <div class="text-[11px] text-slate-400 flex items-center justify-between pt-2 border-t border-slate-100 dark:border-slate-800">
    <span>Status: <span id="health-status" class="font-semibold text-slate-700 dark:text-slate-300">Active</span></span>
    <span id="sync-time">Polling...</span>
  </div>
</div>`,
    style_css: `.battery-widget {
  min-height: 160px;
}
#battery-bar.low {
  background-color: #ef4444 !important;
}
#battery-bar.medium {
  background-color: #f59e0b !important;
}`,
    controller_js: `self.onInit = function() {};

self.onDataUpdated = function(telemetry) {
  var metrics = telemetry && telemetry.metrics ? telemetry.metrics : {};
  var battery = metrics.battery !== undefined ? metrics.battery : 82;
  var rssi = metrics.rssi !== undefined ? metrics.rssi : -68;

  var pctEl = self.ctx.$container.querySelector('#battery-pct');
  var barEl = self.ctx.$container.querySelector('#battery-bar');
  var rssiEl = self.ctx.$container.querySelector('#link-quality');
  var syncEl = self.ctx.$container.querySelector('#sync-time');

  if (pctEl) pctEl.textContent = Math.round(battery) + '%';
  if (barEl) {
    barEl.style.width = Math.min(100, Math.max(0, battery)) + '%';
    barEl.classList.remove('low', 'medium');
    if (battery < 20) barEl.classList.add('low');
    else if (battery < 50) barEl.classList.add('medium');
  }
  if (rssiEl) rssiEl.textContent = 'RSSI: ' + rssi + ' dBm';
  if (syncEl && self.ctx.moment) syncEl.textContent = self.ctx.moment().format('HH:mm:ss');
};

self.onResize = function() {};
self.onDestroy = function() {};`,
    settings_schema: {
      lowThreshold: 20,
      mediumThreshold: 50,
    },
  },
];
