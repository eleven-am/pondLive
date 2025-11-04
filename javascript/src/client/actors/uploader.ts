import { ChannelEventHandler } from './channelEvent';
import { emitEvent } from '../events/eventEmmiter';
import { PondLiveHeaders } from '../routing/router';
import { FileMetaData, PondFile, UploadList } from '../types';

interface PathFile {
    path: string;
    file: File;
}

interface UploadFile extends FileMetaData {
    file: File;
}

interface Upload extends PondFile {
    file: File;
    unsubscribe: () => void;
}

export interface PondUploadResponse {
    accepted: boolean;
    message?: string;
    route?: string;
}

const initUploadFile = (uploadPath: string, userId: string, element: HTMLElement) => {
    const xhr = new XMLHttpRequest();

    xhr.open('POST', uploadPath);
    xhr.setRequestHeader(PondLiveHeaders.LIVE_ROUTER, 'true');
    xhr.setRequestHeader(PondLiveHeaders.LIVE_USER_ID, userId);

    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            const response = JSON.parse(xhr.responseText);

            emitEvent('pond-upload-success', response, element);
        } else {
            emitEvent('pond-upload-error', xhr.responseText, element);
        }
    };

    xhr.onloadstart = () => emitEvent('pond-upload-start', {});
    xhr.onloadend = () => emitEvent('pond-upload-end', {});

    xhr.onprogress = (e) => {
        if (e.lengthComputable) {
            emitEvent('pond-upload-progress', {
                progress: Math.round((e.loaded / e.total) * 100),
            }, element);
        }
    };

    return xhr;
};

const getFilesFromTree = (item: FileSystemEntry, path = '/'): Promise<PathFile[]> => new Promise<PathFile[]>((resolve) => {
    if (item.isFile) {
        const file = item as FileSystemFileEntry;

        file.file((file) => {
            const filePath = `${path}${file.name}`;

            resolve([
                { path: filePath,
                    file },
            ]);
        });
    } else if (item.isDirectory) {
        const directory = item as FileSystemDirectoryEntry;
        const reader = directory.createReader();
        const files: PathFile[] = [];

        reader.readEntries(async (entries) => {
            for (let i = 0; i < entries.length; i++) {
                const entry = entries[i];
                const filePath = `${path}${directory.name}/`;
                const file = await getFilesFromTree(entry, filePath);

                files.push(...file);
            }

            resolve(files);
        });
    }
});

const pathFilesFromFileList = (fileList: FileList | null) => {
    const files: PathFile[] = [];

    if (!fileList) {
        return files;
    }

    for (let i = 0; i < fileList.length; i++) {
        files.push({ path: `/${fileList[i].name}`,
            file: fileList[i] });
    }

    return files;
};

const pathFilesFromDataTransferItemList = (items: DataTransferItemList): Promise<PathFile[]> => new Promise<PathFile[]>(async (resolve) => {
    const rawFiles: FileSystemEntry[] = [];

    for (let i = 0; i < items.length; i++) {
        const item = items[i].webkitGetAsEntry();

        if (item) {
            rawFiles.push(item);
        }
    }

    const promises = rawFiles.map((file) => getFilesFromTree(file));
    const files = await Promise.all(promises);

    resolve(files.flat());
});

const mapPathFilesToUFileMetaData = (files: PathFile[]): UploadFile[] => files.map((file) => ({
    name: file.path,
    size: file.file.size,
    mimeType: file.file.type,
    path: file.path,
    file: file.file,
    lastModified: file.file.lastModified,
    lastModifiedDate: new Date(file.file.lastModified),
}));

async function getFiles (files: FileList | DataTransferItemList | null): Promise<UploadFile[]> {
    if (files instanceof FileList) {
        return mapPathFilesToUFileMetaData(pathFilesFromFileList(files));
    } else if (files instanceof DataTransferItemList) {
        const data = await pathFilesFromDataTransferItemList(files);

        return mapPathFilesToUFileMetaData(data);
    }

    return [];
}

function addFileListener (file: UploadFile, element: HTMLElement, userId: string, handler: ChannelEventHandler): Upload {
    const identifier = `${file.name}-${Math.random().toString(36)
        .substring(2, 15)}`;

    const subscription = handler<PondUploadResponse>(identifier, (event) => {
        if (!event.accepted) {
            emitEvent('pond-upload-error', event.message);
        }

        if (!event.route) {
            emitEvent('pond-upload-error', 'No route provided');

            return;
        }

        const xhr = initUploadFile(event.route, userId, element);
        const formData = new FormData();

        formData.append(file.path, file.file, file.name);
        xhr.send(formData);
    });

    return {
        ...file,
        file: file.file,
        unsubscribe: subscription,
        identifier,
    };
}

export async function addFilesListener (files: FileList | DataTransferItemList | null, element: HTMLElement, userId: string, handler: ChannelEventHandler): Promise<UploadList> {
    const pondFiles = await getFiles(files);

    const subscriptions = pondFiles.map((file) => addFileListener(file, element, userId, handler));

    const identifier = `${Math.random().toString(36)
        .substring(2, 15)}`;

    handler<PondUploadResponse>(identifier, (event) => {
        if (!event.accepted) {
            emitEvent('pond-upload-error', event.message);
        }

        if (!event.route) {
            emitEvent('pond-upload-error', 'No route provided');

            return;
        }

        subscriptions.forEach((file) => file.unsubscribe());
        const xhr = initUploadFile(event.route, userId, element);
        const formData = new FormData();

        pondFiles.forEach((file) => formData.append(file.path, file.file, file.name));
        xhr.send(formData);
    });

    const newFiles = subscriptions.map((file) => ({
        name: file.path,
        size: file.file.size,
        mimeType: file.file.type,
        path: file.path,
        lastModified: file.file.lastModified,
        identifier: file.identifier,
        lastModifiedDate: new Date(file.file.lastModified),
    }));

    return {
        files: newFiles,
        identifier,
    };
}

