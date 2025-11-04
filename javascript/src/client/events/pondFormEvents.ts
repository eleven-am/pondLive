import { HandlerFunction } from './handler';
import { ChannelEventHandler } from '../actors/channelEvent';
import { addFilesListener } from '../actors/uploader';

const pondFocus = (handler: HandlerFunction) => {
    handler('[pond-focus]', 'focus', (_, element) => ({
        value: null,
        dataId: element.getAttribute('pond-data-id'),
    }));
};

const pondBlur = (handler: HandlerFunction) => {
    handler('[pond-blur]', 'blur', (_, element) => ({
        value: null,
        dataId: element.getAttribute('pond-data-id'),
    }));
};

const pondChange = (handler: HandlerFunction) => {
    handler('[pond-change]', 'change', (event, element) => {
        const input = event.target as HTMLInputElement;
        const dataId = element.getAttribute('pond-data-id');

        return {
            value: input.value,
            dataId,
        };
    });
};

const pondInput = (handler: HandlerFunction) => {
    handler('[pond-input]', 'input', (_, element) => ({
        value: (element as HTMLInputElement).value,
        dataId: element.getAttribute('pond-data-id'),
    }));
};

const pondCheck = (handler: HandlerFunction) => {
    handler('[pond-check]', 'change', (_, element) => {
        const dataId = element.getAttribute('pond-data-id');

        return {
            value: (element as HTMLInputElement).checked ? 'on' : 'off',
            dataId,
        };
    });
};

const pondSubmit = (handler: HandlerFunction) => {
    handler('[pond-submit]', 'submit', (evt, element) => {
        evt.preventDefault();
        const elements: NodeListOf<HTMLInputElement> = element.querySelectorAll('input, select, textarea');
        const formData: Record<string, string> = {};

        elements.forEach((input, i) => {
            formData[input.getAttribute('name') || `${i}`] = input.value;
        });

        return {
            value: null,
            formData,
            dataId: element.getAttribute('pond-data-id'),
        };
    });
};

const pondFile = (handler: HandlerFunction, userId: string, channelHandler: ChannelEventHandler) => {
    handler('[pond-file]', 'change', async (_, element) => {
        const input = element as HTMLInputElement;
        const files = input.files;

        if (files) {
            const metaData = await addFilesListener(files, element, userId, channelHandler);

            return {
                value: null,
                files: metaData,
            };
        }

        return null;
    });
};

export const pondFormInit = (handler: HandlerFunction, channelHandler: ChannelEventHandler, userId: string) => {
    pondFocus(handler);
    pondBlur(handler);
    pondChange(handler);
    pondInput(handler);
    pondSubmit(handler);
    pondCheck(handler);
    pondFile(handler, userId, channelHandler);
};
