import { HandlerFunction } from './handler';
import { ChannelEventHandler } from '../actors/channelEvent';
import { addFilesListener } from '../actors/uploader';
import { DragData, UploadList } from '../types';

const pondClick = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-click]', 'click', () => ({
        value: null,
    }));
};

const pondMouseEnter = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-mouseenter]', 'mouseenter', () => ({
        value: null,
    }));
};

const pondMouseLeave = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-mouseleave]', 'mouseleave', () => ({
        value: null,
    }));
};

const pondMouseMove = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-mousemove]', 'mousemove', () => ({
        value: null,
    }));
};

const pondMouseDown = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-mousedown]', 'mousedown', () => ({
        value: null,
    }));
};

const pondMouseUp = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-mouseup]', 'mouseup', () => ({
        value: null,
    }));
};

const pondDoubleClick = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-double-click]', 'dblclick', () => ({
        value: null,
    }));
};

const pondContextMenu = (handler: HandlerFunction<MouseEvent>) => {
    handler('[pond-context-menu]', 'contextmenu', () => ({
        value: null,
    }));
};

const pondDragStart = (handler: HandlerFunction<DragEvent>) => {
    handler('[pond-drag-start]', 'dragstart', (_, element) => {
        const positions = element.getBoundingClientRect();
        const dragData: DragData = {
            top: positions.top,
            left: positions.left,
            width: positions.width,
            height: positions.height,
        };

        return {
            value: null,
            dragData,
        };
    });
};

const pondDragEnd = (handler: HandlerFunction<DragEvent>) => {
    handler('[pond-drag-end]', 'dragend', (_, element) => {
        const positions = element.getBoundingClientRect();
        const dragData: DragData = {
            top: positions.top,
            left: positions.left,
            width: positions.width,
            height: positions.height,
        };

        return {
            value: null,
            dragData,
        };
    });
};

const pondDragOver = (handler: HandlerFunction<DragEvent>) => {
    handler('[pond-drag-over]', 'dragover', (_, element) => {
        const positions = element.getBoundingClientRect();
        const dragData: DragData = {
            top: positions.top,
            left: positions.left,
            width: positions.width,
            height: positions.height,
        };

        return {
            value: null,
            dragData,
        };
    });
};

const pondDragEnter = (handler: HandlerFunction<DragEvent>) => {
    handler('[pond-drag-enter]', 'dragenter', (_, element) => {
        const positions = element.getBoundingClientRect();
        const dragData: DragData = {
            top: positions.top,
            left: positions.left,
            width: positions.width,
            height: positions.height,
        };

        return {
            value: null,
            dragData,
        };
    });
};

const pondDragLeave = (handler: HandlerFunction<DragEvent>) => {
    handler('[pond-drag-leave]', 'dragleave', (_, element) => {
        const positions = element.getBoundingClientRect();
        const dragData: DragData = {
            top: positions.top,
            left: positions.left,
            width: positions.width,
            height: positions.height,
        };

        return {
            value: null,
            dragData,
        };
    });
};

const pondDrop = (handler: HandlerFunction<InputEvent>, channelHandler: ChannelEventHandler, userId: string) => {
    handler('[pond-drop]', 'drop', async (event, element) => {
        event.preventDefault();
        const files = event.dataTransfer?.items;
        const positions = element.getBoundingClientRect();
        const dragData: DragData = {
            top: positions.top,
            left: positions.left,
            width: positions.width,
            height: positions.height,
        };

        let metaData: UploadList | undefined;

        if (files) {
            metaData = await addFilesListener(files, element, userId, channelHandler);
        }

        return {
            value: null,
            files: metaData,
            dragData,
        };
    });
};

const pondUpload = (handler: HandlerFunction<InputEvent>, channelHandler: ChannelEventHandler, userId: string) => {
    handler('[pond-upload]', 'submit', async (event, form) => {
        event.preventDefault();
        const file = form.querySelector('input[type="file"]') as HTMLInputElement;
        const files = file.files;

        let metaData: UploadList | undefined;

        if (files) {
            metaData = await addFilesListener(files, form, userId, channelHandler);
        }

        return {
            value: null,
            files: metaData,
        };
    });
};

const pondPaste = (handler: HandlerFunction<ClipboardEvent>, channelHandler: ChannelEventHandler, userId: string) => {
    handler('[pond-paste]', 'paste', async (event, element) => {
        event.preventDefault();
        const files = event.clipboardData?.items;
        const value = event.clipboardData?.getData('text/plain') ?? null;
        let metaData: UploadList | undefined;

        if (files) {
            metaData = await addFilesListener(files, element, userId, channelHandler);
        }

        return {
            value,
            files: metaData,
        };
    });
};

export const pondMouseEvents = (handler: HandlerFunction<any>, channelHandler: ChannelEventHandler, userId: string) => {
    pondClick(handler);
    pondMouseEnter(handler);
    pondMouseLeave(handler);
    pondMouseMove(handler);
    pondMouseDown(handler);
    pondMouseUp(handler);
    pondDoubleClick(handler);
    pondDragStart(handler);
    pondDragEnd(handler);
    pondDragOver(handler);
    pondDragEnter(handler);
    pondDragLeave(handler);
    pondContextMenu(handler);
    pondDrop(handler, channelHandler, userId);
    pondUpload(handler, channelHandler, userId);
    pondPaste(handler, channelHandler, userId);
};
