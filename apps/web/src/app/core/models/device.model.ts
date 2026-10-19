export * from './telemetry.model';

export type DeviceStatus = 'online' | 'offline' | 'disabled' | 'revoked';

export interface Device {
  id: string;
  organization_id: string;
  name: string;
  device_type: string;
  status: DeviceStatus;
  metadata?: Record<string, unknown>;
  last_seen?: string | null;
  created_at: string;
  updated_at: string;
}

export interface DeviceCredential {
  id: string;
  device_id: string;
  organization_id: string;
  credential_type: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ProvisionResponse {
  device_id: string;
  organization_id: string;
  credential_id: string;
  credential_type: string;
  token: string;
  status: string;
  created_at: string;
  notice: string;
}

export interface CreateDevicePayload {
  id?: string;
  name: string;
  device_type: string;
  metadata?: Record<string, unknown>;
}

export interface UpdateDevicePayload {
  name?: string;
  device_type?: string;
  metadata?: Record<string, unknown>;
}

export interface DeviceStatusActionResponse {
  device_id: string;
  status: string;
  message: string;
  updated_at: string;
}
