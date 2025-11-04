const isFunction = (x: any): x is Function => typeof x === 'function';
const isArray = (x: any): x is any[] => Array.isArray(x);
const isDate = (x: any): x is Date => x instanceof Date;
const isObject = (x: any): x is Record<string, any> => typeof x === 'object' && x !== null && !isArray(x) && !isDate(x);
const isValue = (x: any): boolean => !isObject(x) && !isArray(x);
const isNotEmpty = (x: any): boolean => x !== undefined && x !== null;
const isEmpty = (x: any): boolean => Object.keys(x).length === 0;

export const differ = (obj1: Record<string, any> | undefined, obj2: Record<string, any>): Record<string, any> => {
    const compareValues = (value1: any, value2: any): string => {
        if (value1 === value2) {
            return 'unchanged';
        }
        if (isDate(value1) && isDate(value2) && value1.getTime() === value2.getTime()) {
            return 'unchanged';
        }
        if (value1 === undefined) {
            return 'created';
        }
        if (value2 === undefined) {
            return 'deleted';
        }

        return 'updated';
    };

    const map = (obj1: Record<string, any> | undefined, obj2: Record<string, any>): Record<string, any> => {
        if (isFunction(obj1) || isFunction(obj2)) {
            throw new Error('Invalid argument. Function given, object expected.');
        }

        if (isValue(obj1) || isValue(obj2)) {
            const newType = compareValues(obj1, obj2);

            if (newType === 'updated') {
                return {
                    type: newType,
                    data: obj2,
                };
            }

            return {
                type: newType,
                data: obj1 === undefined ? obj2 : obj1,
            };
        }

        const diff: Record<string, any> = {};

        for (const key in obj1) {
            if (isFunction(obj1[key])) {
                continue;
            }

            let value2;

            if (obj2[key] !== undefined) {
                value2 = obj2[key];
            }

            diff[key] = map(obj1[key], value2);
        }

        for (const key in obj2) {
            if (isFunction(obj2[key]) || diff[key] !== undefined) {
                continue;
            }

            diff[key] = map(undefined, obj2[key]);
        }

        return diff;
    };

    return map(obj1, obj2);
};

export const getChanges = (diffedObject: any): any => {
    const changed: any = {};

    for (const key in diffedObject) {
        if (diffedObject[key]) {
            if (diffedObject[key].type === 'created' || diffedObject[key].type === 'updated') {
                changed[key] = diffedObject[key].data;
            } else if (diffedObject[key].type === 'deleted') {
                changed[key] = null;
            } else if (typeof diffedObject[key] === 'object') {
                const data = getChanges(diffedObject[key]);

                if (isNotEmpty(data) && !isEmpty(data)) {
                    changed[key] = data;
                }
            }
        }
    }

    return changed;
};

function mergeArray (obj1: any[], obj2: any) {
    const mapped: any = obj1.map((item: any, index: number) => {
        if (isNotEmpty(obj2[index])) {
            // eslint-disable-next-line @typescript-eslint/no-use-before-define
            return mergeObjects(item, obj2[index]);
        } else if (obj2[index] === null) {
            return undefined;
        }

        return item;
    });

    for (const key in obj2) {
        if (mapped[key] === undefined && isNotEmpty(obj2[key])) {
            mapped[key] = obj2[key];
        }
    }

    return mapped.filter((item: any) => isNotEmpty(item));
}

function mergeTwoObjects (obj1: any, obj2: any) {
    const merged: any = {};

    for (const key in obj1) {
        if (obj2[key] !== undefined && obj2[key] !== null) {
            if (!isValue(obj2[key])) {
                // eslint-disable-next-line @typescript-eslint/no-use-before-define
                const value = mergeObjects(obj1[key], obj2[key]);

                if (isNotEmpty(value)) {
                    merged[key] = value;
                }
            } else {
                merged[key] = obj2[key];
            }
        } else {
            merged[key] = obj1[key];
        }
    }

    for (const key in obj2) {
        if (merged[key] === undefined && isNotEmpty(obj2[key])) {
            merged[key] = obj2[key];
        }
    }

    return merged;
}

export const mergeObjects = (obj1: any, obj2: any): any => {
    if (obj1 === undefined) {
        return obj2;
    }

    if (obj2 === undefined) {
        return obj1;
    }

    if (obj2 === null) {
        return undefined;
    }

    if (isValue(obj1) && isValue(obj2)) {
        return obj2;
    }

    if (isArray(obj1)) {
        return mergeArray(obj1, obj2);
    }

    return mergeTwoObjects(obj1, obj2);
};
