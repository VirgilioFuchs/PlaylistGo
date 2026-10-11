import type { ReactNode } from "react";
import { Button } from "./Button";

type StatusBannerVariant = "info" | "success" | "warning" | "error";

type StatusBannerProps = {
  title: string;
  message: string;
  variant?: StatusBannerVariant;
  actionLabel?: string;
  onAction?: () => void;
  secondaryActionLabel?: string;
  onSecondaryAction?: () => void;
  children?: ReactNode;
};

export function StatusBanner({
  title,
  message,
  variant = "info",
  actionLabel,
  onAction,
  secondaryActionLabel,
  onSecondaryAction,
  children
}: StatusBannerProps) {
  return (
    <section className={`status-banner status-banner--${variant}`} aria-live={variant === "error" ? "assertive" : "polite"}>
      <div className="status-banner__body">
        <p className="status-banner__title">{title}</p>
        <p className="status-banner__message">{message}</p>
        {children ? <div className="status-banner__extra">{children}</div> : null}
      </div>
      {(actionLabel || secondaryActionLabel) && (
        <div className="status-banner__actions">
          {secondaryActionLabel && onSecondaryAction ? (
            <Button type="button" variant="secondary" onClick={onSecondaryAction}>
              {secondaryActionLabel}
            </Button>
          ) : null}
          {actionLabel && onAction ? (
            <Button type="button" onClick={onAction}>
              {actionLabel}
            </Button>
          ) : null}
        </div>
      )}
    </section>
  );
}
