export type UserStatus = 'pending' | 'active' | 'rejected' | 'blocked';
export type OwnerType = 'person' | 'company';
export type PremisesKind = 'flat' | 'commercial';

/** What a user declares about the property they own in the ЖК. */
export interface Verification {
  owner_type: OwnerType;
  premises_kind: PremisesKind;
  premises_number: string;
  company_bin?: string;
  company_name?: string;
}

export interface User {
  id: string;
  email: string;
  role: 'admin' | 'resident';
  full_name?: string;
  phone?: string;
  jk_id?: string;
  iin?: string;
  status?: UserStatus;
  review_comment?: string;
  owner_type?: OwnerType;
  premises_kind?: PremisesKind;
  premises_number?: string;
  company_bin?: string;
  company_name?: string;
  created_at?: string;
}

export interface LoginPayload {
  login?: string; // IIN, email or phone
  email?: string;
  phone?: string;
  password: string;
}

export interface RegisterPayload extends Verification {
  /** Version of the resident agreement the user accepted on the form. */
  accepted_terms_version: string;
  full_name: string;
  phone: string;
  password: string;
  jk_id?: string;
  iin: string;
}

export interface LoginResponse extends User {
  access_token: string;
  token_type: string;
}
