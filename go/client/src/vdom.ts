import { Logger } from './logger';
import { StructuredNode, ClientNode } from './types';

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

        const domChildren = Array.from(dom.childNodes).filter(shouldHydrate);

        const consumed = hydrateChildren(clientNode.children, json.children, domChildren, dom, refs);
        if (consumed !== json.children.length) {
            throw new Error(`Hydration error: expected ${json.children.length} children, hydrated ${consumed}`);
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
            // Empty text nodes must already exist in SSR DOM; do not create them here.
            const childDom = domChildren[domIdx];
            if (!childDom || childDom.nodeType !== Node.TEXT_NODE) {
                throw new Error(`Hydration error: expected empty text node at index ${i}`);
            }
            const childNode = hydrate(childJson, childDom, refs);
            target.push(childNode);
            domIdx++;
            continue;
        }

        // Non-empty child: must have a corresponding DOM node
        const childDom = domChildren[domIdx];

        if (!childDom) {
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

        // Non-empty child: must have a corresponding DOM node
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

function shouldHydrate(_node: Node): boolean {
    
    
    
    
    
    
    
    return true;
}
