import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { MetadataManager } from '../src/metadata';
import type { MetadataEffect } from '../src/types';

describe('MetadataManager', () => {
  let manager: MetadataManager;

  beforeEach(() => {
    document.head.innerHTML = '';
    manager = new MetadataManager();
  });

  afterEach(() => {
    manager.dispose();
  });

  describe('title', () => {
    it('should update document title', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        title: 'Test Page',
      };
      manager.applyEffect(effect);
      expect(document.title).toBe('Test Page');
    });

    it('should update title multiple times', () => {
      manager.applyEffect({ type: 'metadata', title: 'First' });
      expect(document.title).toBe('First');
      manager.applyEffect({ type: 'metadata', title: 'Second' });
      expect(document.title).toBe('Second');
    });
  });

  describe('description', () => {
    it('should create description meta tag', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        description: 'Test description',
      };
      manager.applyEffect(effect);

      const meta = document.querySelector('meta[name="description"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('Test description');
    });

    it('should update existing description', () => {
      manager.applyEffect({ type: 'metadata', description: 'First' });
      manager.applyEffect({ type: 'metadata', description: 'Second' });

      const metas = document.querySelectorAll('meta[name="description"]');
      expect(metas.length).toBe(1);
      expect(metas[0]?.getAttribute('content')).toBe('Second');
    });

    it('should clear description when clearDescription is true', () => {
      manager.applyEffect({ type: 'metadata', description: 'Test' });
      expect(document.querySelector('meta[name="description"]')).toBeTruthy();

      manager.applyEffect({ type: 'metadata', clearDescription: true });
      expect(document.querySelector('meta[name="description"]')).toBeFalsy();
    });
  });

  describe('meta tags', () => {
    it('should add meta tag with name and content', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        metaAdd: [
          {
            key: 'meta:name:keywords',
            name: 'keywords',
            content: 'test, keywords',
          },
        ],
      };
      manager.applyEffect(effect);

      const meta = document.querySelector('meta[name="keywords"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('test, keywords');
      expect(meta?.getAttribute('data-live-key')).toBe('meta:name:keywords');
    });

    it('should add meta tag with property (OpenGraph)', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        metaAdd: [
          {
            key: 'meta:property:og:title',
            property: 'og:title',
            content: 'Test Title',
          },
        ],
      };
      manager.applyEffect(effect);

      const meta = document.querySelector('meta[property="og:title"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('Test Title');
    });

    it('should add meta tag with charset', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        metaAdd: [
          {
            key: 'meta:charset:utf-8',
            charset: 'utf-8',
          },
        ],
      };
      manager.applyEffect(effect);

      const meta = document.querySelector('meta[charset="utf-8"]');
      expect(meta).toBeTruthy();
    });

    it('should add meta tag with http-equiv', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        metaAdd: [
          {
            key: 'meta:http-equiv:refresh',
            httpEquiv: 'refresh',
            content: '30',
          },
        ],
      };
      manager.applyEffect(effect);

      const meta = document.querySelector('meta[http-equiv="refresh"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('30');
    });

    it('should add meta tag with custom attributes', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        metaAdd: [
          {
            key: 'meta:custom',
            name: 'custom',
            content: 'value',
            attrs: {
              'data-custom': 'test',
              'data-other': 'value',
            },
          },
        ],
      };
      manager.applyEffect(effect);

      const meta = document.querySelector('meta[name="custom"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('data-custom')).toBe('test');
      expect(meta?.getAttribute('data-other')).toBe('value');
    });

    it('should update existing meta tag', () => {
      manager.applyEffect({
        type: 'metadata',
        metaAdd: [{ key: 'meta:name:keywords', name: 'keywords', content: 'first' }],
      });
      manager.applyEffect({
        type: 'metadata',
        metaAdd: [{ key: 'meta:name:keywords', name: 'keywords', content: 'second' }],
      });

      const metas = document.querySelectorAll('meta[name="keywords"]');
      expect(metas.length).toBe(1);
      expect(metas[0]?.getAttribute('content')).toBe('second');
    });

    it('should remove meta tag', () => {
      manager.applyEffect({
        type: 'metadata',
        metaAdd: [{ key: 'meta:name:keywords', name: 'keywords', content: 'test' }],
      });
      expect(document.querySelector('meta[name="keywords"]')).toBeTruthy();

      manager.applyEffect({
        type: 'metadata',
        metaRemove: ['meta:name:keywords'],
      });
      expect(document.querySelector('meta[name="keywords"]')).toBeFalsy();
    });
  });

  describe('link tags', () => {
    it('should add link tag', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        linkAdd: [
          {
            key: 'link:rel:stylesheet|href:/style.css',
            rel: 'stylesheet',
            href: '/style.css',
          },
        ],
      };
      manager.applyEffect(effect);

      const link = document.querySelector('link[rel="stylesheet"]');
      expect(link).toBeTruthy();
      expect(link?.getAttribute('href')).toBe('/style.css');
      expect(link?.getAttribute('data-live-key')).toBe('link:rel:stylesheet|href:/style.css');
    });

    it('should add link with all attributes', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        linkAdd: [
          {
            key: 'link:complex',
            rel: 'preload',
            href: '/font.woff2',
            type: 'font/woff2',
            as: 'font',
            crossorigin: 'anonymous',
            attrs: {
              'data-custom': 'value',
            },
          },
        ],
      };
      manager.applyEffect(effect);

      const link = document.querySelector('link[rel="preload"]');
      expect(link).toBeTruthy();
      expect(link?.getAttribute('as')).toBe('font');
      expect(link?.getAttribute('type')).toBe('font/woff2');
      expect(link?.getAttribute('crossorigin')).toBe('anonymous');
      expect(link?.getAttribute('data-custom')).toBe('value');
    });

    it('should update existing link tag', () => {
      manager.applyEffect({
        type: 'metadata',
        linkAdd: [{ key: 'link:icon', rel: 'icon', href: '/old.ico' }],
      });
      manager.applyEffect({
        type: 'metadata',
        linkAdd: [{ key: 'link:icon', rel: 'icon', href: '/new.ico' }],
      });

      const links = document.querySelectorAll('link[rel="icon"]');
      expect(links.length).toBe(1);
      expect(links[0]?.getAttribute('href')).toBe('/new.ico');
    });

    it('should remove link tag', () => {
      manager.applyEffect({
        type: 'metadata',
        linkAdd: [{ key: 'link:stylesheet', rel: 'stylesheet', href: '/style.css' }],
      });
      expect(document.querySelector('link[rel="stylesheet"]')).toBeTruthy();

      manager.applyEffect({
        type: 'metadata',
        linkRemove: ['link:stylesheet'],
      });
      expect(document.querySelector('link[rel="stylesheet"]')).toBeFalsy();
    });
  });

  describe('script tags', () => {
    it('should add script tag with src', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        scriptAdd: [
          {
            key: 'script:src:/app.js',
            src: '/app.js',
          },
        ],
      };
      manager.applyEffect(effect);

      const script = document.querySelector('script[src="/app.js"]');
      expect(script).toBeTruthy();
      expect(script?.getAttribute('data-live-key')).toBe('script:src:/app.js');
    });

    it('should add script with async and defer', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        scriptAdd: [
          {
            key: 'script:analytics',
            src: '/analytics.js',
            async: true,
            defer: true,
          },
        ],
      };
      manager.applyEffect(effect);

      const script = document.querySelector('script[src="/analytics.js"]') as HTMLScriptElement;
      expect(script).toBeTruthy();
      expect(script.async).toBe(true);
      expect(script.defer).toBe(true);
    });

    it('should add inline script', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        scriptAdd: [
          {
            key: 'script:inline:config',
            inner: 'window.config = { test: true };',
          },
        ],
      };
      manager.applyEffect(effect);

      const script = document.querySelector('script[data-live-key="script:inline:config"]');
      expect(script).toBeTruthy();
      expect(script?.textContent).toBe('window.config = { test: true };');
    });

    it('should add module script', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        scriptAdd: [
          {
            key: 'script:module',
            src: '/module.js',
            module: true,
          },
        ],
      };
      manager.applyEffect(effect);

      const script = document.querySelector('script[src="/module.js"]') as HTMLScriptElement;
      expect(script).toBeTruthy();
      expect(script.type).toBe('module');
    });

    it('should add script with custom attributes', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        scriptAdd: [
          {
            key: 'script:custom',
            src: '/script.js',
            attrs: {
              'data-custom': 'value',
              'data-other': 'test',
            },
          },
        ],
      };
      manager.applyEffect(effect);

      const script = document.querySelector('script[src="/script.js"]');
      expect(script).toBeTruthy();
      expect(script?.getAttribute('data-custom')).toBe('value');
      expect(script?.getAttribute('data-other')).toBe('test');
    });

    it('should replace script when key exists', () => {
      manager.applyEffect({
        type: 'metadata',
        scriptAdd: [{ key: 'script:app', src: '/old.js' }],
      });
      const oldScript = document.querySelector('script[src="/old.js"]');
      expect(oldScript).toBeTruthy();

      manager.applyEffect({
        type: 'metadata',
        scriptAdd: [{ key: 'script:app', src: '/new.js' }],
      });

      expect(document.querySelector('script[src="/old.js"]')).toBeFalsy();
      expect(document.querySelector('script[src="/new.js"]')).toBeTruthy();
    });

    it('should remove script tag', () => {
      manager.applyEffect({
        type: 'metadata',
        scriptAdd: [{ key: 'script:test', src: '/test.js' }],
      });
      expect(document.querySelector('script[src="/test.js"]')).toBeTruthy();

      manager.applyEffect({
        type: 'metadata',
        scriptRemove: ['script:test'],
      });
      expect(document.querySelector('script[src="/test.js"]')).toBeFalsy();
    });
  });

  describe('combined effects', () => {
    it('should apply multiple changes at once', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        title: 'Combined Test',
        description: 'Test description',
        metaAdd: [
          { key: 'meta:keywords', name: 'keywords', content: 'test' },
          { key: 'meta:og', property: 'og:title', content: 'OG Title' },
        ],
        linkAdd: [
          { key: 'link:icon', rel: 'icon', href: '/icon.png' },
        ],
        scriptAdd: [
          { key: 'script:app', src: '/app.js' },
        ],
      };
      manager.applyEffect(effect);

      expect(document.title).toBe('Combined Test');
      expect(document.querySelector('meta[name="description"]')).toBeTruthy();
      expect(document.querySelector('meta[name="keywords"]')).toBeTruthy();
      expect(document.querySelector('meta[property="og:title"]')).toBeTruthy();
      expect(document.querySelector('link[rel="icon"]')).toBeTruthy();
      expect(document.querySelector('script[src="/app.js"]')).toBeTruthy();
    });

    it('should handle add and remove in same effect', () => {
      manager.applyEffect({
        type: 'metadata',
        metaAdd: [
          { key: 'meta:old', name: 'old', content: 'value' },
          { key: 'meta:keep', name: 'keep', content: 'value' },
        ],
      });

      manager.applyEffect({
        type: 'metadata',
        metaAdd: [
          { key: 'meta:new', name: 'new', content: 'value' },
        ],
        metaRemove: ['meta:old'],
      });

      expect(document.querySelector('meta[name="old"]')).toBeFalsy();
      expect(document.querySelector('meta[name="keep"]')).toBeTruthy();
      expect(document.querySelector('meta[name="new"]')).toBeTruthy();
    });
  });

  describe('dispose', () => {
    it('should remove all managed tags on dispose', () => {
      const effect: MetadataEffect = {
        type: 'metadata',
        metaAdd: [{ key: 'meta:test', name: 'test', content: 'value' }],
        linkAdd: [{ key: 'link:test', rel: 'stylesheet', href: '/test.css' }],
        scriptAdd: [{ key: 'script:test', src: '/test.js' }],
      };
      manager.applyEffect(effect);

      expect(document.querySelector('meta[data-live-key]')).toBeTruthy();
      expect(document.querySelector('link[data-live-key]')).toBeTruthy();
      expect(document.querySelector('script[data-live-key]')).toBeTruthy();

      manager.dispose();

      expect(document.querySelector('meta[data-live-key]')).toBeFalsy();
      expect(document.querySelector('link[data-live-key]')).toBeFalsy();
      expect(document.querySelector('script[data-live-key]')).toBeFalsy();
    });
  });
});
