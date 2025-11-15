import type { ListPathDescriptor } from './types';
import { DomRegistry } from './dom-registry';

export class ListManager {
  constructor(private dom: DomRegistry) {}

  registerLists(descriptors?: ListPathDescriptor[]): void {
    this.dom.registerLists(descriptors);
  }
}
