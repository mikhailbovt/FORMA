import { Check, UserCircle } from "@phosphor-icons/react";
import { Button, Dialog } from "./ui.jsx";

export const templates = [
  { id: "editorial", name: "Editorial", description: "Refined serif identity with a balanced two-column rhythm.", detail: "Distinctive · Balanced" },
  { id: "ats", name: "ATS Safe", description: "Conservative single-column structure for reliable parsing.", detail: "Recommended · One column" },
  { id: "compact", name: "Compact", description: "Tighter spacing for experienced candidates with more content.", detail: "Dense · Content-first" },
  { id: "modern", name: "Modern", description: "Confident sans-serif typography and a crisp visual hierarchy.", detail: "Clean · Contemporary" },
  { id: "classic", name: "Classic", description: "Traditional centered header with an understated serif voice.", detail: "Formal · Timeless" },
  { id: "portrait", name: "Portrait", description: "A polished header designed to feature an optional photo.", detail: "Personal · Photo-ready" },
  { id: "minimal", name: "Minimal", description: "Extra breathing room with quiet typography and fine rules.", detail: "Calm · Spacious" },
];

function TemplateMiniature({ template }) {
  return <div className={`template-miniature template-miniature--${template.id}`} aria-hidden="true">
    {template.id === "portrait" && <UserCircle className="template-miniature__photo" size={34} weight="thin" />}
    <div className="template-miniature__header"><strong>ALEX MORGAN</strong><span>PRODUCT ENGINEER</span><small>alexmorgan.dev · Austin, TX</small></div>
    <div className="template-miniature__section"><b>EXPERIENCE</b><span>Product Engineer</span><small>Shipped measurable product improvements.</small><small>Built reliable systems for growing teams.</small></div>
    <div className="template-miniature__section"><b>SKILLS</b><small>Product · Go · TypeScript · Analytics</small></div>
  </div>;
}

export function TemplatesDialog({ open, value, paperSize, onClose, onChange }) {
  return (
    <Dialog open={open} onClose={onClose} title="Choose a template" description="Templates change presentation only. Your content stays untouched." className="dialog--templates">
      <div className="template-grid">
        {templates.map((template) => (
          <button type="button" className={`template-card ${value === template.id ? "is-selected" : ""}`} key={template.id} onClick={() => onChange({ template: template.id, paper_size: paperSize })}>
            <TemplateMiniature template={template} />
            <div className="template-card__copy"><strong>{template.name}</strong><span>{template.description}</span><small>{template.detail}</small></div>
            {value === template.id && <Check className="template-card__check" weight="bold" size={18} />}
          </button>
        ))}
      </div>
      <div className="paper-picker">
        <div><strong>Paper size</strong><span>Switch without changing your content.</span></div>
        <div className="segmented-control" role="group" aria-label="Paper size">
          {[["a4", "A4"], ["letter", "US Letter"]].map(([id, label]) => <button type="button" key={id} className={paperSize === id ? "is-selected" : ""} onClick={() => onChange({ template: value, paper_size: id })}>{label}</button>)}
        </div>
      </div>
      <div className="dialog__actions"><Button onClick={onClose}>Done</Button></div>
    </Dialog>
  );
}
