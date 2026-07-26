import {
  ArrowDown,
  ArrowUp,
  Briefcase,
  Certificate,
  Code,
  Eye,
  EyeSlash,
  FileText,
  Folder,
  GearSix,
  Globe,
  GraduationCap,
  MagicWand,
  Plus,
  TextAlignLeft,
  Translate,
  UserCircle,
} from "@phosphor-icons/react";
import { sectionCatalog } from "../data/sampleResume.js";

const icons = {
  basics: UserCircle,
  summary: TextAlignLeft,
  experience: Briefcase,
  projects: Folder,
  portfolio: Globe,
  education: GraduationCap,
  skills: Code,
  certifications: Certificate,
  languages: Translate,
};

export function SectionRail({ document, selected, activeView = "section", onSelect, onAdd, onMove, onToggle, onOpenReview, onOpenTemplates, onOpenSettings }) {
  const hidden = new Set(document.hidden_sections || []);
  const sections = (document.section_order || []).map((id) => sectionCatalog.find((section) => section.id === id)).filter(Boolean);
  return (
    <nav className="section-rail" aria-label="Resume sections">
      <div className="section-rail__list">
        {sections.map((section, index) => {
          const Icon = icons[section.id] || Folder;
          const isHidden = hidden.has(section.id);
          return (
            <div className={`section-nav-row ${selected === section.id ? "is-selected" : ""} ${isHidden ? "is-hidden" : ""}`} key={section.id}>
              <button className="section-nav-row__main" type="button" aria-label={section.label} title={section.label} onClick={() => onSelect(section.id)} aria-current={selected === section.id ? "page" : undefined}>
                <Icon size={20} weight="regular" />
                <span>{section.label}</span>
              </button>
              <div className="section-nav-row__tools">
                <button type="button" aria-label={`Move ${section.label} up`} title="Move up" disabled={index === 0} onClick={() => onMove(section.id, -1)}><ArrowUp size={14} /></button>
                <button type="button" aria-label={`Move ${section.label} down`} title="Move down" disabled={index === sections.length - 1} onClick={() => onMove(section.id, 1)}><ArrowDown size={14} /></button>
                <button type="button" aria-label={`${isHidden ? "Show" : "Hide"} ${section.label}`} title={isHidden ? "Show" : "Hide"} onClick={() => onToggle(section.id)}>{isHidden ? <EyeSlash size={15} /> : <Eye size={15} />}</button>
              </div>
            </div>
          );
        })}
      </div>
      <button type="button" className="section-rail__add" aria-label="Add section" title="Add section" onClick={onAdd}><Plus size={20} /><span>Add section</span></button>
      <div className="section-rail__utilities" aria-label="Resume tools">
        <button type="button" aria-label="AI review" title="AI review" className={activeView === "review" ? "is-selected" : ""} onClick={onOpenReview}><MagicWand size={20} /><span>AI review</span></button>
        <button type="button" aria-label="Templates" title="Templates" onClick={onOpenTemplates}><FileText size={20} /><span>Templates</span></button>
        <button type="button" aria-label="AI settings" title="AI settings" onClick={onOpenSettings}><GearSix size={20} /><span>AI settings</span></button>
      </div>
    </nav>
  );
}
