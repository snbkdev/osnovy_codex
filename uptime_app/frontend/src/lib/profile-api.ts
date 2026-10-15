import { authRequest, updateCurrentUser } from "./session";
import type { User } from "./auth-types";

export function saveProfile(fullName: string, dateOfBirth: string) {
  return updateCurrentUser(() => authRequest<User>("profile", { full_name: fullName, date_of_birth: dateOfBirth || null }));
}

export function saveAvatar(file: Blob) {
  const body = new FormData();
  body.append("avatar", file, "avatar.png");
  return updateCurrentUser(() => authRequest<User>("avatar", body));
}
