import type { MetadataEffect, MetadataTagPayload, LinkTagPayload, ScriptTagPayload } from './types';
import { Logger } from './logger';

export class MetadataManager {
  private metaTags = new Map<string, HTMLMetaElement>();
  private linkTags = new Map<string, HTMLLinkElement>();
  private scriptTags = new Map<string, HTMLScriptElement>();
  private descriptionMeta: HTMLMetaElement | null = null;

  constructor() {
    this.indexExistingTags();
  }

  private indexExistingTags(): void {
    if (typeof document === 'undefined') {
      return;
    }

    document.querySelectorAll('meta[data-live-key]').forEach((el) => {
      if (el instanceof HTMLMetaElement) {
        const key = el.getAttribute('data-live-key');
        if (key) {
          this.metaTags.set(key, el);
          if (el.name === 'description') {
            this.descriptionMeta = el;
          }
        }
      }
    });

    document.querySelectorAll('link[data-live-key]').forEach((el) => {
      if (el instanceof HTMLLinkElement) {
        const key = el.getAttribute('data-live-key');
        if (key) {
          this.linkTags.set(key, el);
        }
      }
    });

    document.querySelectorAll('script[data-live-key]').forEach((el) => {
      if (el instanceof HTMLScriptElement) {
        const key = el.getAttribute('data-live-key');
        if (key) {
          this.scriptTags.set(key, el);
        }
      }
    });
  }

  applyEffect(effect: MetadataEffect): void {
    if (typeof document === 'undefined') {
      return;
    }

    Logger.debug('[Metadata]', 'applying effect', effect);

    if (effect.title !== undefined) {
      document.title = effect.title;
    }

    if (effect.description !== undefined) {
      this.updateDescription(effect.description);
    } else if (effect.clearDescription) {
      this.clearDescription();
    }

    if (effect.metaRemove) {
      for (const key of effect.metaRemove) {
        this.removeMeta(key);
      }
    }

    if (effect.metaAdd) {
      for (const payload of effect.metaAdd) {
        this.addOrUpdateMeta(payload);
      }
    }

    if (effect.linkRemove) {
      for (const key of effect.linkRemove) {
        this.removeLink(key);
      }
    }

    if (effect.linkAdd) {
      for (const payload of effect.linkAdd) {
        this.addOrUpdateLink(payload);
      }
    }

    if (effect.scriptRemove) {
      for (const key of effect.scriptRemove) {
        this.removeScript(key);
      }
    }

    if (effect.scriptAdd) {
      for (const payload of effect.scriptAdd) {
        this.addOrUpdateScript(payload);
      }
    }
  }

  private updateDescription(content: string): void {
    if (!this.descriptionMeta) {
      this.descriptionMeta = document.createElement('meta');
      this.descriptionMeta.name = 'description';
      this.descriptionMeta.setAttribute('data-live-managed', 'true');
      document.head.appendChild(this.descriptionMeta);
    }
    this.descriptionMeta.content = content;
  }

  private clearDescription(): void {
    if (this.descriptionMeta && this.descriptionMeta.hasAttribute('data-live-managed')) {
      this.descriptionMeta.remove();
      this.descriptionMeta = null;
    }
  }

  private addOrUpdateMeta(payload: MetadataTagPayload): void {
    let el = this.metaTags.get(payload.key);

    if (!el) {
      el = document.createElement('meta');
      el.setAttribute('data-live-key', payload.key);
      document.head.appendChild(el);
      this.metaTags.set(payload.key, el);
    }

    if (payload.name) el.name = payload.name;
    if (payload.content !== undefined) el.content = payload.content;
    if (payload.property) el.setAttribute('property', payload.property);
    if (payload.charset) el.setAttribute('charset', payload.charset);
    if (payload.httpEquiv) el.setAttribute('http-equiv', payload.httpEquiv);
    if (payload.itemProp) el.setAttribute('itemprop', payload.itemProp);

    if (payload.attrs) {
      for (const [key, value] of Object.entries(payload.attrs)) {
        el.setAttribute(key, value);
      }
    }

    if (payload.name === 'description') {
      this.descriptionMeta = el;
    }
  }

  private removeMeta(key: string): void {
    const el = this.metaTags.get(key);
    if (el) {
      if (el === this.descriptionMeta) {
        this.descriptionMeta = null;
      }
      el.remove();
      this.metaTags.delete(key);
    }
  }

  private addOrUpdateLink(payload: LinkTagPayload): void {
    let el = this.linkTags.get(payload.key);

    if (!el) {
      el = document.createElement('link');
      el.setAttribute('data-live-key', payload.key);
      document.head.appendChild(el);
      this.linkTags.set(payload.key, el);
    }

    if (payload.rel) el.rel = payload.rel;
    if (payload.href) el.href = payload.href;
    if (payload.type) el.type = payload.type;
    if (payload.as) el.setAttribute('as', payload.as);
    if (payload.media) el.media = payload.media;
    if (payload.hreflang) el.hreflang = payload.hreflang;
    if (payload.title) el.title = payload.title;
    if (payload.crossorigin) el.setAttribute('crossorigin', payload.crossorigin);
    if (payload.integrity) el.integrity = payload.integrity;
    if (payload.referrerpolicy) el.setAttribute('referrerpolicy', payload.referrerpolicy);
    if (payload.sizes) el.setAttribute('sizes', payload.sizes);

    if (payload.attrs) {
      for (const [key, value] of Object.entries(payload.attrs)) {
        el.setAttribute(key, value);
      }
    }
  }

  private removeLink(key: string): void {
    const el = this.linkTags.get(key);
    if (el) {
      el.remove();
      this.linkTags.delete(key);
    }
  }

  private addOrUpdateScript(payload: ScriptTagPayload): void {
    const existing = this.scriptTags.get(payload.key);
    if (existing) {
      existing.remove();
      this.scriptTags.delete(payload.key);
    }

    const el = document.createElement('script');
    el.setAttribute('data-live-key', payload.key);

    if (payload.src) el.src = payload.src;
    if (payload.type) el.type = payload.type;
    if (payload.async) el.async = true;
    if (payload.defer) el.defer = true;
    if (payload.module) el.type = 'module';
    if (payload.noModule) el.setAttribute('nomodule', '');
    if (payload.crossorigin) el.setAttribute('crossorigin', payload.crossorigin);
    if (payload.integrity) el.integrity = payload.integrity;
    if (payload.referrerpolicy) el.setAttribute('referrerpolicy', payload.referrerpolicy);
    if (payload.nonce) el.nonce = payload.nonce;
    if (payload.inner) el.textContent = payload.inner;

    if (payload.attrs) {
      for (const [key, value] of Object.entries(payload.attrs)) {
        el.setAttribute(key, value);
      }
    }

    document.head.appendChild(el);
    this.scriptTags.set(payload.key, el);
  }

  private removeScript(key: string): void {
    const el = this.scriptTags.get(key);
    if (el) {
      el.remove();
      this.scriptTags.delete(key);
    }
  }

  dispose(): void {
    this.metaTags.forEach((el) => {
      if (el.hasAttribute('data-live-key')) {
        el.remove();
      }
    });
    this.linkTags.forEach((el) => {
      if (el.hasAttribute('data-live-key')) {
        el.remove();
      }
    });
    this.scriptTags.forEach((el) => {
      if (el.hasAttribute('data-live-key')) {
        el.remove();
      }
    });
    this.metaTags.clear();
    this.linkTags.clear();
    this.scriptTags.clear();
    this.descriptionMeta = null;
  }
}
