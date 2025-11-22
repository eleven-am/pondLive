function upload(element, transport) {
    let activeXhr = null;
    let uploadConfig = null;
    let pendingFile = null;

    const handleChange = (event) => {
        const input = event.target;
        const files = input.files;

        if (!files || files.length === 0) {
            return;
        }

        const file = files[0];
        pendingFile = { file, input };

        transport.send('ready', {
            name: file.name,
            size: file.size,
            type: file.type
        });
    };

    const startUpload = (file, input) => {
        if (activeXhr) {
            activeXhr.abort();
            activeXhr = null;
        }

        const { endpoint, sessionId, uploadId } = uploadConfig;
        const target = `${endpoint}/${encodeURIComponent(sessionId)}/${encodeURIComponent(uploadId)}`;
        const xhr = new XMLHttpRequest();

        xhr.upload.onprogress = (event) => {
            const loaded = event.loaded;
            const total = event.lengthComputable ? event.total : file.size;
            transport.send('progress', { loaded, total });
        };

        xhr.onerror = () => {
            activeXhr = null;
            transport.send('error', { error: 'Upload failed' });
        };

        xhr.onabort = () => {
            activeXhr = null;
            transport.send('cancelled', {});
        };

        xhr.onload = () => {
            activeXhr = null;
            if (xhr.status < 200 || xhr.status >= 300) {
                transport.send('error', { error: `Upload failed (${xhr.status})` });
            } else {
                transport.send('progress', { loaded: file.size, total: file.size });
                if (input) {
                    input.value = '';
                }
            }
        };

        const form = new FormData();
        form.append('file', file);
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

    transport.on('start', (config) => {
        uploadConfig = config;

        if (pendingFile) {
            const { file, input } = pendingFile;

            if (config.maxSize && config.maxSize > 0 && file.size > config.maxSize) {
                transport.send('error', {
                    error: `File exceeds maximum size (${config.maxSize} bytes)`
                });
                input.value = '';
                pendingFile = null;
                return;
            }

            if (config.accept && config.accept.length > 0) {
                const fileType = file.type;
                const fileName = file.name;
                const accepted = config.accept.some(pattern => {
                    if (pattern.startsWith('.')) {
                        return fileName.endsWith(pattern);
                    }
                    if (pattern.includes('*')) {
                        const regex = new RegExp('^' + pattern.replace(/\*/g, '.*') + '$');
                        return regex.test(fileType);
                    }
                    return fileType === pattern;
                });

                if (!accepted) {
                    transport.send('error', {
                        error: `File type not accepted. Allowed: ${config.accept.join(', ')}`
                    });
                    input.value = '';
                    pendingFile = null;
                    return;
                }
            }

            transport.send('change', {
                name: file.name,
                size: file.size,
                type: file.type
            });

            startUpload(file, input);
            pendingFile = null;
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
