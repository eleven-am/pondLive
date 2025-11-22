function upload(element, transport) {
    let activeXhr = null;
    let uploadConfig = null;
    let pending = null;

    const handleChange = (event) => {
        const input = event.target;
        const files = input.files;

        if (!files || files.length === 0) {
            return;
        }

        const fileArray = Array.from(files);
        pending = { files: fileArray, input };

        const first = fileArray[0];
        transport.send('ready', {
            name: first.name,
            size: first.size,
            type: first.type
        });

        if (uploadConfig) {
            processPending(uploadConfig);
        }
    };

    const startUpload = (files, input) => {
        if (activeXhr) {
            activeXhr.abort();
            activeXhr = null;
        }

        const target = uploadConfig?.url;
        if (!target) {
            transport.send('error', { error: 'Upload target not configured' });
            return;
        }
        const xhr = new XMLHttpRequest();

        if (uploadConfig.token) {
            xhr.setRequestHeader('X-Upload-Token', uploadConfig.token);
        }

        xhr.upload.onprogress = (event) => {
            const loaded = event.loaded;
            const total = event.lengthComputable ? event.total : files[0].size;
            transport.send('progress', {
                loaded,
                total,
                name: files[0].name,
                size: files[0].size,
                index: uploadState.index,
                count: uploadState.count
            });
        };

        xhr.onerror = () => {
            activeXhr = null;
            transport.send('error', { error: 'Upload failed' });
            advanceQueue();
        };

        xhr.onabort = () => {
            activeXhr = null;
            transport.send('cancelled', {});
            uploadState = resetState();
        };

        xhr.onload = () => {
            activeXhr = null;
            if (xhr.status < 200 || xhr.status >= 300) {
                transport.send('error', { error: `Upload failed (${xhr.status})` });
            } else {
                const totalSize = files[0].size;
                transport.send('progress', {
                    loaded: totalSize,
                    total: totalSize,
                    name: files[0].name,
                    size: files[0].size,
                    index: uploadState.index,
                    count: uploadState.count
                });
                if (input) {
                    input.value = '';
                }
            }
            advanceQueue();
        };

        const form = new FormData();

        form.append('file', files[0]);
        xhr.open('POST', target, true);
        xhr.send(form);

        activeXhr = xhr;
    };

    const cancelUpload = () => {
        if (activeXhr) {
            activeXhr.abort();
            activeXhr = null;
        }
    };

    const processPending = (config) => {
        if (!pending) {
            return;
        }

        const { files, input } = pending;
        const totalSize = files.reduce((acc, f) => acc + f.size, 0);

        if (config.maxSize && config.maxSize > 0 && totalSize > config.maxSize) {
            transport.send('error', {
                error: `File exceeds maximum size (${config.maxSize} bytes)`
            });
            input.value = '';
            pending = null;
            return;
        }

        if (config.accept && config.accept.length > 0) {
            const accepts = (file) => {
                const fileType = file.type;
                const fileName = file.name;
                return config.accept.some(pattern => {
                    if (pattern.startsWith('.')) {
                        return fileName.endsWith(pattern);
                    }
                    if (pattern.includes('*')) {
                        const regex = new RegExp('^' + pattern.replace(/\*/g, '.*') + '$');
                        return regex.test(fileType);
                    }
                    return fileType === pattern;
                });
            };

            if (!files.every(accepts)) {
                transport.send('error', {
                    error: `File type not accepted. Allowed: ${config.accept.join(', ')}`
                });
                input.value = '';
                pending = null;
                return;
            }
        }

        files.forEach((f) => uploadState.queue.push({ file: f, input }));
        uploadState.count = uploadState.queue.length;
        pending = null;
        queueNext();
    };

    const resetState = () => ({
        queue: [],
        index: 0,
        count: 0,
        inProgress: false,
        activeInput: null,
        activeFile: null,
    });

    let uploadState = resetState();

    const queueNext = () => {
        if (uploadState.inProgress) {
            return;
        }
        if (uploadState.queue.length === 0) {
            uploadState = resetState();
            return;
        }
        const next = uploadState.queue[0];
        uploadState.inProgress = true;
        uploadState.activeInput = next.input;
        uploadState.activeFile = next.file;

        transport.send('change', {
            name: next.file.name,
            size: next.file.size,
            type: next.file.type,
            index: uploadState.index,
            count: uploadState.count
        });

        startUpload([next.file], next.input);
    };

    const advanceQueue = () => {
        uploadState.index++;
        uploadState.queue.shift();
        uploadState.inProgress = false;
        uploadState.activeInput = null;
        uploadState.activeFile = null;
        if (uploadState.queue.length > 0) {
            queueNext();
        } else {
            uploadState = resetState();
        }
    };

    transport.on('start', (config) => {
        uploadConfig = config || {};
        if (pending) {
            processPending(uploadConfig);
        } else {
            queueNext();
        }
    });

    transport.on('cancel', () => {
        cancelUpload();
        if (element && element.value) {
            element.value = '';
        }
    });

    element.addEventListener('change', handleChange);

    return () => {
        element.removeEventListener('change', handleChange);
        cancelUpload();
    };
}
