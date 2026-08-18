export interface User {
  id: string;
  email: string;
  role: 'admin' | 'resident';
  full_name?: string;
  phone?: string;
  jk_id?: string;
}

export interface LoginPayload {
  email?: string;
  phone?: string;
  password: string;
}
