import { UserService } from "../services/user.service";

export function createController() {
  return new UserService();
}
