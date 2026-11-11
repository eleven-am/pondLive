export interface Channel<Down, Up> {
  on(fn: (m: Down) => void): void;
  send(m: Up): void;
}
