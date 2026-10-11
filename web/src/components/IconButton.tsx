import type { ButtonHTMLAttributes, ReactNode } from "react";
import styles from "./IconButton.module.css";

type IconButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> & {
  children: ReactNode;
  selected?: boolean;
  "aria-label": string;
};

export function IconButton({ children, className, selected = false, ...props }: IconButtonProps) {
  const buttonClassName = [styles.button, selected ? styles.selected : "", className].filter(Boolean).join(" ");

  return <button {...props} className={buttonClassName} aria-pressed={selected || undefined}>{children}</button>;
}
