import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type RefObject,
} from "react";

interface UseVirtualListOptions {
  count: number;
  scrollContainerRef: RefObject<HTMLElement | null>;
  estimateSize: (index: number) => number;
  overscan?: number;
  enabled?: boolean;
}

export interface VirtualItem {
  index: number;
  start: number;
  size: number;
  end: number;
}

export function useVirtualList({
  count,
  scrollContainerRef,
  estimateSize,
  overscan = 3,
  enabled = true,
}: UseVirtualListOptions) {
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(0);
  const [measuredHeights, setMeasuredHeights] = useState<Record<number, number>>({});
  const observerRef = useRef<ResizeObserver | null>(null);
  const elementMapRef = useRef(new Map<number, HTMLElement>());
  const refCallbackMapRef = useRef(new Map<number, (node: HTMLElement | null) => void>());

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) {
      return;
    }

    const updateMetrics = () => {
      setScrollTop((current) =>
        current === container.scrollTop ? current : container.scrollTop,
      );
      setViewportHeight((current) =>
        current === container.clientHeight ? current : container.clientHeight,
      );
    };

    updateMetrics();
    container.addEventListener("scroll", updateMetrics, { passive: true });

    const resizeObserver = new ResizeObserver(updateMetrics);
    resizeObserver.observe(container);

    return () => {
      container.removeEventListener("scroll", updateMetrics);
      resizeObserver.disconnect();
    };
  }, [scrollContainerRef]);

  useEffect(() => {
    const resizeObserver = new ResizeObserver((entries) => {
      setMeasuredHeights((prev) => {
        let changed = false;
        const next = { ...prev };

        for (const entry of entries) {
          const el = entry.target as HTMLElement;
          const index = Number(el.dataset.virtualIndex);
          if (!Number.isFinite(index)) {
            continue;
          }
          const height = Math.ceil(entry.contentRect.height);
          if (height > 0 && next[index] !== height) {
            next[index] = height;
            changed = true;
          }
        }

        return changed ? next : prev;
      });
    });

    observerRef.current = resizeObserver;
    for (const element of elementMapRef.current.values()) {
      resizeObserver.observe(element);
    }

    return () => {
      resizeObserver.disconnect();
      observerRef.current = null;
    };
  }, []);

  const measureElement = useCallback((index: number) => {
    const existing = refCallbackMapRef.current.get(index);
    if (existing) {
      return existing;
    }

    const callback = (node: HTMLElement | null) => {
      const prev = elementMapRef.current.get(index);
      if (prev === node) {
        return;
      }
      if (prev && observerRef.current) {
        observerRef.current.unobserve(prev);
      }
      if (!node) {
        elementMapRef.current.delete(index);
        return;
      }

      node.dataset.virtualIndex = String(index);
      elementMapRef.current.set(index, node);
      observerRef.current?.observe(node);
    };

    refCallbackMapRef.current.set(index, callback);
    return callback;
  }, []);

  const metrics = useMemo(() => {
    const starts = new Array<number>(count);
    const sizes = new Array<number>(count);
    let totalSize = 0;

    for (let index = 0; index < count; index += 1) {
      starts[index] = totalSize;
      const size = measuredHeights[index] ?? estimateSize(index);
      sizes[index] = size;
      totalSize += size;
    }

    return { starts, sizes, totalSize };
  }, [count, estimateSize, measuredHeights]);

  const visibleRange = useMemo(() => {
    if (!enabled || count === 0) {
      return { startIndex: 0, endIndex: count - 1 };
    }

    const viewTop = Math.max(0, scrollTop);
    const viewBottom = viewTop + Math.max(viewportHeight, 1);

    let startIndex = 0;
    while (
      startIndex < count - 1 &&
      metrics.starts[startIndex] + metrics.sizes[startIndex] < viewTop
    ) {
      startIndex += 1;
    }

    let endIndex = startIndex;
    while (endIndex < count - 1 && metrics.starts[endIndex] < viewBottom) {
      endIndex += 1;
    }

    startIndex = Math.max(0, startIndex - overscan);
    endIndex = Math.min(count - 1, endIndex + overscan);

    return { startIndex, endIndex };
  }, [count, enabled, metrics.sizes, metrics.starts, overscan, scrollTop, viewportHeight]);

  const virtualItems = useMemo(() => {
    if (count === 0) {
      return [] as VirtualItem[];
    }

    const items: VirtualItem[] = [];
    const startIndex = enabled ? visibleRange.startIndex : 0;
    const endIndex = enabled ? visibleRange.endIndex : count - 1;

    for (let index = startIndex; index <= endIndex; index += 1) {
      items.push({
        index,
        start: metrics.starts[index],
        size: metrics.sizes[index],
        end: metrics.starts[index] + metrics.sizes[index],
      });
    }

    return items;
  }, [count, enabled, metrics.sizes, metrics.starts, visibleRange.endIndex, visibleRange.startIndex]);

  const scrollToIndex = useCallback(
    (index: number, behavior: ScrollBehavior = "auto", align: "start" | "center" = "start") => {
      const container = scrollContainerRef.current;
      if (!container || index < 0 || index >= count) {
        return;
      }

      const size = metrics.sizes[index] ?? estimateSize(index);
      let top = metrics.starts[index] ?? 0;
      if (align === "center") {
        top = Math.max(0, top - container.clientHeight / 2 + size / 2);
      }
      container.scrollTo({ top, behavior });
    },
    [count, estimateSize, metrics.sizes, metrics.starts, scrollContainerRef],
  );

  return {
    virtualItems,
    totalSize: metrics.totalSize,
    topSpacerHeight: virtualItems[0]?.start ?? 0,
    bottomSpacerHeight:
      metrics.totalSize - (virtualItems[virtualItems.length - 1]?.end ?? 0),
    measureElement,
    scrollToIndex,
  };
}
