import { format } from "./lib/format";
import type { Channel } from "./lib/types";

interface Down { add: number }
interface Up { value: string }

export default function (el: Element, io: Channel<Down, Up>) {
  let total = 0;
  io.on((m) => {
    total += m.add;
    io.send({ value: format(total) });
  });
  return () => {
    total = 0;
  };
}
