import type { ClientChannel } from '@eleven-am/pondsocket/types';


import { pondEventHandler } from './handler';
import { pondFormInit } from './pondFormEvents';
import { pondKeyboardEvents } from './pondKeyboardEvents';
import { pondMouseEvents } from './pondMouseEvents';
import { buildEventListener } from '../actors/channelEvent';
import { DomWatcher } from '../html/domWatcher';

export function initEventListeners (channel: ClientChannel, watcher: DomWatcher, userId: string): void {
    const handler = pondEventHandler(channel, watcher);
    const listener = buildEventListener(channel);

    pondKeyboardEvents(handler);
    pondFormInit(handler, listener, userId);
    pondMouseEvents(handler, listener, userId);
}
