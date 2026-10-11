import { useId, type ReactNode } from "react";

type SurfaceSectionProps = {
  eyebrow: string;
  title: string;
  description?: string;
  actions?: ReactNode;
  meta?: ReactNode;
  children?: ReactNode;
  className?: string;
};

export function SurfaceSection({
  eyebrow,
  title,
  description,
  actions,
  meta,
  children,
  className
}: SurfaceSectionProps) {
  const titleId = useId();

  return (
    <section className={className ? `surface-section ${className}` : "surface-section"} aria-labelledby={titleId}>
      <header className="surface-section__header">
        <div className="surface-section__copy">
          <p className="surface-section__eyebrow">{eyebrow}</p>
          <h2 id={titleId} className="surface-section__title">
            {title}
          </h2>
          {description ? <p className="surface-section__description">{description}</p> : null}
        </div>

        {meta || actions ? (
          <div className="surface-section__aside">
            {meta ? <div className="surface-section__meta">{meta}</div> : null}
            {actions ? <div className="surface-section__actions">{actions}</div> : null}
          </div>
        ) : null}
      </header>

      {children ? <div className="surface-section__body">{children}</div> : null}
    </section>
  );
}
