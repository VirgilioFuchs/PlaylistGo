import type { ButtonHTMLAttributes, ReactNode } from "react";
import styles from "./Button.module.css";

type ButtonVariant = "primary" | "secondary";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
  variant?: ButtonVariant;
  loading?: boolean;
  loadingLabel?: string;
  selected?: boolean;
};

export function Button({
  children,
  className,
  disabled = false,
  loading = false,
  loadingLabel = "Carregando...",
  selected = false,
  variant = "primary",
  ...props
}: ButtonProps) {
  const isDisabled = disabled || loading;
  const buttonClassName = [styles.button, styles[variant], selected ? styles.selected : "", className]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      {...props}
      className={buttonClassName}
      disabled={isDisabled}
      aria-busy={loading || undefined}
      aria-pressed={selected || undefined}
    >
      <span className={loading ? styles.labelLoading : styles.label}>{children}</span>
      {loading ? <span className={styles.loadingLabel}>{loadingLabel}</span> : null}
    </button>
  );
}
