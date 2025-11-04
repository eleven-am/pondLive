import { mergeObjects } from './differ';
import { makeStyles } from './makeStyles';
import { html } from './parser';

describe('html parser', () => {
    it('should parse a simple html string', () => {
        const className = 'test';
        const message = 'Hello World';
        const string = html`<div class="${className}">${message}</div>`;

        expect(string.getParts()).toEqual({
            0: 'test',
            1: 'Hello World',
            s: ['<div class="', '">', '</div>'],
        });
    });

    it('should parse a html string with multiple dynamic values', () => {
        const className = 'test';
        const message = 'Hello World';
        const secondClassName = 'test2';
        const secondMessage = 'Hello again';
        const string = html`<div class="${className}">${message}<span class="${secondClassName}">${secondMessage}</span></div>`;

        expect(string.getParts()).toEqual({
            0: 'test',
            1: 'Hello World',
            2: 'test2',
            3: 'Hello again',
            s: ['<div class="', '">', '<span class="', '">', '</span></div>'],
        });
    });

    it('should parse a html string with nested dynamic values', () => {
        const firsName = 'John';
        const lastName = 'Doe';
        const age = 42;
        const address = '1 Park Avenue';
        const className = 'test';
        const secondClassName = 'test2';

        const string = html`<div class="${className}">${html`<div class="${secondClassName}">${firsName} ${lastName} is ${age} years old and lives at ${address}</div>`}</div>`;

        expect(string.getParts()).toEqual({
            0: 'test',
            1: {
                0: 'test2',
                1: 'John',
                2: 'Doe',
                3: 42,
                4: '1 Park Avenue',
                s: ['<div class="', '">', ' ', ' is ', ' years old and lives at ', '</div>'],
            },
            s: ['<div class="', '">', '</div>'],
        });
    });

    it('should parse a html string with nested dynamic values and multiple dynamic values', () => {
        const list = (name: string, age: number) => html`<li>${name} is ${age} years old</li>`;
        const users = [
            { name: 'John',
                age: 42 }, { name: 'Jane',
                age: 43 }, { name: 'Jack',
                age: 44 },
        ];

        const string = html`<ul>${users.map((user) => list(user.name, user.age))}</ul>`;

        expect(string.getParts()).toEqual({
            0: {
                0: {
                    0: 'John',
                    1: 42,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                1: {
                    0: 'Jane',
                    1: 43,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                2: {
                    0: 'Jack',
                    1: 44,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                s: ['', '', '', ''],
            },
            s: ['<ul>', '</ul>'],
        });

        const newUsers = users.concat([
            { name: 'Jill',
                age: 45 },
        ]);
        const newString = html`<ul>${newUsers.map((user) => list(user.name, user.age))}</ul>`;

        expect(newString.getParts()).toEqual({
            0: {
                0: {
                    0: 'John',
                    1: 42,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                1: {
                    0: 'Jane',
                    1: 43,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                2: {
                    0: 'Jack',
                    1: 44,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                3: {
                    0: 'Jill',
                    1: 45,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                s: ['', '', '', '', ''],
            },
            s: ['<ul>', '</ul>'],
        });
    });

    it('should return parsed html to string', () => {
        const parsed = {
            0: {
                0: {
                    0: 'John',
                    1: 42,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                1: {
                    0: 'Jane',
                    1: 43,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                2: {
                    0: 'Jack',
                    1: 44,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                s: ['', '', '', ''],
            },
            s: ['<ul>', '</ul>'],
        };

        const htmlString = html``;

        expect(htmlString.parsedHtmlToString(parsed)).toEqual('<ul><li>John is 42 years old</li><li>Jane is 43 years old</li><li>Jack is 44 years old</li></ul>');
    });

    it('should return parsed html to string with nested dynamic values', () => {
        const parsed = {
            0: 'test',
            1: {
                0: 'test2',
                1: 'John',
                2: 'Doe',
                3: 42,
                4: '1 Park Avenue',
                s: ['<div class="', '">', ' ', ' is ', ' years old and lives at ', '</div>'],
            },
            s: ['<div class="', '">', '</div>'],
        };

        const htmlString = html``;

        expect(htmlString.parsedHtmlToString(parsed)).toEqual('<div class="test"><div class="test2">John Doe is 42 years old and lives at 1 Park Avenue</div></div>');
    });

    it('should return parsed html to string with nested dynamic values and multiple dynamic values', () => {
        const parsed = {
            0: {
                0: {
                    0: 'John',
                    1: 42,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                1: {
                    0: 'Jane',
                    1: 43,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                2: {
                    0: 'Jack',
                    1: 44,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                3: {
                    0: 'Jill',
                    1: 45,
                    s: ['<li>', ' is ', ' years old</li>'],
                },
                s: ['', '', '', '', ''],
            },
            s: ['<ul>', '</ul>'],
        };

        const htmlString = html``;

        expect(htmlString.parsedHtmlToString(parsed)).toEqual('<ul><li>John is 42 years old</li><li>Jane is 43 years old</li><li>Jack is 44 years old</li><li>Jill is 45 years old</li></ul>');
    });

    it('should parse a html string to plain string', () => {
        const className = 'test';
        const message = 'Hello World';
        const string = html`<div class="${className}">${message}</div>`;

        expect(string.toString()).toEqual('<div class="test">Hello World</div>');
    });

    it('should parse a html string with nested dynamic values to plain string', () => {
        const firsName = 'John';
        const lastName = 'Doe';
        const age = 42;
        const address = '1 Park Avenue';
        const className = 'test';
        const secondClassName = 'test2';

        const string = html`<div class="${className}">${html`<div class="${secondClassName}">${firsName} ${lastName} is ${age} years old and lives at ${address}</div>`}</div>`;

        expect(string.toString()).toEqual('<div class="test"><div class="test2">John Doe is 42 years old and lives at 1 Park Avenue</div></div>');
    });

    it('should parse a html string with nested dynamic values and multiple dynamic values to plain string', () => {
        const list = (name: string, age: number) => html`<li>${name} is ${age} years old</li>`;
        const users = [
            { name: 'John',
                age: 42 }, { name: 'Jane',
                age: 43 }, { name: 'Jack',
                age: 44 },
        ];

        const string = html`<ul>${users.map((user) => list(user.name, user.age))}</ul>`;

        expect(string.toString()).toEqual('<ul><li>John is 42 years old</li><li>Jane is 43 years old</li><li>Jack is 44 years old</li></ul>');
    });

    it('should be able to tell the difference between two html strings', () => {
        const className = 'test';
        const message = 'Hello World';
        const message2 = 'Hello World 2';

        const string = html`<div class="${className}">${message}</div>`;
        const string2 = html`<div class="${className}">${message2}</div>`;

        expect(string.differentiate(string2)).toEqual({
            1: 'Hello World 2',
        });
    });

    it('should be able to tell the difference between two html strings with nested dynamic values', () => {
        const firsName = 'John';
        const lastName = 'Doe';
        const age = 42;
        const address = '1 Park Avenue';
        const secondAddress = '2 Park Avenue';
        const className = 'test';
        const secondClassName = 'test2';

        const string = html`<div class="${className}">${html`<div class="${secondClassName}">${firsName} ${lastName} is ${age} years old and lives at ${address}</div>`}</div>`;
        const string2 = html`<div class="${className}">${html`<div class="${secondClassName}">${firsName} ${lastName} is ${age} years old and lives at ${secondAddress}</div>`}</div>`;

        expect(string.differentiate(string2)).toEqual({
            1: {
                4: '2 Park Avenue',
            },
        });
    });

    it('should be able to tell the difference between two html strings with nested dynamic values and multiple dynamic values', () => {
        const list = (name: string, age: number) => html`<li>${name} is ${age} years old</li>`;
        const users = [
            { name: 'John',
                age: 42 }, { name: 'Jane',
                age: 43 }, { name: 'Jack',
                age: 44 },
        ];

        const string = html`<ul>${users.map((user) => list(user.name, user.age))}</ul>`;
        const string2 = html`<ul>${users.map((user) => list(user.name, user.age + 1))}</ul>`;

        expect(string.differentiate(string2)).toEqual({
            0: {
                0: {
                    1: 43,
                },
                1: {
                    1: 44,
                },
                2: {
                    1: 45,
                },
            },
        });
    });

    it('should parse a html string with array dynamic values  to plain string', () => {
        const className = 'test';
        const users = [
            { name: 'John',
                age: 42 }, { name: 'Jane',
                age: 43 }, { name: 'Jack',
                age: 44 },
        ];

        const string = html`<div class="${className}">${users.map((user) => html`<li>${user.name} is ${user.age} years old</li>`)}</div>`;

        expect(string.toString()).toEqual('<div class="test"><li>John is 42 years old</li><li>Jane is 43 years old</li><li>Jack is 44 years old</li></div>');
    });

    it('should parse empty dynamic values to plain string', () => {
        const string = html`<div>${''}</div>`;

        expect(string.toString()).toEqual('<div></div>');

        expect(string.getParts()).toEqual({ 0: '',
            s: ['<div>', '</div>'] });

        // it should be able to tell the difference between two html strings with empty dynamic values
        const string2 = html`<div>${'hello'}</div>`;

        expect(string2.toString()).toEqual('<div>hello</div>');

        expect(string2.getParts()).toEqual({ 0: 'hello',
            s: ['<div>', '</div>'] });

        expect(string.differentiate(string2)).toEqual({
            0: 'hello',
        });

        expect(string2.differentiate(string)).toEqual({
            0: '',
        });
    });

    it('should detect the changes and rerender correctly', () => {
        let value: string | null = 'Hello World';
        const string = html`<div>${value || ''}</div>`;

        expect(string.toString()).toEqual('<div>Hello World</div>');

        const parts = string.getParts();

        expect(parts).toEqual({ 0: 'Hello World',
            s: ['<div>', '</div>'] });

        value = null;
        const string2 = html`<div>${value}</div>`;

        expect(string2.toString()).toEqual('<div></div>');

        expect(string2.getParts()).toEqual({ 0: null,
            s: ['<div>', '</div>'] });
        const diff = string.differentiate(string2);

        expect(diff).toEqual({
            0: null,
        });

        const merge = mergeObjects(parts, diff);

        expect(merge).toEqual({ s: ['<div>', '</div>'] });

        const htmlString = string.parsedHtmlToString(merge);

        expect(htmlString).toEqual('<div></div>');
    });

    it('should detect the changes and rerender correctly with nested dynamic values', () => {
        const className = 'test';
        const message = 'Hello World';
        const message2 = 'Hello World 2';

        const string = html`<div class="${className}">${message}</div>`;
        const string2 = html`<div class="${className}">${message2}</div>`;

        expect(string.toString()).toEqual('<div class="test">Hello World</div>');

        const parts = string.getParts();

        expect(parts).toEqual({ 1: 'Hello World',
            0: 'test',
            s: ['<div class="', '">', '</div>'] });

        const diff = string.differentiate(string2);

        expect(diff).toEqual({
            1: 'Hello World 2',
        });

        const merge = mergeObjects(parts, diff);

        expect(merge).toEqual({ 1: 'Hello World 2',
            0: 'test',
            s: ['<div class="', '">', '</div>'] });

        const htmlString = string.parsedHtmlToString(merge);

        expect(htmlString).toEqual('<div class="test">Hello World 2</div>');
    });

    it('should detect the changes and rerender correctly with nested dynamic values and multiple dynamic values', () => {
        const htmlString = html`<div class="${'test'}">${'Hello World'}</div>`;
        const htmlString2 = html``;

        expect(htmlString.toString()).toEqual('<div class="test">Hello World</div>');
        const getParts = htmlString.getParts();

        expect(getParts).toEqual({ 1: 'Hello World',
            0: 'test',
            s: ['<div class="', '">', '</div>'] });

        const diff = htmlString.differentiate(htmlString2);

        expect(diff).toEqual({
            0: null,
            1: null,
            s: {
                0: '',
                1: null,
                2: null,
            },
        });

        const merge = mergeObjects(getParts, diff);

        expect(merge).toEqual({
            s: [''],
        });
    });

    it('should be able to reconstruct the html string from the parsed html object', () => {
        const htmlString = html`<div class="${'test'}">${'Hello World'}</div>`;
        const htmlString2 = html``;

        expect(htmlString.toString()).toEqual('<div class="test">Hello World</div>');
        const getParts = htmlString.getParts();

        expect(getParts).toEqual({ 1: 'Hello World',
            0: 'test',
            s: ['<div class="', '">', '</div>'] });

        const diff = htmlString.differentiate(htmlString2);

        expect(diff).toEqual({
            0: null,
            1: null,
            s: {
                0: '',
                1: null,
                2: null,
            },
        });

        const merge = htmlString.patch(diff);

        expect(merge).toEqual(html``);
    });
});

describe('CssGenerator', () => {
    it('should generate a record object from a string', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; }`;

        expect(data.classes).toEqual({ body: 'body-hello' });
    });

    it('should generate a record object from a string with multiple classes', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; } .test { color: blue; }`;

        expect(data.classes).toEqual({ body: 'body-hello',
            test: 'test-hello' });
    });

    it('should generate a record object from a string with multiple classes and multiple rules', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; } .test { color: blue; } .test2 { color: green; }`;

        expect(data.classes).toEqual({ body: 'body-hello',
            test: 'test-hello',
            test2: 'test2-hello' });
    });

    it('should generate a style element from a string', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; }`;

        expect(data.string.toString()).toEqual('<style>.body-hello { color: red; }</style>');
    });

    it('should generate a style element from a string with multiple classes', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; } .test { color: blue; }`;

        expect(data.string.toString()).toEqual('<style>.body-hello { color: red; } .test-hello { color: blue; }</style>');
    });

    it('should generate a style element from a string with multiple classes and multiple rules', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; } .test { color: blue; } .testDo { color: green; }`;

        expect(data.string.toString()).toEqual('<style>.body-hello { color: red; } .test-hello { color: blue; } .testDo-hello { color: green; }</style>');
    });

    it('should ignore non class styles', () => {
        const css = makeStyles('hello');

        const data = css`body { color: red; }`;

        expect(data.string.toString()).toEqual('<style>body { color: red; }</style>');

        const idData = css`#body { color: red; }`;

        expect(idData.string.toString()).toEqual('<style>#body { color: red; }</style>');

        const elementData = css`div { color: red; }`;

        expect(elementData.string.toString()).toEqual('<style>div { color: red; }</style>');
    });

    it('should generate a style element from a string with multiple classes and multiple rules and multiple media queries', () => {
        const css = makeStyles('hello');

        const data = css`.body { color: red; } .test { color: blue; } .testDo { color: green; } @media (max-width: 600px) { .body { color: red; } .test { color: blue; } .testDo { color: green; } }`;

        expect(data.string.toString()).toEqual('<style>.body-hello { color: red; } .test-hello { color: blue; } .testDo-hello { color: green; } @media (max-width: 600px) { .body-hello { color: red; } .test-hello { color: blue; } .testDo-hello { color: green; } }</style>');
    });

    it('should correctly handle pseudo classes', () => {
        const css = makeStyles('hello');

        const data = css`.body:hover { color: red; } .test:active { color: blue; } .testDo:focus { color: green; }`;

        expect(data.string.toString()).toEqual('<style>.body-hello:hover { color: red; } .test-hello:active { color: blue; } .testDo-hello:focus { color: green; }</style>');
    });

    it('should correctly handle pseudo elements', () => {
        const css = makeStyles('hello');

        const data = css`.body::before { color: red; } .test::after { color: blue; } .testDo::selection { color: green; }`;

        expect(data.string.toString()).toEqual('<style>.body-hello::before { color: red; } .test-hello::after { color: blue; } .testDo-hello::selection { color: green; }</style>');
    });

    it('should handle sibling selectors', () => {
        const css = makeStyles('hello');

        const data = css`.body + .test { color: red; } .test ~ .testDo { color: blue; }`;

        expect(data.string.toString()).toEqual('<style>.body-hello + .test-hello { color: red; } .test-hello ~ .testDo-hello { color: blue; }</style>');
    });
});
