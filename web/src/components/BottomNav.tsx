import type { ReactNode } from "react";
import { useState } from "react";
import styles from "./BottomNav.module.css";

type AppTab = "home" | "search" | "playlists";

type BottomNavProps = {
  activeTab: AppTab;
  onChange: (tab: AppTab) => void;
};

type NavItem = {
  id: AppTab;
  label: string;
  icon: ReactNode;
};

const items: NavItem[] = [
  {
    id: "home",
    label: "Home",
    icon: (
      <svg viewBox="0 0 24 24" role="presentation" focusable="false">
        <path d="M4.5 10.5 12 4l7.5 6.5v8a1 1 0 0 1-1 1h-4.5v-5h-4v5H5.5a1 1 0 0 1-1-1z" />
      </svg>
    )
  },
  {
    id: "search",
    label: "Search",
    icon: (
      <svg viewBox="0 0 24 24" role="presentation" focusable="false">
        <circle cx="11" cy="11" r="5.5" />
        <path d="m16 16 4 4" />
      </svg>
    )
  },
  {
    id: "playlists",
    label: "Playlists",
    icon: (
      <svg viewBox="0 0 24 24" role="presentation" focusable="false">
        <path d="M8 6.5h8.5" />
        <path d="M8 11.5h8.5" />
        <path d="M8 16.5H13" />
        <circle cx="6" cy="6.5" r="1.25" />
        <circle cx="6" cy="11.5" r="1.25" />
        <circle cx="6" cy="16.5" r="1.25" />
      </svg>
    )
  }
];

export function BottomNav({ activeTab, onChange }: BottomNavProps) {
  const [isRailExpanded, setIsRailExpanded] = useState(false);

  return (
    <nav
      className={`${styles.root}${isRailExpanded ? ` ${styles.expanded}` : ""}`}
      aria-label="Navegação principal"
      data-rail-state={isRailExpanded ? "expanded" : "collapsed"}
    >
      <button
        type="button"
        className={styles.railToggle}
        aria-label={isRailExpanded ? "Recolher navegação" : "Expandir navegação"}
        aria-expanded={isRailExpanded}
        onClick={() => setIsRailExpanded((current) => !current)}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
          <path d={isRailExpanded ? "m14.5 5-7 7 7 7" : "m9.5 5 7 7-7 7"} />
        </svg>
      </button>

      {items.map((item) => {
        const selected = item.id === activeTab;

        return (
          <button
            key={item.id}
            type="button"
            className={`${styles.item}${selected ? ` ${styles.active}` : ""}`}
            aria-current={selected ? "page" : undefined}
            onClick={() => onChange(item.id)}
          >
            <span className={styles.icon} aria-hidden="true">
              {item.icon}
            </span>
            <span className={styles.label}>{item.label}</span>
          </button>
        );
      })}
    </nav>
  );
}
