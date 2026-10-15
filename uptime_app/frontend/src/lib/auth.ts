// Public facade; transport, session state and domain operations live separately.
export type { User, TokenResponse } from "./auth-types";
export { ApiError } from "./http-client";
export {
  getAuthState, getServerAuthState, subscribeAuth, initializeAuth,
  refreshSession, login, logout, authRequest, publicAuthRequest,
} from "./session";
export { saveProfile, saveAvatar } from "./profile-api";
