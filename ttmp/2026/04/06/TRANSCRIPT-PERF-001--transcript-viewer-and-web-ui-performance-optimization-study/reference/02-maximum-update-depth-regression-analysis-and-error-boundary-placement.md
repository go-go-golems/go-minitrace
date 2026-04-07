---
Title: Maximum update depth regression analysis and error boundary placement
Ticket: TRANSCRIPT-PERF-001
Status: active
Topics:
    - performance
    - frontend
    - react
    - web-ui
    - transcript-analysis
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/App.tsx
      Note: Recommended app-shell / route-level error boundary location
    - Path: web/src/components/SessionBrowser/SessionBrowser.tsx
      Note: Second call site sharing the same virtualization hook risk
    - Path: web/src/components/TranscriptViewer/TranscriptViewer.tsx
      Note: Focused transcript navigation and virtual-row measurement call site
    - Path: web/src/components/shared/useVirtualList.ts
      Note: Likely root cause of the maximum-update-depth loop via callback-ref measurement and synchronous state writes
    - Path: web/src/pages/TranscriptViewerPage.tsx
      Note: Recommended transcript-specific error boundary location
ExternalSources: []
Summary: Short analysis of the maximum-update-depth regression introduced after virtualization, plus recommended error-boundary placement.
LastUpdated: 2026-04-06T22:35:00-04:00
WhatFor: Explain the likely root cause of the infinite update loop and identify where error boundaries belong in the current app structure.
WhenToUse: Use when fixing the virtualization regression or deciding how to contain future frontend crashes in the transcript and session routes.
---


# Maximum update depth regression analysis and error boundary placement

## Goal

Explain why the focused transcript URL can trigger `Maximum update depth exceeded`, identify the most likely faulty code path, and recommend where error boundaries should be added in the current app.

## Context

The failing URL is a transcript route with both focus and composer state encoded in the query string:

```text
/sessions/019d0295-d06b-7033-b154-a991a94672b6?tab=transcript&focusType=tool_call&focusId=call_nLyNZ7JQ6qoGkVMVyGeE5fIN&composeType=tool_call&composeTarget=call_nLyNZ7JQ6qoGkVMVyGeE5fIN
```

The runtime error points to:

- `measureElement` in `web/src/components/shared/useVirtualList.ts:112`
- React state dispatch inside the ref/measurement path

This matters because the failure is not isolated to transcript code. The hook is shared and also powers Session Browser virtualization.

## Quick Reference

## 1) What is most likely wrong?

### Primary root cause

The most likely bug is that `useVirtualList.measureElement` is implemented as a **callback-ref factory that returns a new ref callback every render**:

```ts
const measureElement = useCallback(
  (index: number) => (node: HTMLElement | null) => {
    ...
    setMeasuredHeights(...)
  },
  [],
);
```

and is then used like this:

```tsx
ref={measureElement(item.index)}
```

That means:

- every render creates a fresh ref callback for every virtual row,
- React clears the old ref and sets the new ref during commit,
- the new ref callback immediately runs measurement logic,
- that callback calls `setMeasuredHeights(...)` synchronously,
- which triggers another render,
- which creates another new ref callback,
- and the cycle can repeat until React hits maximum update depth.

### Why the focused tool-call URL makes it easier to trigger

This specific URL is not just “open transcript.” It also asks the viewer to:

1. focus a tool-call target,
2. force-expand the containing block,
3. show the inline composer for that same tool-call target,
4. scroll the virtual list toward the focused block,
5. then scroll the inner DOM anchor into view.

That combination increases layout churn and makes the virtual-row measurement path much hotter.

### Secondary contributors

These are probably not the root bug, but they amplify it:

- `TranscriptViewer` scrolls to the focused block via `scrollToIndex(...)`
- scroll updates `scrollTop` in `useVirtualList`
- that recomputes `visibleRange`
- that changes the rendered virtual items
- new virtual items means more ref callback churn
- `measureElement` also re-observes elements on every callback run
- `content-visibility` / expanded body layout can make measured heights settle over multiple commits

In short: the ref callback is doing stateful work at exactly the wrong lifecycle boundary.

## 2) Where in code is the likely fault?

### Highest-confidence fault site

**File:** `web/src/components/shared/useVirtualList.ts`

Relevant lines:

- measurement callback factory: around `95-118`
- synchronous state write inside ref callback: around `110-114`

This line is the smoking gun:

```ts
setMeasuredHeights((prevHeights) =>
  prevHeights[index] === height ? prevHeights : { ...prevHeights, [index]: height },
);
```

Even though there is a guard, it still runs from a callback ref that is recreated every render.

### Supporting call sites

**Transcript route**
- `web/src/components/TranscriptViewer/TranscriptViewer.tsx`
- `ref={measureElement(item.index)}` inside the virtualized block list

**Session Browser route**
- `web/src/components/SessionBrowser/SessionBrowser.tsx`
- also uses the same hook and the same callback-ref pattern

So this is a shared-hook bug, not a transcript-only bug.

## 3) What is the likely fix direction?

Not implementing it in this note yet, but the correct direction is:

### A. Make row ref callbacks stable per index

Instead of returning a new callback every render, cache one callback per index.

Conceptually:

```ts
const refCallbacks = useRef(new Map<number, (node: HTMLElement | null) => void>());

function getMeasureRef(index: number) {
  let cb = refCallbacks.current.get(index);
  if (!cb) {
    cb = (node) => {
      ...
    };
    refCallbacks.current.set(index, cb);
  }
  return cb;
}
```

### B. Stop calling `setMeasuredHeights` directly from the ref callback

The ref callback should preferably:

- register/unregister the DOM node,
- attach `ResizeObserver`,
- maybe store the node reference,
- but avoid synchronous React state writes during ref attachment.

The `ResizeObserver` path should be the primary measurement-update path.

### C. Avoid unobserve/observe churn unless the node identity actually changed

Right now the hook unobserves/observes on every callback run. That is safe only if the callback itself is stable. Once the callback churns, observer churn becomes part of the problem.

## 4) Where should error boundaries go?

### Important principle

Error boundaries are **not** the fix for this bug. They are containment.

They should prevent one broken high-risk subtree from taking down the whole SPA, but they should not be used to mask the loop.

### Recommended placement

#### Boundary 1: Global app-shell boundary

**Place:** around routed content in `web/src/App.tsx` or just inside `AppLayout`

**Why:**
- preserves a usable shell if a route crashes
- gives the user a controlled fallback instead of a React red screen / blank app

**What it should catch:**
- any unhandled render/commit crash in route trees

**What the fallback should show:**
- “The page crashed”
- current route
- a link/button back to `/sessions`
- optional “reload app” action

#### Boundary 2: Transcript route boundary

**Place:** in `web/src/pages/TranscriptViewerPage.tsx`, around `<TranscriptViewer ... />`

**Why:**
- transcript viewer is currently the highest-risk subtree
- it now contains virtualization, focused scrolling, collapses, annotations, and deep-link state
- if it fails, the user should still keep the page-level context rather than losing the whole app

**Fallback should include:**
- session ID
- “Back to sessions” button
- optional “Open query” button if helpful
- short technical message mentioning transcript rendering failed

#### Boundary 3: Session Browser route boundary

**Place:** in `web/src/App.tsx` or `SessionBrowserPage.tsx`

**Why:**
- Session Browser now shares the same virtualization hook
- same failure class can happen there too

#### Optional Boundary 4: Query route boundary

**Place:** around `QueryEditorPage`

**Why:**
- not because of this exact bug,
- but because it is another heavy interactive surface with large result rendering

### Where *not* to put them

#### Not per row / per block / per tool-call

Do **not** add boundaries inside:

- each `BlockCard`
- each virtual row
- each `ToolCallRow`

Reasons:
- too noisy
- too much component overhead
- poor fallback UX
- does not address the real failure domain, which is the shared virtualized subtree

#### Not inside `useVirtualList`

Hooks are not where boundaries belong. The boundary belongs at the component subtree that uses the hook.

## 5) Best practical boundary strategy for this app

If I were implementing boundaries next, I would do this order:

1. add a reusable `AppErrorBoundary` component,
2. wrap each route element in `App.tsx`,
3. add a more specific boundary around `TranscriptViewer` in `TranscriptViewerPage.tsx`,
4. optionally add one around `SessionBrowserPage` because it shares the virtualization hook.

That gives:

- coarse global containment,
- plus focused containment for the highest-risk route.

## Usage Examples

### Immediate debugging conclusion

If the crash stack points at `measureElement` again, I would assume the problem is still in the shared virtualization ref/measurement lifecycle before suspecting transcript-specific logic.

### Immediate engineering plan

1. fix `useVirtualList` first,
2. then add route-level error boundaries,
3. then retest both transcript and Session Browser deep links.

## Recommended next action

I should next do two things, in this order:

1. **Fix `useVirtualList`**
   - stabilize ref callbacks per index
   - remove synchronous `setMeasuredHeights` writes from the ref callback if possible
   - reduce observe/unobserve churn

2. **Add error boundaries**
   - app-shell route boundary in `App.tsx`
   - transcript route boundary in `TranscriptViewerPage.tsx`
   - likely Session Browser route boundary too, since it uses the same hook
