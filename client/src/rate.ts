interface Slot {
    run: () => void;
}

const queued = new Set<Slot>();

export function flush(upTo?: Slot): void {
    for (const s of [...queued]) {
        queued.delete(s);
        s.run();
        if (s === upTo) break;
    }
}

export function limit<D>(db: number | undefined, th: number | undefined, send: (d: D) => void): (d: D) => void {
    if (db === undefined && th === undefined) {
        return (d) => {
            flush();
            send(d);
        };
    }
    let data: D;
    let timer: any;
    let busy = false;
    const frame = th === 0;
    const slot: Slot = {
        run: () => {
            send(data);
            if (db === undefined) arm();
        },
    };
    const end = () => {
        busy = false;
        if (queued.has(slot)) flush(slot);
    };
    const arm = () => {
        frame ? cancelAnimationFrame(timer) : clearTimeout(timer);
        busy = true;
        timer = frame ? requestAnimationFrame(end) : setTimeout(end, th);
    };
    const later = () => {
        queued.delete(slot);
        queued.add(slot);
    };
    return (d) => {
        data = d;
        if (db !== undefined) {
            later();
            clearTimeout(timer);
            timer = setTimeout(() => flush(slot), db);
        } else if (busy) {
            later();
        } else {
            flush();
            send(d);
            arm();
        }
    };
}
