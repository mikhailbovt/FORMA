import { useState } from "react";
import {
  Camera,
  CaretDown,
  MagicWand,
  Plus,
  Trash,
  X,
} from "@phosphor-icons/react";
import { addArrayItem, removeArrayItem, uid, updateArrayItem } from "../lib/resume.js";
import { sectionCatalog } from "../data/sampleResume.js";
import { Button, Field } from "./ui.jsx";

const sectionCopy = {
  basics: "The essentials recruiters use to identify and contact you.",
  summary: "A focused introduction tailored to the role you want.",
  experience: "Roles, responsibilities, and outcomes you can defend.",
  projects: "Selected work that demonstrates relevant ability.",
  portfolio: "Links to shipped work, writing, talks, or case studies.",
  education: "Degrees, programs, and relevant academic details.",
  skills: "Group related skills so they remain easy to scan.",
  certifications: "Current credentials that support your candidacy.",
  languages: "Languages and honest proficiency levels.",
};

function splitComma(value) {
  return String(value || "").split(",").map((item) => item.trim()).filter(Boolean);
}

function splitLines(value) {
  return String(value || "").split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
}

function fileToDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result);
    reader.onerror = () => reject(new Error("Could not read this image."));
    reader.readAsDataURL(file);
  });
}

function loadImage(source) {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("This image format is not supported."));
    image.src = source;
  });
}

async function prepareProfilePhoto(file) {
  if (!file?.type?.match(/^image\/(jpeg|png|webp)$/)) throw new Error("Choose a JPG, PNG, or WebP image.");
  if (file.size > 10_000_000) throw new Error("Choose an image smaller than 10 MB.");
  const source = await fileToDataURL(file);
  const image = await loadImage(source);
  const size = Math.min(image.naturalWidth, image.naturalHeight);
  const sourceX = Math.max(0, (image.naturalWidth - size) / 2);
  const sourceY = Math.max(0, (image.naturalHeight - size) / 2);
  const canvas = window.document.createElement("canvas");
  canvas.width = 480;
  canvas.height = 480;
  const context = canvas.getContext("2d");
  if (!context) throw new Error("Your browser could not prepare this photo.");
  context.drawImage(image, sourceX, sourceY, size, size, 0, 0, 480, 480);
  return canvas.toDataURL("image/jpeg", 0.82);
}

function EditorHeader({ section, onOpenReview }) {
  const label = sectionCatalog.find((item) => item.id === section)?.label || "Section";
  return (
    <header className="form-panel__header">
      <div>
        <span className="form-panel__eyebrow">Resume content</span>
        <h1>{label}</h1>
        <p>{sectionCopy[section]}</p>
      </div>
      <button type="button" className="form-panel__ai-shortcut" onClick={onOpenReview}>
        <MagicWand size={16} /> AI review
      </button>
    </header>
  );
}

function ItemCard({ title, subtitle, initialOpen = false, onRemove, children }) {
  const [open, setOpen] = useState(initialOpen);
  return (
    <article className={`editor-item ${open ? "is-open" : ""}`}>
      <button className="editor-item__toggle" type="button" aria-expanded={open} onClick={() => setOpen(!open)}>
        <span><strong>{title || "Untitled item"}</strong>{subtitle && <small>{subtitle}</small>}</span>
        <CaretDown size={16} weight="bold" />
      </button>
      {open && <div className="editor-item__body">
        {children}
        <button className="editor-item__remove" type="button" onClick={onRemove}><Trash size={15} /> Remove item</button>
      </div>}
    </article>
  );
}

function AddItemButton({ label, onClick }) {
  return <button className="editor-add-item" type="button" onClick={onClick}><Plus size={17} />{label}</button>;
}

function EmptySection({ label, onAdd }) {
  return <div className="editor-section-empty"><p>No items yet.</p><Button variant="secondary" onClick={onAdd}><Plus size={16} />{label}</Button></div>;
}

function BasicsEditor({ resume, onChange }) {
  const [photoError, setPhotoError] = useState("");
  const [photoBusy, setPhotoBusy] = useState(false);
  const basics = resume.basics || {};
  const update = (field, value) => onChange({ ...resume, basics: { ...basics, [field]: value } });
  const updateProfiles = (profiles) => update("profiles", profiles);

  async function handlePhoto(event) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    setPhotoError("");
    setPhotoBusy(true);
    try { update("photo_url", await prepareProfilePhoto(file)); }
    catch (error) { setPhotoError(error.message); }
    finally { setPhotoBusy(false); }
  }

  return <div className="form-stack">
    <section className="form-card form-card--photo">
      <div className="photo-control">
        <div className="photo-control__preview">
          {basics.photo_url ? <img src={basics.photo_url} alt="Profile preview" /> : <Camera size={24} />}
        </div>
        <div>
          <strong>Profile photo <span>Optional</span></strong>
          <p>Templates place it differently. ATS Safe hides it automatically.</p>
          <div className="photo-control__actions">
            <label className="button button--secondary button--sm">{photoBusy ? "Preparing…" : basics.photo_url ? "Replace photo" : "Add photo"}<input type="file" accept="image/jpeg,image/png,image/webp" hidden disabled={photoBusy} onChange={handlePhoto} /></label>
            {basics.photo_url && <button type="button" onClick={() => update("photo_url", "")}><X size={14} />Remove</button>}
          </div>
          {photoError && <span className="photo-control__error" role="alert">{photoError}</span>}
        </div>
      </div>
    </section>
    <section className="form-card">
      <h2>Identity</h2>
      <div className="field-row"><Field label="Full name"><input value={basics.full_name || ""} onChange={(event) => update("full_name", event.target.value)} /></Field><Field label="Professional headline"><input value={basics.headline || ""} onChange={(event) => update("headline", event.target.value)} /></Field></div>
    </section>
    <section className="form-card">
      <h2>Contact details</h2>
      <div className="field-row"><Field label="Email"><input type="email" value={basics.email || ""} onChange={(event) => update("email", event.target.value)} /></Field><Field label="Phone"><input value={basics.phone || ""} onChange={(event) => update("phone", event.target.value)} /></Field></div>
      <div className="field-row"><Field label="Location"><input value={basics.location || ""} onChange={(event) => update("location", event.target.value)} /></Field><Field label="Website"><input value={basics.website || ""} placeholder="yourname.dev" onChange={(event) => update("website", event.target.value)} /></Field></div>
    </section>
    <section className="form-card">
      <div className="form-card__title"><div><h2>Profile links</h2><p>LinkedIn, GitHub, Behance, or another relevant profile.</p></div><button type="button" onClick={() => updateProfiles([...(basics.profiles || []), { id: uid("profile"), network: "LinkedIn", username: "", url: "" }])}><Plus size={16} />Add link</button></div>
      {(basics.profiles || []).map((profile) => <div className="profile-link-row" key={profile.id}>
        <Field label="Label"><input value={profile.network || ""} onChange={(event) => updateProfiles((basics.profiles || []).map((item) => item.id === profile.id ? { ...item, network: event.target.value } : item))} /></Field>
        <Field label="URL"><input value={profile.url || ""} placeholder="https://…" onChange={(event) => updateProfiles((basics.profiles || []).map((item) => item.id === profile.id ? { ...item, url: event.target.value } : item))} /></Field>
        <button type="button" aria-label={`Remove ${profile.network || "profile"}`} onClick={() => updateProfiles((basics.profiles || []).filter((item) => item.id !== profile.id))}><Trash size={16} /></button>
      </div>)}
    </section>
  </div>;
}

function SummaryEditor({ resume, onChange }) {
  const value = resume.summary || "";
  return <div className="form-stack"><section className="form-card"><Field label="Professional summary" hint="Aim for 2–4 sentences: who you are, what you do well, and the impact you create."><textarea rows="11" maxLength="800" value={value} onChange={(event) => onChange({ ...resume, summary: event.target.value })} /></Field><div className="field-counter">{value.length} / 800</div></section></div>;
}

function ExperienceEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "experience", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "experience", id));
  const add = () => onChange(addArrayItem(resume, "experience"));
  if (!(resume.experience || []).length) return <EmptySection label="Add experience" onAdd={add} />;
  return <div className="form-stack">{resume.experience.map((item, index) => <ItemCard key={item.id} title={item.position} subtitle={item.company} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <div className="field-row"><Field label="Role"><input value={item.position || ""} onChange={(event) => update(item.id, "position", event.target.value)} /></Field><Field label="Company"><input value={item.company || ""} onChange={(event) => update(item.id, "company", event.target.value)} /></Field></div>
    <div className="field-row"><Field label="Location"><input value={item.location || ""} onChange={(event) => update(item.id, "location", event.target.value)} /></Field><Field label="Start"><input type="month" value={item.start_date || ""} onChange={(event) => update(item.id, "start_date", event.target.value)} /></Field></div>
    <div className="field-row"><Field label="End"><input type="month" disabled={item.current} value={item.end_date || ""} onChange={(event) => update(item.id, "end_date", event.target.value)} /></Field><label className="checkbox-field checkbox-field--panel"><input type="checkbox" checked={Boolean(item.current)} onChange={(event) => update(item.id, "current", event.target.checked)} />Current role</label></div>
    <Field label="Role summary"><textarea rows="3" value={item.summary || ""} onChange={(event) => update(item.id, "summary", event.target.value)} /></Field>
    <Field label="Achievements" hint="One verified outcome per line"><textarea rows="6" value={(item.highlights || []).join("\n")} onChange={(event) => update(item.id, "highlights", splitLines(event.target.value))} /></Field>
  </ItemCard>)}<AddItemButton label="Add another role" onClick={add} /></div>;
}

function ProjectsEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "projects", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "projects", id));
  const add = () => onChange(addArrayItem(resume, "projects"));
  if (!(resume.projects || []).length) return <EmptySection label="Add project" onAdd={add} />;
  return <div className="form-stack">{resume.projects.map((item, index) => <ItemCard key={item.id} title={item.name} subtitle={item.role || item.url} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <div className="field-row"><Field label="Project name"><input value={item.name || ""} onChange={(event) => update(item.id, "name", event.target.value)} /></Field><Field label="Your role"><input value={item.role || ""} onChange={(event) => update(item.id, "role", event.target.value)} /></Field></div>
    <Field label="Project URL"><input value={item.url || ""} onChange={(event) => update(item.id, "url", event.target.value)} /></Field>
    <Field label="Description"><textarea rows="4" value={item.description || ""} onChange={(event) => update(item.id, "description", event.target.value)} /></Field>
    <Field label="Highlights" hint="One result per line"><textarea rows="4" value={(item.highlights || []).join("\n")} onChange={(event) => update(item.id, "highlights", splitLines(event.target.value))} /></Field>
    <Field label="Technologies" hint="Comma-separated"><input value={(item.technologies || []).join(", ")} onChange={(event) => update(item.id, "technologies", splitComma(event.target.value))} /></Field>
  </ItemCard>)}<AddItemButton label="Add another project" onClick={add} /></div>;
}

function PortfolioEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "portfolio", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "portfolio", id));
  const add = () => onChange(addArrayItem(resume, "portfolio"));
  if (!(resume.portfolio || []).length) return <EmptySection label="Add portfolio item" onAdd={add} />;
  return <div className="form-stack">{resume.portfolio.map((item, index) => <ItemCard key={item.id} title={item.name} subtitle={item.url} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <Field label="Title"><input value={item.name || ""} onChange={(event) => update(item.id, "name", event.target.value)} /></Field><Field label="URL"><input value={item.url || ""} onChange={(event) => update(item.id, "url", event.target.value)} /></Field><Field label="Description"><textarea rows="5" value={item.description || ""} onChange={(event) => update(item.id, "description", event.target.value)} /></Field>
  </ItemCard>)}<AddItemButton label="Add portfolio item" onClick={add} /></div>;
}

function EducationEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "education", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "education", id));
  const add = () => onChange(addArrayItem(resume, "education"));
  if (!(resume.education || []).length) return <EmptySection label="Add education" onAdd={add} />;
  return <div className="form-stack">{resume.education.map((item, index) => <ItemCard key={item.id} title={item.institution} subtitle={[item.degree, item.area].filter(Boolean).join(" · ")} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <Field label="Institution"><input value={item.institution || ""} onChange={(event) => update(item.id, "institution", event.target.value)} /></Field>
    <div className="field-row"><Field label="Degree"><input value={item.degree || ""} onChange={(event) => update(item.id, "degree", event.target.value)} /></Field><Field label="Field of study"><input value={item.area || ""} onChange={(event) => update(item.id, "area", event.target.value)} /></Field></div>
    <div className="field-row"><Field label="Location"><input value={item.location || ""} onChange={(event) => update(item.id, "location", event.target.value)} /></Field><Field label="Graduation"><input value={item.end_date || ""} onChange={(event) => update(item.id, "end_date", event.target.value)} /></Field></div>
  </ItemCard>)}<AddItemButton label="Add education" onClick={add} /></div>;
}

function SkillsEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "skills", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "skills", id));
  const add = () => onChange(addArrayItem(resume, "skills"));
  if (!(resume.skills || []).length) return <EmptySection label="Add skill group" onAdd={add} />;
  return <div className="form-stack">{resume.skills.map((item, index) => <ItemCard key={item.id} title={item.name} subtitle={(item.items || []).slice(0, 3).join(", ")} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <Field label="Group name"><input value={item.name || ""} onChange={(event) => update(item.id, "name", event.target.value)} /></Field><Field label="Skills" hint="Comma-separated"><textarea rows="4" value={(item.items || []).join(", ")} onChange={(event) => update(item.id, "items", splitComma(event.target.value))} /></Field>
  </ItemCard>)}<AddItemButton label="Add skill group" onClick={add} /></div>;
}

function CertificationsEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "certifications", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "certifications", id));
  const add = () => onChange(addArrayItem(resume, "certifications"));
  if (!(resume.certifications || []).length) return <EmptySection label="Add certification" onAdd={add} />;
  return <div className="form-stack">{resume.certifications.map((item, index) => <ItemCard key={item.id} title={item.name} subtitle={item.issuer} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <div className="field-row"><Field label="Certification"><input value={item.name || ""} onChange={(event) => update(item.id, "name", event.target.value)} /></Field><Field label="Issuer"><input value={item.issuer || ""} onChange={(event) => update(item.id, "issuer", event.target.value)} /></Field></div><div className="field-row"><Field label="Date"><input value={item.date || ""} onChange={(event) => update(item.id, "date", event.target.value)} /></Field><Field label="URL"><input value={item.url || ""} onChange={(event) => update(item.id, "url", event.target.value)} /></Field></div>
  </ItemCard>)}<AddItemButton label="Add certification" onClick={add} /></div>;
}

function LanguagesEditor({ resume, onChange }) {
  const update = (id, field, value) => onChange(updateArrayItem(resume, "languages", id, (item) => ({ ...item, [field]: value })));
  const remove = (id) => onChange(removeArrayItem(resume, "languages", id));
  const add = () => onChange(addArrayItem(resume, "languages"));
  if (!(resume.languages || []).length) return <EmptySection label="Add language" onAdd={add} />;
  return <div className="form-stack">{resume.languages.map((item, index) => <ItemCard key={item.id} title={item.language} subtitle={item.fluency} initialOpen={index === 0} onRemove={() => remove(item.id)}>
    <div className="field-row"><Field label="Language"><input value={item.language || ""} onChange={(event) => update(item.id, "language", event.target.value)} /></Field><Field label="Proficiency"><input value={item.fluency || ""} placeholder="Native, C1, B2…" onChange={(event) => update(item.id, "fluency", event.target.value)} /></Field></div>
  </ItemCard>)}<AddItemButton label="Add language" onClick={add} /></div>;
}

const editors = {
  basics: BasicsEditor,
  summary: SummaryEditor,
  experience: ExperienceEditor,
  projects: ProjectsEditor,
  portfolio: PortfolioEditor,
  education: EducationEditor,
  skills: SkillsEditor,
  certifications: CertificationsEditor,
  languages: LanguagesEditor,
};

export function SectionEditorPanel({ section, document: resume, onChange, onOpenReview }) {
  const EditorComponent = editors[section] || SummaryEditor;
  return <div className="form-panel" data-editor-section={section}>
    <EditorHeader section={section} onOpenReview={onOpenReview} />
    <div className="form-panel__content"><EditorComponent resume={resume} onChange={onChange} /></div>
  </div>;
}
