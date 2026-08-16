export interface User {
  id: string;
  email: string;
  role: 'admin' | 'superadmin' | 'client';
  full_name?: string;
  jk_id?: string;
}

export interface LoginPayload {
  email: string;
  password_hash: string;
}
