import { describeFailures, runFile, type Failure, type FixtureFile } from '../contract/harness';

declare const __FIXTURES__: FixtureFile[];

function run(): void {
    const failures: Failure[] = [];
    let cases = 0;
    let steps = 0;
    let edits = 0;
    for (const file of __FIXTURES__) {
        const result = runFile(file);
        failures.push(...result.failures);
        cases += result.cases;
        steps += result.steps;
        edits += result.edits;
    }
    const verdict = failures.length === 0 ? 'PASS' : 'FAIL';
    const summary = `${verdict} ${__FIXTURES__.length} files, ${cases} cases, ${steps} steps, ${edits} edits, ${failures.length} failures, ${navigator.userAgent}`;
    const report = failures.length === 0 ? summary : `${summary}\n\n${describeFailures(failures, 20)}`;
    const next = new DOMParser().parseFromString('<!doctype html><html><head><title></title></head><body><pre id="result"></pre></body></html>', 'text/html');
    next.title = `${verdict} ${cases - failures.length}/${cases}`;
    next.getElementById('result')!.textContent = report;
    document.replaceChild(document.importNode(next.documentElement, true), document.documentElement);
    (window as unknown as { __contractResult: string }).__contractResult = report;
}

run();
