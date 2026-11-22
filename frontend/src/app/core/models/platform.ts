/** A ЖК as the platform operator sees it (GET /platform/jks). */
export interface PlatformJK {
  id: string;
  name: string;
  address: string;
  cadastral_number: string;
  owner_id: string;
  created_at: string;
  storages_count: number;
  admins_count: number;
  residents_count: number;
}

/** An ОСИ account of a ЖК. */
export interface PlatformAdmin {
  id: string;
  full_name: string;
  email: string;
  phone: string;
  jk_id: string | null;
  created_at: string;
  must_change_password: boolean;
}

export interface JKForm {
  name: string;
  address: string;
  cadastral_number: string;
}

export interface AccountForm {
  full_name: string;
  email: string;
  phone: string;
}

export interface CreateJKPayload extends JKForm {
  admin: AccountForm;
}

/** The temporary password is returned once and never again. */
export interface CreateJKResponse {
  jk: PlatformJK;
  admin: PlatformAdmin;
  temporary_password: string;
}

export interface AddAdminResponse {
  admin: PlatformAdmin;
  temporary_password: string;
}
