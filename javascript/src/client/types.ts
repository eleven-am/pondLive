export interface LiveEvent<Events extends string = string> {
    action: Events;
    value: string | null;
    dataId: string | null;
    files?: UploadList;
    dragData?: DragData;
    formData?: Record<string, string>;
    address: string;
}

export interface FileMetaData {
    name: string;
    size: number;
    mimeType: string;
    path: string;
    lastModified: number;
    lastModifiedDate: Date;
}

export interface PondFile extends FileMetaData {
    identifier: string;
}

export interface UploadList {
    identifier: string;
    files: PondFile[];
}

export interface DragData {
    top: number;
    left: number;
    width: number;
    height: number;
}
