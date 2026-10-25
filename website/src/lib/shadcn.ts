export { cn } from 'cn';

export type WithElementRef<Props, Element extends HTMLElement = HTMLElement> = Props & { ref?: Element | null };
