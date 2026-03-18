import { UserService as Svc } from "@/services/user.service";

export function handler() {
  return new Svc();
}
