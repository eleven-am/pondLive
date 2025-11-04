import { differ, getChanges, mergeObjects } from './differ';

describe('differ', () => {
    it('should return unchanged if both objects are the same', () => {
        const obj1 = {
            a: 1,
            b: 2,
            c: 3,
        };
        const obj2 = {
            a: 1,
            b: 2,
            c: 3,
        };
        const result = differ(obj1, obj2);

        expect(result).toEqual({
            a: {
                type: 'unchanged',
                data: 1,
            },
            b: {
                type: 'unchanged',
                data: 2,
            },
            c: {
                type: 'unchanged',
                data: 3,
            },
        });
    });

    it('should return created if the first object is undefined', () => {
        const obj1 = undefined;
        const obj2 = {
            a: 1,
            b: 2,
            c: 3,
        };
        const result = differ(obj1, obj2);

        expect(result).toEqual({
            type: 'created',
            data: {
                a: 1,
                b: 2,
                c: 3,
            },
        });
    });

    it('should return deleted if the second object is undefined', () => {
        const obj1 = {
            a: 1,
            b: 2,
            c: 3,
        };
        const result = differ(obj1, {});

        expect(result).toEqual({
            a: {
                type: 'deleted',
                data: 1,
            },
            b: {
                type: 'deleted',
                data: 2,
            },
            c: {
                type: 'deleted',
                data: 3,
            },
        });
    });

    it('should be able to handle nested objects', () => {
        const obj1 = {
            a: 1,
            b: {
                c: 2,
                d: 3,
            },
        };
        const obj2 = {
            a: 1,
            b: {
                c: 2,
                d: 4,
            },
        };
        const result = differ(obj1, obj2);

        expect(result).toEqual({
            a: {
                type: 'unchanged',
                data: 1,
            },
            b: {
                c: {
                    type: 'unchanged',
                    data: 2,
                },
                d: {
                    type: 'updated',
                    data: 4,
                },
            },
        });
    });

    it('should throw an error if 2 functions are passed', () => {
        const obj1 = () => {
        };
        const obj2 = () => {
        };

        expect(() => differ(obj1, obj2)).toThrow('Invalid argument. Function given, object expected.');
    });

    it('should throw an error if 1 function and 1 object are passed', () => {
        const obj1 = () => {
        };
        const obj2 = {
            a: 1,
        };

        expect(() => differ(obj1, obj2)).toThrow('Invalid argument. Function given, object expected.');
    });

    it('should throw an error if 1 object and 1 function are passed', () => {
        const obj1 = {
            a: 1,
        };
        const obj2 = () => {
        };

        expect(() => differ(obj1, obj2)).toThrow('Invalid argument. Function given, object expected.');
    });

    it('should ignore a nested function', () => {
        const obj1 = {
            a: 1,
            b: {
                c: 2,
                d: () => {
                    // since it ignores functions, this would be treated as undefined
                },
            },
        };
        const obj2 = {
            a: 1,
            b: {
                c: 2,
                d: 4,
            },
        };
        const result = differ(obj1, obj2);

        expect(result).toEqual({
            a: {
                type: 'unchanged',
                data: 1,
            },
            b: {
                c: {
                    type: 'unchanged',
                    data: 2,
                },
                d: {
                    type: 'created',
                    data: 4,
                },
            },
        });
    });

    it('should compare dates', () => {
        const obj1 = {
            a: 1,
            b: new Date('2019-01-01'),
        };
        const obj2 = {
            a: 1,
            b: new Date('2019-01-02'),
        };
        const result = differ(obj1, obj2);

        expect(result).toEqual({
            a: {
                type: 'unchanged',
                data: 1,
            },
            b: {
                type: 'updated',
                data: new Date('2019-01-02'),
            },
        });

        // equal dates
        const obj3 = {
            a: 1,
            b: new Date('2019-01-01'),
        };

        const result2 = differ(obj1, obj3);

        expect(result2).toEqual({
            a: {
                type: 'unchanged',
                data: 1,
            },
            b: {
                type: 'unchanged',
                data: new Date('2019-01-01'),
            },
        });
    });

    it('should compare arrays', () => {
        const obj1 = {
            a: 1,
            b: [1, 2, 3],
        };
        const obj2 = {
            a: 1,
            b: [1, 2, 4],
        };
        const result = differ(obj1, obj2);

        expect(result).toEqual({
            a: {
                type: 'unchanged',
                data: 1,
            },
            b: {
                0: {
                    type: 'unchanged',
                    data: 1,
                },
                1: {
                    type: 'unchanged',
                    data: 2,
                },
                2: {
                    type: 'updated',
                    data: 4,
                },
            },
        });
    });
});

describe('getChanges', () => {
    it('should be able to get only changed properties1', () => {
        const obj = {
            a: {
                type: 'unchanged',
                data: 1,
            },
        };

        const result = getChanges(obj);

        expect(result).toEqual({});
    });

    it('should be able to get only changed properties2', () => {
        const obj = {
            0: {
                a: {
                    type: 'updated',
                    data: 2,
                },
            },
            1: {
                type: 'created',
                data: 3,
            },
        };

        const result = getChanges(obj);

        expect(result).toEqual({ 0: { a: 2 },
            1: 3 });
    });

    it('should be able to get only changed properties3', () => {
        const obj = {
            3: {
                a: {
                    type: 'created',
                    data: 2,
                },
                b: {
                    type: 'deleted',
                    data: 3,
                },
                c: {
                    type: 'unchanged',
                    data: 4,
                },
                d: {
                    type: 'updated',
                    data: 5,
                },
            },
        };

        const result = getChanges(obj);

        expect(result).toEqual({ 3: { a: 2,
            b: null,
            d: 5 } });
    });

    it('should be able to get only changed properties4', () => {
        const obj = {
            0: { type: 'unchanged',
                data: 'test' },
            1: { type: 'updated',
                data: 'Hello World 2' },
            s: {
                0: { type: 'unchanged',
                    data: '<div class="' },
                1: { type: 'unchanged',
                    data: '">' },
                2: { type: 'unchanged',
                    data: '</div>' },
            },
        };

        const result = getChanges(obj);

        expect(result).toEqual({ 1: 'Hello World 2' });
    });
});

describe('mergeObjects', () => {
    it('should be able to merge two objects1', () => {
        const obj1 = {
            a: 1,
            b: 2,
            c: 3,
        };
        const obj2 = {
            a: 2,
            d: 4,
        };
        const result = mergeObjects(obj1, obj2);

        expect(result).toEqual({
            a: 2,
            b: 2,
            c: 3,
            d: 4,
        });
    });

    it('should be able to merge two objects2', () => {
        const obj1 = {
            a: 1,
            b: {
                c: 2,
                d: 3,
            },
        };
        const obj2 = {
            a: 2,
            b: {
                c: 4,
                e: 5,
            },
        };
        const result = mergeObjects(obj1, obj2);

        expect(result).toEqual({
            a: 2,
            b: {
                c: 4,
                d: 3,
                e: 5,
            },
        });
    });

    it('should be able to merge two objects3', () => {
        const obj1 = undefined;
        const obj2 = {
            a: 2,
            b: {
                c: 4,
            },
        };

        const result = mergeObjects(obj1, obj2);

        expect(result).toEqual(obj2);

        const result2 = mergeObjects(obj2, obj1);

        expect(result2).toEqual(obj2);
    });

    it('should remove null values from original object1', () => {
        const obj1 = {
            a: 1,
            b: 2,
            c: 3,
        };
        const obj2 = {
            a: 2,
            b: null,
        };
        const result = mergeObjects(obj1, obj2);

        expect(result).toEqual({
            a: 2,
            c: 3,
        });
    });

    it('should remove null values from original object2', () => {
        const obj1 = {
            a: 1,
            b: {
                c: 2,
                d: 3,
            },
        };
        const obj2 = {
            a: 2,
            b: {
                c: null,
                e: 5,
            },
        };
        const result = mergeObjects(obj1, obj2);

        expect(result).toEqual({
            a: 2,
            b: {
                d: 3,
                e: 5,
            },
        });
    });
});
