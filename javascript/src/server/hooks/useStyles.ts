import { LiveContext } from '../context/liveContext';
import { html, Html } from '../parser/parser';

type CSSProperties = Record<string, string | number>;

function hyphenate (string: string): string {
    return string.replace(/[A-Z]|^ms/g, '-$&').toLowerCase();
}

function styleToCSS (style: CSSProperties): string {
    return Object.keys(style).map((key) => html`${hyphenate(key)}:${style[key]}`)
        .join(';');
}

type CSSClasses = Record<string, CSSProperties>;

function classesToCSS (refId: string, classes: CSSClasses): Html {
    return html`<style>${Object.keys(classes).map((key) => html`.${refId}-${key}{${styleToCSS(classes[key])}}`)}</style>`;
}

interface CSSObject {
    css: Html;
    classes: Record<string, string>;
}

function classesToObj (refId: string, classes: CSSClasses): CSSObject {
    const css = classesToCSS(refId, classes);
    const classesObj: Record<string, string> = {};

    Object.keys(classes).forEach((key) => {
        classesObj[key] = `${refId}-${key}`;
    });

    return {
        css,
        classes: classesObj,
    };
}

type CSSGenerator = (props: any) => CSSClasses;
export function makeStyles<A extends CSSClasses> (classes: A): (context: LiveContext) => { [K in keyof A]: string };
export function makeStyles<B extends CSSGenerator> (classes: B): (context: LiveContext, props: Parameters<B>[0]) => { [K in keyof ReturnType<B>]: string };

export function makeStyles (classes: CSSGenerator | CSSClasses) {
    return (context: LiveContext, props?: any) => {
        const { css, classes: classesObj } = classesToObj(context.manager.id, typeof classes === 'function' ? classes(props) : classes);

        context.addStyle(css);

        return classesObj as any;
    };
}
