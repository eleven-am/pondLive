export class FakeSocket {
    static instances: FakeSocket[] = [];

    onopen: (() => void) | null = null;
    onmessage: ((event: { data: unknown }) => void) | null = null;
    onclose: ((event: { code: number }) => void) | null = null;
    sent: string[] = [];
    closed = false;
    readyState = 1;

    constructor(readonly url: string) {
        FakeSocket.instances.push(this);
    }

    send(data: string): void {
        this.sent.push(data);
    }

    close(): void {
        this.closed = true;
        this.readyState = 3;
        this.onclose?.({ code: 1006 });
    }

    open(): void {
        this.onopen?.();
    }

    receive(frame: unknown): void {
        this.onmessage?.({ data: JSON.stringify(frame) });
    }

    drop(code = 1006): void {
        this.onclose?.({ code });
    }

    frames(): Array<Record<string, unknown>> {
        return this.sent.map((s) => JSON.parse(s));
    }

    static latest(): FakeSocket {
        return FakeSocket.instances[FakeSocket.instances.length - 1];
    }
}
