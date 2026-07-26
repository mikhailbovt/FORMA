import {
  ChartBar,
  DotsSixVertical,
  EnvelopeSimple,
  LinkSimple,
  MagicWand,
  MapPin,
  Phone,
  Plus,
  TextAlignLeft,
} from "@phosphor-icons/react";
import { Fragment } from "react";
import { InlineEdit } from "./ui.jsx";
import { formatPeriod, updateArrayItem } from "../lib/resume.js";

function Contact({ icon, value, href }) {
  if (!value) return null;
  const content = <><span className="resume-contact__icon">{icon}</span><span>{value}</span></>;
  return href ? <a className="resume-contact" href={href} target="_blank" rel="noreferrer">{content}</a> : <span className="resume-contact">{content}</span>;
}

function Section({ id, label, selected, previewMode, selectable, onSelect, children, className = "" }) {
  return (
    <section
      className={`resume-section resume-section--${id} ${selected && !previewMode ? "is-selected" : ""} ${selected && previewMode ? "is-preview-target" : ""} ${className}`}
      onClick={(event) => {
        if (!previewMode || selectable) {
          event.stopPropagation();
          onSelect(id);
        }
      }}
      data-section={id}
    >
      {selected && !previewMode && <DotsSixVertical className="resume-section__drag" size={17} weight="bold" aria-hidden="true" />}
      {label && <h2 className="resume-section__title">{label}</h2>}
      {children}
    </section>
  );
}

export function ResumeDocument({
  document: resume,
  selectedSection,
  onSelectSection,
  onChange,
  onRewrite,
  onShorten,
  onAddMetric,
  onAddItem,
  previewMode = false,
  selectable = false,
  activeBullet,
  onActiveBullet,
}) {
  const locale = resume.language || "en";
  const hidden = new Set(resume.hidden_sections || []);

  const updateBasics = (field, value) => onChange({ ...resume, basics: { ...resume.basics, [field]: value } });
  const updateItem = (section, id, field, value) => onChange(updateArrayItem(resume, section, id, (item) => ({ ...item, [field]: value })));
  const updateHighlight = (section, id, index, value) => onChange(updateArrayItem(resume, section, id, (item) => {
    const highlights = [...(item.highlights || [])];
    highlights[index] = value;
    return { ...item, highlights };
  }));

  const renderers = {
    basics: () => (
      <Section id="basics" selected={selectedSection === "basics"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <div className={`resume-header ${resume.basics.photo_url ? "resume-header--photo" : ""}`}>
          <div className="resume-header__identity">
            <InlineEdit editable={!previewMode} className="resume-name" value={resume.basics.full_name} label="Full name" onCommit={(value) => updateBasics("full_name", value)} />
            <InlineEdit editable={!previewMode} className="resume-headline" value={resume.basics.headline} label="Professional headline" onCommit={(value) => updateBasics("headline", value)} />
          </div>
          <div className="resume-contacts" aria-label="Contact information">
            <Contact icon={<EnvelopeSimple size={13} />} value={resume.basics.email} href={resume.basics.email ? `mailto:${resume.basics.email}` : undefined} />
            <Contact icon={<Phone size={13} />} value={resume.basics.phone} />
            <Contact icon={<MapPin size={13} />} value={resume.basics.location} />
            <Contact icon={<LinkSimple size={13} />} value={resume.basics.website} href={resume.basics.website ? `https://${resume.basics.website.replace(/^https?:\/\//, "")}` : undefined} />
            {(resume.basics.profiles || []).slice(0, 2).map((profile) => <Contact key={profile.id || profile.url} icon={<LinkSimple size={13} />} value={profile.network || profile.url} href={profile.url} />)}
          </div>
          {resume.basics.photo_url && <img className="resume-photo" src={resume.basics.photo_url} alt={`${resume.basics.full_name || "Candidate"} profile`} />}
        </div>
      </Section>
    ),
    summary: () => (
      <Section id="summary" selected={selectedSection === "summary"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <InlineEdit editable={!previewMode} multiline className="resume-summary" value={resume.summary} label="Professional summary" onCommit={(value) => onChange({ ...resume, summary: value })} />
      </Section>
    ),
    experience: () => (
      <Section id="experience" label="Experience" selected={selectedSection === "experience"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <div className="resume-stack">
          {(resume.experience || []).map((item) => (
            <article className="resume-entry resume-entry--experience" key={item.id}>
              <div className="resume-entry__meta">
                <InlineEdit editable={!previewMode} className="resume-entry__company" value={item.company} label="Company" onCommit={(value) => updateItem("experience", item.id, "company", value)} />
                <InlineEdit editable={!previewMode} value={item.location} label="Location" onCommit={(value) => updateItem("experience", item.id, "location", value)} />
                <span>{formatPeriod(item.start_date, item.end_date, item.current, locale)}</span>
              </div>
              <div className="resume-entry__body">
                <InlineEdit editable={!previewMode} className="resume-entry__role" value={item.position} label="Role" onCommit={(value) => updateItem("experience", item.id, "position", value)} />
                <InlineEdit editable={!previewMode} multiline className="resume-entry__summary" value={item.summary} label="Role summary" onCommit={(value) => updateItem("experience", item.id, "summary", value)} />
                <div className={`resume-entry__highlights ${selectedSection === "experience" && activeBullet?.itemId === item.id && !previewMode ? "is-selected-highlights" : ""}`}>
                  {selectedSection === "experience" && activeBullet?.itemId === item.id && !previewMode && <DotsSixVertical className="resume-entry__drag" size={17} weight="bold" aria-hidden="true" />}
                  <ul>
                    {(item.highlights || []).map((highlight, index) => {
                      const isActive = activeBullet?.itemId === item.id && activeBullet?.index === index;
                      return (
                        <li className={isActive ? "is-active" : ""} key={`${item.id}-${index}`} onClick={(event) => {
                          event.stopPropagation();
                          onSelectSection("experience");
                          onActiveBullet({ itemId: item.id, index, text: highlight });
                        }}>
                          <InlineEdit editable={!previewMode} multiline value={highlight} label={`Achievement ${index + 1}`} onCommit={(value) => updateHighlight("experience", item.id, index, value)} />
                        </li>
                      );
                    })}
                  </ul>
                  {selectedSection === "experience" && !previewMode && activeBullet?.itemId === item.id && (
                    <div className="context-toolbar" role="toolbar" aria-label="AI writing actions">
                      <button type="button" onClick={(event) => { event.stopPropagation(); onRewrite(); }}><MagicWand size={15} />Rewrite</button>
                      <button type="button" onClick={(event) => { event.stopPropagation(); onShorten(); }}><TextAlignLeft size={15} />Shorten</button>
                      <button type="button" onClick={(event) => { event.stopPropagation(); onAddMetric(); }}><ChartBar size={15} />Add metric</button>
                    </div>
                  )}
                </div>
              </div>
            </article>
          ))}
          {!previewMode && selectedSection === "experience" && <button type="button" className="resume-add-row" onClick={(event) => { event.stopPropagation(); onAddItem("experience"); }}><Plus size={14} /> Add role</button>}
        </div>
      </Section>
    ),
    projects: () => (
      <Section id="projects" label="Projects" selected={selectedSection === "projects"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <div className="resume-stack">
          {(resume.projects || []).map((item) => (
            <article className="resume-entry resume-entry--simple" key={item.id}>
              <div className="resume-entry__body">
                <div className="resume-entry__role-line">
                  <InlineEdit editable={!previewMode} className="resume-entry__role" value={item.name} label="Project name" onCommit={(value) => updateItem("projects", item.id, "name", value)} />
                  <span>{formatPeriod(item.start_date, item.end_date, false, locale)}</span>
                </div>
                <InlineEdit editable={!previewMode} multiline className="resume-entry__summary" value={item.description} label="Project description" onCommit={(value) => updateItem("projects", item.id, "description", value)} />
              </div>
            </article>
          ))}
          {!previewMode && selectedSection === "projects" && <button type="button" className="resume-add-row" onClick={(event) => { event.stopPropagation(); onAddItem("projects"); }}><Plus size={14} /> Add project</button>}
        </div>
      </Section>
    ),
    portfolio: () => (
      <Section id="portfolio" label="Portfolio" selected={selectedSection === "portfolio"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <div className="resume-stack">
          {(resume.portfolio || []).map((item) => (
            <article className="resume-entry resume-entry--simple" key={item.id}>
              <div className="resume-entry__body">
                <div className="resume-entry__role-line">
                  <InlineEdit editable={!previewMode} className="resume-entry__role" value={item.name} label="Portfolio item" onCommit={(value) => updateItem("portfolio", item.id, "name", value)} />
                  {item.url && <a className="resume-entry__url" href={item.url} target="_blank" rel="noreferrer">View work</a>}
                </div>
                <InlineEdit editable={!previewMode} multiline className="resume-entry__summary" value={item.description} label="Portfolio description" onCommit={(value) => updateItem("portfolio", item.id, "description", value)} />
              </div>
            </article>
          ))}
          {!previewMode && selectedSection === "portfolio" && <button type="button" className="resume-add-row" onClick={(event) => { event.stopPropagation(); onAddItem("portfolio"); }}><Plus size={14} /> Add portfolio item</button>}
        </div>
      </Section>
    ),
    education: () => (
      <Section id="education" label="Education" selected={selectedSection === "education"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <div className="resume-stack">
          {(resume.education || []).map((item) => (
            <article className="resume-entry resume-entry--education" key={item.id}>
              <div>
                <InlineEdit editable={!previewMode} className="resume-entry__company" value={item.institution} label="Institution" onCommit={(value) => updateItem("education", item.id, "institution", value)} />
                <span>{item.location}</span>
              </div>
              <div>
                <InlineEdit editable={!previewMode} className="resume-entry__role" value={[item.degree, item.area].filter(Boolean).join(" · ")} label="Degree" onCommit={(value) => updateItem("education", item.id, "area", value)} />
                <span>{formatPeriod(item.start_date, item.end_date, false, locale)}</span>
              </div>
            </article>
          ))}
          {!previewMode && selectedSection === "education" && <button type="button" className="resume-add-row" onClick={(event) => { event.stopPropagation(); onAddItem("education"); }}><Plus size={14} /> Add education</button>}
        </div>
      </Section>
    ),
    skills: () => (
      <Section id="skills" label="Skills" selected={selectedSection === "skills"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <div className="resume-skills">
          {(resume.skills || []).map((group) => (
            <p key={group.id}><InlineEdit editable={!previewMode} className="resume-skill-name" value={group.name} label="Skill group" onCommit={(value) => updateItem("skills", group.id, "name", value)} /><span>{(group.items || []).join(", ")}</span></p>
          ))}
        </div>
      </Section>
    ),
    certifications: () => (
      <Section id="certifications" label="Certifications" selected={selectedSection === "certifications"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        {(resume.certifications || []).map((item) => <p className="resume-inline-row" key={item.id}><strong>{item.name}</strong><span>{item.issuer}</span><span>{item.date}</span></p>)}
      </Section>
    ),
    languages: () => (
      <Section id="languages" label="Languages" selected={selectedSection === "languages"} previewMode={previewMode} selectable={selectable} onSelect={onSelectSection}>
        <p className="resume-inline-list">{(resume.languages || []).map((item) => `${item.language} — ${item.fluency}`).join(" · ")}</p>
      </Section>
    ),
  };

  return (
    <article className={`resume-paper resume-paper--${resume.template || "editorial"} resume-paper--${resume.paper_size || "a4"}`} onClick={() => !previewMode && onSelectSection(null)}>
      {(resume.section_order || []).filter((id) => !hidden.has(id)).map((id) => renderers[id] ? <Fragment key={id}>{renderers[id]()}</Fragment> : null)}
    </article>
  );
}
