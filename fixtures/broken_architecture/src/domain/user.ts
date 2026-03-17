import { Logger } from "../infra/logger";

export class UserService {
    private logger: Logger;
    
    constructor() {
        this.logger = new Logger();
    }
}
