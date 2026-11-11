function upload(element, transport) {
    let tusPromise = null;
    let tusSrc = null;
    let generation = 0;
    let activeUpload = null;
    let uploadConfig = null;
    let pending = null;

    const resetState = () => ({
        queue: [],
        index: 0,
        count: 0,
        inProgress: false,
    });

    let uploadState = resetState();

    const ensureTus = (src) => {
        if (window.tus) return Promise.resolve();
        if (!src) return Promise.reject(new Error('Upload script not configured'));
        if (tusPromise && tusSrc === src) return tusPromise;

        tusSrc = src;
        tusPromise = new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = src;
            script.onload = resolve;
            script.onerror = () => {
                tusPromise = null;
                script.remove();
                reject(new Error('Failed to load tus-js-client'));
            };
            document.head.appendChild(script);
        });

        return tusPromise;
    };

    const handleChange = (event) => {
        const input = event.target;
        const files = input.files;
        if (!files || files.length === 0) {
            return;
        }
        const fileArray = Array.from(files);
        cancelUpload();
        uploadState = resetState();
        pending = { files: fileArray, input };
        uploadConfig = null;

        const first = fileArray[0];
        transport.send('ready', {
            name: first.name,
            size: first.size,
            type: first.type,
            count: fileArray.length
        });
    };

    const cancelUpload = () => {
        generation++;
        if (activeUpload) {
            activeUpload.abort(true);
            activeUpload = null;
        }
    };

    const startUpload = (file, input) => {
        if (activeUpload) {
            activeUpload.abort(true);
            activeUpload = null;
        }

        const token = uploadConfig?.token;
        if (!token) {
            transport.send('error', { error: 'Upload token not configured' });
            return;
        }
        if (!uploadConfig.endpoint) {
            transport.send('error', { error: 'Upload endpoint not configured' });
            return;
        }

        const ticket = ++generation;
        const current = () => ticket === generation;

        ensureTus(uploadConfig.tusScript).then(() => {
            if (!current()) return;

            const upload = new window.tus.Upload(file, {
                endpoint: uploadConfig.endpoint,
                retryDelays: [0, 1000, 3000, 5000],
                storeFingerprintForResuming: false,
                headers: { 'X-Upload-Token': token },
                metadata: {
                    filename: file.name,
                    filetype: file.type,
                },
                onProgress: (loaded, total) => {
                    if (!current()) return;
                    transport.send('progress', {
                        loaded,
                        total,
                        name: file.name,
                        size: file.size,
                        index: uploadState.index,
                        count: uploadState.count
                    });
                },
                onSuccess: () => {
                    if (!current()) return;
                    activeUpload = null;
                    transport.send('progress', {
                        loaded: file.size,
                        total: file.size,
                        name: file.name,
                        size: file.size,
                        index: uploadState.index,
                        count: uploadState.count
                    });
                    transport.send('complete', {
                        name: file.name,
                        size: file.size,
                        type: file.type,
                        index: uploadState.index,
                        count: uploadState.count
                    });
                    if (input) {
                        input.value = '';
                    }
                    advanceQueue();
                },
                onError: (error) => {
                    if (!current()) return;
                    activeUpload = null;
                    transport.send('error', { error: error.message || 'Upload failed' });
                    advanceQueue();
                },
            });

            activeUpload = upload;
            upload.start();
        }).catch((error) => {
            if (!current()) return;
            transport.send('error', { error: error.message || 'Failed to initialize upload' });
        });
    };

    const normalizeType = (value) => String(value || '').split(';')[0].trim().toLowerCase();

    const acceptsFile = (file, acceptList, unverifiable) => {
        if (!acceptList || acceptList.length === 0) return true;
        const fileType = normalizeType(file.type);
        const fileName = String(file.name || '').toLowerCase();
        return acceptList.some((raw) => {
            const pattern = String(raw).trim().toLowerCase();
            if (pattern === '') return false;
            if (unverifiable && unverifiable.some((u) => String(u).trim().toLowerCase() === pattern)) return false;
            if (pattern === '*' || pattern === '*/*') return true;
            if (pattern.startsWith('.')) return fileName.endsWith(pattern);
            if (pattern.endsWith('/*')) return fileType !== '' && fileType.startsWith(pattern.slice(0, -1));
            return fileType === pattern;
        });
    };

    const processPending = (config) => {
        if (!pending) return;
        const { files, input } = pending;

        if (!config.multiple && files.length > 1) {
            transport.send('error', { error: 'Only one file may be uploaded at a time' });
            input.value = '';
            pending = null;
            return;
        }

        for (const f of files) {
            if (config.maxSize && config.maxSize > 0 && f.size > config.maxSize) {
                transport.send('error', { error: `File exceeds maximum size (${config.maxSize} bytes)` });
                input.value = '';
                pending = null;
                return;
            }

            if (!acceptsFile(f, config.accept, config.unverifiable)) {
                transport.send('error', { error: `File type not accepted. Allowed: ${(config.accept || []).join(', ')}` });
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
        pending = null;
        if (element && element.value) {
            element.value = '';
        }
        uploadState = resetState();
        transport.send('cancelled', {});
    });

    element.addEventListener('change', handleChange);

    return () => {
        element.removeEventListener('change', handleChange);
        cancelUpload();
        uploadState = resetState();
    };
}
