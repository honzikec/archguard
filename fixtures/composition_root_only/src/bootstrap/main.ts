import { UserService } from "../services/user.service";

export function boot() {
  return new UserService();
}
