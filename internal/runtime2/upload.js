function upload(element, transport) {
    let activeXhr = null;
    let uploadConfig = null;
    let pending = null;

    const resetState = () => ({
        queue: [],
        index: 0,
        count: 0,
        inProgress: false,
    });

    let uploadState = resetState();

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

    const cancelUpload = () => {
        if (activeXhr) {
            activeXhr.abort();
            activeXhr = null;
        }
    };

    const startUpload = (file, input) => {
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
        xhr.open('POST', target, true);

        if (uploadConfig.token) {
            xhr.setRequestHeader('X-Upload-Token', uploadConfig.token);
        }

        xhr.upload.onprogress = (event) => {
            const loaded = event.loaded;
            const total = event.lengthComputable ? event.total : file.size;
            transport.send('progress', {
                loaded,
                total,
                name: file.name,
                size: file.size,
                index: uploadState.index,
                count: uploadState.count
            });
        };

        xhr.onerror = () => {
            activeXhr = null;
            transport.send('error', { error: 'Upload failed (network)' });
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
                const msg = xhr.responseText || `Upload failed (${xhr.status})`;
                transport.send('error', { error: msg });
            } else {
                const totalSize = file.size;
                transport.send('progress', {
                    loaded: totalSize,
                    total: totalSize,
                    name: file.name,
                    size: file.size,
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
        form.append('file', file);
        xhr.send(form);
        activeXhr = xhr;
    };

    const acceptsFile = (file, acceptList) => {
        if (!acceptList || acceptList.length === 0) return true;
        const fileType = file.type;
        const fileName = file.name;
        return acceptList.some(pattern => {
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

    const processPending = (config) => {
        if (!pending) return;
        const { files, input } = pending;

        // Validate each file per config
        for (const f of files) {
            if (config.maxSize && config.maxSize > 0 && f.size > config.maxSize) {
                transport.send('error', { error: `File exceeds maximum size (${config.maxSize} bytes)` });
                input.value = '';
                pending = null;
                return;
            }
            if (!acceptsFile(f, config.accept)) {
                transport.send('error', { error: `File type not accepted. Allowed: ${config.accept.join(', ')}` });
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

    const queueNext = () => {
        if (uploadState.inProgress) return;
        if (uploadState.queue.length === 0) {
            uploadState = resetState();
            return;
        }
        const next = uploadState.queue[0];
        uploadState.inProgress = true;

        transport.send('change', {
            name: next.file.name,
            size: next.file.size,
            type: next.file.type,
            index: uploadState.index,
            count: uploadState.count
        });

        startUpload(next.file, next.input);
    };

    const advanceQueue = () => {
        uploadState.index++;
        uploadState.queue.shift();
        uploadState.inProgress = false;
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
        uploadState = resetState();
    });

    element.addEventListener('change', handleChange);

    return () => {
        element.removeEventListener('change', handleChange);
        cancelUpload();
        uploadState = resetState();
    };
}
