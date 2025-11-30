module.exports = {
    transform: {
        '^.+\\.ts?$': 'ts-jest',
    },
    moduleFileExtensions: ['ts', 'js', 'json', 'node', 'd.ts'],
    collectCoverage: true,
    collectCoverageFrom: [
        "src-old-old/**/*.{js,jsx,ts,tsx}",
        "!src-old-old/**/*.d.ts"
    ],
};
