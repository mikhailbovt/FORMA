import { useEffect, useRef } from "react";
import { X } from "@phosphor-icons/react";

export function Button({ variant = "primary", size = "md", className = "", type = "button", ...props }) {
  return <button type={type} className={`button button--${variant} button--${size} ${className}`} {...props} />;
}

export function IconButton({ label, className = "", children, ...props }) {
  return (
    <button type="button" className={`icon-button ${className}`} aria-label={label} title={label} {...props}>
      {children}
    </button>
  );
}

export function InlineEdit({ value = "", onCommit, multiline = false, className = "", label, placeholder = "Click to edit", editable = true }) {
  const ref = useRef(null);

  useEffect(() => {
    if (ref.current && document.activeElement !== ref.current && ref.current.innerText !== String(value || "")) {
      ref.current.innerText = String(value || "");
    }
  }, [value]);

  function commit() {
    const next = ref.current?.innerText.replace(/\n{3,}/g, "\n\n").trim() || "";
    if (next !== String(value || "")) onCommit(next);
  }

  return (
    <span
      ref={ref}
      className={`inline-edit ${multiline ? "inline-edit--multiline" : ""} ${className}`}
      contentEditable={editable}
      suppressContentEditableWarning
      role={editable ? "textbox" : undefined}
      aria-label={label}
      aria-multiline={multiline || undefined}
      data-placeholder={placeholder}
      onBlur={editable ? commit : undefined}
      onKeyDown={(event) => {
        if (!editable) return;
        if (event.key === "Escape") {
          ref.current.innerText = String(value || "");
          ref.current.blur();
        }
        if (!multiline && event.key === "Enter") {
          event.preventDefault();
          ref.current.blur();
        }
      }}
    >
      {value}
    </span>
  );
}

export function Dialog({ open, title, description, onClose, children, className = "" }) {
  useEffect(() => {
    if (!open) return undefined;
    const onKeyDown = (event) => event.key === "Escape" && onClose();
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="dialog-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className={`dialog ${className}`} role="dialog" aria-modal="true" aria-labelledby="dialog-title">
        <div className="dialog__header">
          <div>
            <h2 id="dialog-title">{title}</h2>
            {description && <p>{description}</p>}
          </div>
          <IconButton label="Close" onClick={onClose}><X size={20} /></IconButton>
        </div>
        <div className="dialog__body">{children}</div>
      </section>
    </div>
  );
}

export function Field({ label, hint, children, className = "" }) {
  return (
    <label className={`field ${className}`}>
      <span className="field__label">{label}</span>
      {children}
      {hint && <span className="field__hint">{hint}</span>}
    </label>
  );
}

export function EmptyState({ icon, title, description, action }) {
  return (
    <div className="empty-state">
      {icon && <div className="empty-state__icon">{icon}</div>}
      <h2>{title}</h2>
      <p>{description}</p>
      {action}
    </div>
  );
}

export function Spinner({ label = "Loading" }) {
  return <span className="spinner" role="status" aria-label={label} />;
}

export function Toast({ message, tone = "neutral", onDismiss }) {
  useEffect(() => {
    if (!message) return undefined;
    const timer = window.setTimeout(onDismiss, 3200);
    return () => window.clearTimeout(timer);
  }, [message, onDismiss]);
  if (!message) return null;
  return <div className={`toast toast--${tone}`} role="status">{message}</div>;
}
