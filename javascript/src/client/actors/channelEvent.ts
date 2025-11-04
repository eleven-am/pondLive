import type { ClientChannel } from '@eleven-am/pondsocket/types';

export type ChannelEventHandler = <T>(event: string, callback: (data: T) => void) => () => void;

export function buildEventListener (channel: ClientChannel): ChannelEventHandler {
    return (event, callback, unsub = true) => {
        const unsubscribe = channel.onMessage((evt, message) => {
            if (evt === event) {
                if (unsub) {
                    unsubscribe();
                }

                return callback(message as any);
            }
        });

        return unsubscribe;
    };
}
