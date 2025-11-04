import type { ClientChannel } from '@eleven-am/pondsocket/types';

import { LiveEvent } from '../types';

export const emitEvent = (event: string, data: any, element?: Element) => {
    const eventEmitter = new CustomEvent(event, {
        detail: data,
        bubbles: true,
        cancelable: true,
    });

    if (element) {
        element.dispatchEvent(eventEmitter);
    } else {
        window.dispatchEvent(eventEmitter);
    }
};

export const broadcaster = (channel: ClientChannel, data: LiveEvent) => {
    channel.broadcastFrom('event', data);
};

export const streamLinedBroadcaster = (channel: ClientChannel, event: string, data: LiveEvent) => {
    channel.broadcastFrom(event, data);
};
