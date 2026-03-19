import { UserService } from "../services/user.service";

export function handleRequest() {
  return new UserService();
}
