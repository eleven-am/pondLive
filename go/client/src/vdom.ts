import { StructuredNode, ClientNode } from './types';
import { Logger } from './logger';

export function hydrate(json: StructuredNode, dom: Node, refs?: Map<string, ClientNode>): ClientNode {
    const isWrapper = (!json.tag && !json.text && !json.comment && json.children) || json.fragment;

    const clientNode: ClientNode = { ...json, el: isWrapper ? null : dom, children: undefined };

    if (!isWrapper) {
        (dom as any).__pondNode = clientNode;
    }

    if (json.refId && refs) {
        refs.set(json.refId, clientNode);
    }

    if (json.tag) {
        if (dom.nodeType !== Node.ELEMENT_NODE) {
            throw new Error(`Hydration error: expected element <${json.tag}> but found nodeType ${dom.nodeType}`);
        } else {
            const el = dom as Element;
            if (el.tagName.toLowerCase() !== json.tag.toLowerCase()) {
                throw new Error(`Hydration error: expected tag <${json.tag}> but found <${el.tagName}>`);
            }
        }
    } else if (json.text !== undefined && json.text !== '') {
        if (dom.nodeType !== Node.TEXT_NODE) {
            throw new Error(`Hydration error: expected text node but found nodeType ${dom.nodeType}`);
        }
    }

    if (json.children && json.children.length > 0) {
        clientNode.children = [];

        let domChildren = Array.from(dom.childNodes).filter(shouldHydrate);

        if (json.tag === 'style') {
            domChildren = [];
        }

        const consumed = hydrateChildren(clientNode.children, json.children, domChildren, dom, refs);
        const expected = countRenderableNodes(json.children);
        if (consumed !== expected) {
            throw new Error(`Hydration error: expected ${expected} renderable children, hydrated ${consumed}`);
        }
    }

    return clientNode;
}

function hydrateChildren(
    target: ClientNode[],
    jsonChildren: StructuredNode[],
    domChildren: Node[],
    parentDom: Node,
    refs?: Map<string, ClientNode>
): number {
    let domIdx = 0;

    for (let i = 0; i < jsonChildren.length; i++) {
        const childJson = jsonChildren[i];

        
        const isWrapper = (!childJson.tag && !childJson.text && !childJson.comment && childJson.children) || childJson.fragment;

        if (isWrapper) {
            
            const wrapperNode: ClientNode = { ...childJson, el: null, children: [] };
            if (childJson.refId && refs) refs.set(childJson.refId, wrapperNode);

            if (childJson.children) {
                
                
                const consumed = hydrateChildrenWithConsumption(
                    wrapperNode.children!,
                    childJson.children,
                    domChildren,
                    domIdx,
                    parentDom,
                    refs
                );
                domIdx += consumed;
            }

            target.push(wrapperNode);
            continue;
        }

        
        if (childJson.text === '') {
            const childDom = domChildren[domIdx];
            if (!childDom || childDom.nodeType !== Node.TEXT_NODE) {
                throw new Error(`Hydration error: expected empty text node at index ${i}`);
            }
            const childNode = hydrate(childJson, childDom, refs);
            target.push(childNode);
            domIdx++;
            continue;
        }

        const childDom = domChildren[domIdx];

        if (!childDom) {
            Logger.error('Hydration', 'Missing DOM node', {
                parentTag: (parentDom as any).tagName,
                expectedIndex: i,
                domChildrenCount: domChildren.length,
                jsonChildrenCount: jsonChildren.length,
                jsonChildSummary: {
                    tag: childJson.tag,
                    text: childJson.text,
                    comment: childJson.comment,
                    key: childJson.key,
                    componentId: childJson.componentId,
                },
            });
            throw new Error(`Hydration error: missing DOM node for child index ${i}`);
        }

        const childNode = hydrate(childJson, childDom, refs);
        target.push(childNode);
        domIdx++;
    }

    return domIdx;
}


function hydrateChildrenWithConsumption(
    target: ClientNode[],
    jsonChildren: StructuredNode[],
    domChildren: Node[],
    startIdx: number,
    parentDom: Node,
    refs?: Map<string, ClientNode>
): number {
    let domIdx = startIdx;

    for (let i = 0; i < jsonChildren.length; i++) {
        const childJson = jsonChildren[i];

        
        const isWrapper = (!childJson.tag && !childJson.text && !childJson.comment && childJson.children) || childJson.fragment;

        if (isWrapper) {
            
            const wrapperNode: ClientNode = { ...childJson, el: null, children: [] };
            if (childJson.refId && refs) refs.set(childJson.refId, wrapperNode);

            if (childJson.children) {
                const consumed = hydrateChildrenWithConsumption(
                    wrapperNode.children!,
                    childJson.children,
                    domChildren,
                    domIdx,
                    parentDom,
                    refs
                );
                domIdx += consumed;
            }

            target.push(wrapperNode);
            continue;
        }

        
        if (childJson.text === '') {
            const childDom = domChildren[domIdx];
            if (!childDom || childDom.nodeType !== Node.TEXT_NODE) {
                throw new Error(`Hydration error: expected empty text node at index ${i}`);
            }
            const childNode = hydrate(childJson, childDom, refs);
            target.push(childNode);
            domIdx++;
            continue;
        }

        const childDom = domChildren[domIdx];

        if (!childDom) {
            throw new Error(`Hydration error: missing DOM node for child index ${i}`);
        }

        const childNode = hydrate(childJson, childDom, refs);
        target.push(childNode);
        domIdx++;
    }

    return domIdx - startIdx;
}

function shouldHydrate(node: Node): boolean {
    if (node.nodeType === Node.COMMENT_NODE) return false;
    if (node.nodeType === Node.TEXT_NODE) {
        const text = (node as Text).data;
        if (text.trim() === '') return false;
    }
    return true;
}

function countRenderableNodes(nodes: StructuredNode[] | undefined): number {
    if (!nodes || nodes.length === 0) return 0;
    let count = 0;
    for (const n of nodes) {
        const isWrapper = (!n.tag && !n.text && !n.comment && n.children) || n.fragment;
        if (isWrapper) {
            count += countRenderableNodes(n.children);
            continue;
        }
        if (n.tag || n.text !== undefined || n.comment) {
            count++;
        }
    }
    return count;
}
