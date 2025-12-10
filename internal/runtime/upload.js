function upload(element, transport) {
    let tusPromise = null;
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

    const ensureTus = () => {
        if (window.tus) return Promise.resolve();
        if (tusPromise) return tusPromise;

        tusPromise = new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = 'https://cdn.jsdelivr.net/npm/tus-js-client@4.2.3/dist/tus.min.js';
            script.onload = resolve;
            script.onerror = () => reject(new Error('Failed to load tus-js-client'));
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

        ensureTus().then(() => {
            const upload = new window.tus.Upload(file, {
                endpoint: '/tus/',
                retryDelays: [0, 1000, 3000, 5000],
                metadata: {
                    token: token,
                    filename: file.name,
                    filetype: file.type,
                },
                onProgress: (loaded, total) => {
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
                    activeUpload = null;
                    transport.send('error', { error: error.message || 'Upload failed' });
                    advanceQueue();
                },
            });

            activeUpload = upload;
            upload.start();
        }).catch((error) => {
            transport.send('error', { error: error.message || 'Failed to initialize upload' });
        });
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
        transport.send('cancelled', {});
    });

    element.addEventListener('change', handleChange);

    return () => {
        element.removeEventListener('change', handleChange);
        cancelUpload();
        uploadState = resetState();
    };
}
