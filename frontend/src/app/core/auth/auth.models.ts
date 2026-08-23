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

export interface LoginResponse extends User {
  access_token: string;
  token_type: string;
}
