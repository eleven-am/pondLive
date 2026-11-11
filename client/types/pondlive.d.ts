export interface Channel<Down, Up, Props = undefined> {
    on(fn: (msg: Down) => void): void;
    send(msg: Up): void;
    readonly props: Props;
    onProps(fn: (props: Props) => void): void;
}

export type Cleanup = () => void;

export type ScriptModule<Down, Up, Props = undefined> = (
    el: Element,
    io: Channel<Down, Up, Props>,
) => void | Cleanup | Promise<void | Cleanup>;
