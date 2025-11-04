const removeElementAndSiblings = (parent: Element, childIndex: number) => {
    const children = parent.childNodes;

    for (let i = children.length - 1; i >= childIndex; i--) {
        const child = children[i];

        child.remove();
    }
};

function replaceElements (obj1: Element, obj2: Element): void {
    if (obj1 instanceof Text && obj2 instanceof Text && obj1.textContent !== obj2.textContent) {
        obj1.replaceWith(obj2.cloneNode(true));

        return;
    }

    if (obj1 instanceof Text && obj2 instanceof Element) {
        obj1.replaceWith(obj2.cloneNode(true));

        return;
    }

    if (obj1 instanceof Element && obj2 instanceof Text) {
        obj1.replaceWith(obj2.cloneNode(true));

        return;
    }

    if (obj1.tagName !== obj2.tagName) {
        obj1.replaceWith(obj2.cloneNode(true));

        return;
    }

    // eslint-disable-next-line guard-for-in
    for (const key in obj2.attributes) {
        const attribute = obj2.attributes[key];

        if (attribute && attribute.name && attribute.value) {
            if (!obj1.hasAttribute(attribute.name) || obj1.getAttribute(attribute.name) !== attribute.value) {
                obj1.setAttribute(attribute.name, attribute.value);
            }
        }
    }

    // eslint-disable-next-line guard-for-in
    for (const key in obj1.attributes) {
        const attribute = obj1.attributes[key];

        if (attribute && attribute.name && attribute.value) {
            if (!obj2.hasAttribute(attribute.name)) {
                obj1.removeAttribute(attribute.name);
            } else if (obj2.hasAttribute(attribute.name) && obj2.getAttribute(attribute.name) !== attribute.value) {
                obj1.setAttribute(attribute.name, obj2.getAttribute(attribute.name)!);
            }
        }
    }

    const maxChildren = Math.max(obj1.childNodes.length, obj2.childNodes.length);

    for (let i = 0; i < maxChildren; i++) {
        const child1 = obj1.childNodes[i] as Element;
        const child2 = obj2.childNodes[i] as Element;

        if (child1 && child2) {
            replaceElements(child1, child2);
        } else if (child1) {
            return removeElementAndSiblings(obj1, i);
        } else if (child2) {
            obj1.appendChild(child2.cloneNode(true));
        }
    }
}

export function updateTheDom (htmlString: string) {
    const div = document.createElement('div');
    const app = document.getElementById('app');

    if (!app) {
        throw new Error('App element not found');
    }

    div.innerHTML = htmlString.trim();
    div.setAttribute('id', 'app');

    replaceElements(app, div);
}

