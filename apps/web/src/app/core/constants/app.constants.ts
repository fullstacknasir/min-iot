export const DEFAULT_API_BASE_URL = 'http://localhost:8080/api/v1';
export const HEALTH_CHECK_URL = 'http://localhost:8080/health';
export const LIVE_TELEMETRY_STREAM_URL = 'http://localhost:8080/api/v1/devices';

export const STORAGE_ACCESS_TOKEN_KEY = 'miniot_access_token';
export const STORAGE_REFRESH_TOKEN_KEY = 'miniot_refresh_token';
export const STORAGE_USER_KEY = 'miniot_current_user';
export const STORAGE_ORG_KEY = 'miniot_current_org';

export const ROLE_ADMIN = 'admin';
export const ROLE_OPERATOR = 'operator';
export const ROLE_VIEWER = 'viewer';

export const STATUS_ONLINE = 'online';
export const STATUS_OFFLINE = 'offline';
export const STATUS_DISABLED = 'disabled';
export const STATUS_REVOKED = 'revoked';

export const TIME_RANGE_1H = '1h';
export const TIME_RANGE_6H = '6h';
export const TIME_RANGE_24H = '24h';
export const TIME_RANGE_7D = '7d';
export const TIME_RANGE_30D = '30d';

export const DEFAULT_AGGREGATE_BUCKET = '5m';
export const DEFAULT_HISTORY_LIMIT = 100;
export const DEFAULT_REFRESH_INTERVAL_MS = 5000;
export const SSE_RECONNECT_INTERVAL_MS = 3000;

export const FORMAT_DATETIME_DISPLAY = 'YYYY-MM-DD HH:mm:ss';
export const FORMAT_TIME_ONLY = 'HH:mm:ss';
export const FORMAT_DATE_ONLY = 'YYYY-MM-DD';
export const FORMAT_CHART_AXIS = 'HH:mm:ss';

export const DEFAULT_PAGE = 1;
export const DEFAULT_PAGE_LIMIT = 10;
export const DEFAULT_GRID_PAGE_LIMIT = 6;
export const PAGE_SIZE_OPTIONS: readonly number[] = [5, 10, 20, 50];
export const GRID_PAGE_SIZE_OPTIONS: readonly number[] = [6, 9, 12, 24];
