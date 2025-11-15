module.exports = {
    transform: {
        '^.+\\.ts?$': 'ts-jest',
    },
    moduleFileExtensions: ['ts', 'js', 'json', 'node', 'd.ts'],
    collectCoverage: true,
    collectCoverageFrom: [
        "src-old/**/*.{js,jsx,ts,tsx}",
        "!src-old/**/*.d.ts"
    ],
};
