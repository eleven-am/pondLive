type UserId = string;
type Action = () => void | Promise<void>;

export class UserQueue {
    #userQueue: Map<UserId, Action[]>;

    #active: Set<UserId>;

    constructor () {
        this.#userQueue = new Map();
        this.#active = new Set();
    }

    public async add (userId: UserId, action: Action) {
        if (this.#active.has(userId)) {
            this.#userQueue.get(userId)?.push(action);
        } else {
            this.#active.add(userId);
            await action();

            while (this.#userQueue.get(userId)?.length) {
                const queuedAction = this.#userQueue.get(userId)?.shift();

                if (queuedAction) {
                    await queuedAction();
                }
            }

            this.#userQueue.delete(userId);
            this.#active.delete(userId);
        }
    }
}
