export type User = { id: string; email: string; email_verified: boolean; full_name: string; avatar_url: string; date_of_birth: string | null };
export type TokenResponse = { access_token: string; token_type: "Bearer"; expires_in: number; user: User };
