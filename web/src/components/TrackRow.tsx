type TrackRowProps = {
  title: string;
  subtitle: string;
  actionLabel: string;
  onAction: () => void;
  disabled?: boolean;
  selected?: boolean;
};

export function TrackRow({ title, subtitle, actionLabel, onAction, disabled = false, selected = false }: TrackRowProps) {
  return (
    <article className={`track-row${selected ? " track-row--selected" : ""}`}>
      <div className="track-row__body">
        <strong className="track-row__title">{title}</strong>
        <span className="track-row__subtitle">{subtitle}</span>
      </div>
      <IconButton
        type="button"
        className="track-row__action"
        aria-label={actionLabel}
        onClick={onAction}
        disabled={disabled}
      >
        −
      </IconButton>
    </article>
  );
}
import { IconButton } from "./IconButton";
