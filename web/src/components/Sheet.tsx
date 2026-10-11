import { useEffect, useRef } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import { IconButton } from "./IconButton";
import styles from "./Sheet.module.css";

type SheetProps = {
  title: string;
  open: boolean;
  onClose: () => void;
  locked?: boolean;
  header?: ReactNode;
  bodyClassName?: string;
  children: ReactNode;
};

function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(
    container.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  );
}

export function Sheet({
  title,
  open,
  onClose,
  locked = false,
  header,
  bodyClassName,
  children
}: SheetProps) {
  const panelRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!open) return;

    const background = Array.from(
      document.querySelectorAll<HTMLElement>(".shell-tab-panel, [data-mini-player-closed='true'], nav[aria-label='Navegação principal']")
    );
    const previousState = background.map((element) => ({
      element,
      ariaHidden: element.getAttribute("aria-hidden"),
      inert: element.inert
    }));

    background.forEach((element) => {
      element.setAttribute("aria-hidden", "true");
      element.inert = true;
    });
    panelRef.current?.querySelector<HTMLElement>("button:not([disabled]), input:not([disabled])")?.focus();

    return () => {
      previousState.forEach(({ element, ariaHidden, inert }) => {
        if (ariaHidden === null) {
          element.removeAttribute("aria-hidden");
        } else {
          element.setAttribute("aria-hidden", ariaHidden);
        }
        element.inert = inert;
      });
    };
  }, [open]);

  if (!open) {
    return null;
  }

  function handleKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (event.key === "Escape" && !locked) {
      event.preventDefault();
      onClose();
      return;
    }

    if (event.key !== "Tab") return;

    const panel = panelRef.current;
    if (!panel) return;

    const focusable = getFocusableElements(panel);
    if (focusable.length === 0) {
      event.preventDefault();
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  const composedBodyClassName = [styles.body, bodyClassName].filter(Boolean).join(" ");

  return (
    <div className={styles.sheet} role="dialog" aria-modal="true" aria-label={title} onKeyDown={handleKeyDown}>
      <div className={styles.scrim} onClick={locked ? undefined : onClose} aria-hidden="true" />
      <section className={styles.panel} ref={panelRef} tabIndex={-1}>
        {header ? <header className={styles.customHeader}>{header}</header> : (
          <header className={styles.header}>
            <div className={styles.handle} aria-hidden="true" />
            <div className={styles.titleBlock}>
              <h2>{title}</h2>
            </div>
            <div className={styles.actions}>
              <IconButton type="button" onClick={onClose} disabled={locked} aria-label={`Fechar ${title}`}>
                <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                  <path d="m7 7 10 10M17 7 7 17" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                </svg>
              </IconButton>
            </div>
          </header>
        )}
        <div className={composedBodyClassName}>{children}</div>
      </section>
    </div>
  );
}
