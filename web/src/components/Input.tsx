import { forwardRef, useId } from "react";
import type { InputHTMLAttributes, ReactNode } from "react";
import styles from "./Input.module.css";

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  error?: ReactNode;
  label?: ReactNode;
  leadingIcon?: ReactNode;
};

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { "aria-describedby": ariaDescribedBy, className, error, label, leadingIcon, ...props },
  ref
) {
  const errorId = useId();
  const describedBy = [ariaDescribedBy, error ? errorId : undefined].filter(Boolean).join(" ") || undefined;
  const fieldClassName = [styles.field, error ? styles.error : ""].filter(Boolean).join(" ");
  const inputClassName = [styles.input, className].filter(Boolean).join(" ");
  const control = (
    <span className={fieldClassName}>
      {leadingIcon ? <span className={styles.leadingIcon} aria-hidden="true">{leadingIcon}</span> : null}
      <input
        {...props}
        ref={ref}
        className={inputClassName}
        aria-describedby={describedBy}
        aria-invalid={error ? true : undefined}
      />
    </span>
  );

  if (!label) {
    return (
      <div className={styles.root}>
        {control}
        {error ? <span className={styles.errorMessage} id={errorId}>{error}</span> : null}
      </div>
    );
  }

  return (
    <label className={styles.root}>
      <span className={styles.label}>{label}</span>
      {control}
      {error ? <span className={styles.errorMessage} id={errorId}>{error}</span> : null}
    </label>
  );
});
