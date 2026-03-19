import { UserService } from "../services/user.service";

export function route() {
  return new UserService();
}
