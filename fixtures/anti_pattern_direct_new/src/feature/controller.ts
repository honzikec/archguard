import { UserService } from "../services/user.service";

export function createController() {
  const service = new UserService();
  return service;
}
