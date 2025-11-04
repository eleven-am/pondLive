import fs from 'fs';

type KeyOf<T> = keyof T;
type SortingOrder = 'asc' | 'desc';

export function sortBy<DataType> (array: DataType[], keys: KeyOf<DataType> | KeyOf<DataType>[], order: SortingOrder | SortingOrder[]): DataType[] {
    const sortFields = Array.isArray(keys) ? keys : [keys];
    const ordersArray = Array.isArray(order) ? order : [order];

    return array.sort((a, b) => {
        let i = 0;

        while (i < sortFields.length) {
            if (ordersArray[i] === 'asc') {
                if (a[sortFields[i]] < b[sortFields[i]]) {
                    return -1;
                }
                if (a[sortFields[i]] > b[sortFields[i]]) {
                    return 1;
                }
            } else {
                if (a[sortFields[i]] < b[sortFields[i]]) {
                    return 1;
                }
                if (a[sortFields[i]] > b[sortFields[i]]) {
                    return -1;
                }
            }
            i++;
        }

        return 0;
    });
}

export function uuidV4 () {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
        // eslint-disable-next-line no-bitwise
        const r = Math.random() * 16 | 0;
        // eslint-disable-next-line no-bitwise
        const v = c === 'x' ? r : (r & 0x3 | 0x8);

        return v.toString(16);
    });
}

export function deepCompare (firstObject: any, secondObject: any): boolean {
    if (firstObject === secondObject) {
        return true;
    }

    if (typeof firstObject === 'function' && typeof secondObject === 'function') {
        return firstObject.toString() === secondObject.toString();
    }

    if (firstObject instanceof Date && secondObject instanceof Date) {
        return firstObject.getTime() === secondObject.getTime();
    }

    if (Array.isArray(firstObject) && Array.isArray(secondObject)) {
        if (firstObject.length !== secondObject.length) {
            return false;
        }

        return firstObject.every((item, index) => deepCompare(item, secondObject[index]));
    }

    if (firstObject && secondObject && typeof firstObject === 'object' && typeof secondObject === 'object') {
        if (firstObject.constructor !== secondObject.constructor) {
            return false;
        }
        const properties = Object.keys(firstObject);

        if (properties.length !== Object.keys(secondObject).length) {
            return false;
        }

        return properties.every((prop) => deepCompare(firstObject[prop], secondObject[prop]));
    }

    return false;
}

export function isEmpty (obj: any): boolean {
    return Object.keys(obj).length === 0;
}

export function fileExists (filePath: string) {
    return new Promise<boolean>((resolve) => {
        fs.stat(filePath, (err, stats) => {
            if (err) {
                return resolve(false);
            }

            resolve(stats.isFile());
        });
    });
}
