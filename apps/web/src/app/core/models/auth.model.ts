export type UserRole = 'admin' | 'operator' | 'viewer';

export interface User {
  id: string;
  organization_id: string;
  email: string;
  role: UserRole;
  created_at: string;
  updated_at: string;
}

export interface Organization {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
  organization?: Organization;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export interface RefreshTokenPayload {
  refresh_token: string;
}

export interface CreateOrgUserPayload {
  email: string;
  password: string;
  role: UserRole;
}
