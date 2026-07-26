import {
  AlignmentType,
  BorderStyle,
  Document,
  ExternalHyperlink,
  HeadingLevel,
  ImageRun,
  Packer,
  Paragraph,
  Tab,
  Table,
  TableCell,
  TableLayoutType,
  TableRow,
  TextRun,
  VerticalAlign,
  WidthType,
} from "docx";
import { fileSafeName } from "./resume.js";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
const PAGE_SIZES = {
  a4: { width: 11_906, height: 16_838 },
  letter: { width: 12_240, height: 15_840 },
};
const PAGE_MARGIN = 936; // 0.65 in, in twentieths of a point (twips).
const INK = "111111";
const MUTED = "555555";
const HAIRLINE = "B8B8B2";
const LINK = "1F4B6E";
const SECTION_LABELS = {
  summary: "Professional summary",
  experience: "Experience",
  projects: "Projects",
  portfolio: "Portfolio",
  education: "Education",
  skills: "Skills",
  certifications: "Certifications",
  languages: "Languages",
};
const DEFAULT_ORDER = [
  "basics",
  "summary",
  "experience",
  "projects",
  "portfolio",
  "education",
  "skills",
  "certifications",
  "languages",
];

const noneBorder = { style: BorderStyle.NONE, size: 0, color: "FFFFFF" };
const noBorders = {
  top: noneBorder,
  bottom: noneBorder,
  left: noneBorder,
  right: noneBorder,
  insideHorizontal: noneBorder,
  insideVertical: noneBorder,
};

function cleanText(value) {
  return String(value ?? "")
    .replace(/\u00a0/g, " ")
    .replace(/[\u2010-\u2015\u2212]/g, "-")
    .trim();
}

function nonEmpty(values) {
  return values.map(cleanText).filter(Boolean);
}

function formatDate(value, locale) {
  const [year, month] = cleanText(value).split("-");
  if (!year) return "";
  if (!month) return year;
  const date = new Date(Number(year), Number(month) - 1, 1);
  if (Number.isNaN(date.getTime())) return cleanText(value);
  try {
    return new Intl.DateTimeFormat(locale || "en", { month: "short", year: "numeric" }).format(date);
  } catch {
    return new Intl.DateTimeFormat("en", { month: "short", year: "numeric" }).format(date);
  }
}

export function formatDOCXPeriod(startDate, endDate, current = false, locale = "en") {
  return nonEmpty([
    formatDate(startDate, locale),
    current ? "Present" : formatDate(endDate, locale),
  ]).join(" - ");
}

function safeURL(value, kind = "web") {
  const candidate = cleanText(value);
  if (!candidate) return "";
  if (kind === "email") return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(candidate) ? `mailto:${candidate}` : "";
  if (/^[a-z][a-z0-9+.-]*:/i.test(candidate) && !/^https?:\/\//i.test(candidate)) return "";
  const normalized = /^https?:\/\//i.test(candidate) ? candidate : `https://${candidate}`;
  try {
    const parsed = new URL(normalized);
    return ["http:", "https:"].includes(parsed.protocol) ? parsed.href : "";
  } catch {
    return "";
  }
}

function linkRun(label, target) {
  const display = cleanText(label);
  const href = safeURL(target);
  if (!display || !href) return display ? new TextRun(display) : null;
  return new ExternalHyperlink({
    link: href,
    children: [new TextRun({ text: display, color: LINK, underline: {} })],
  });
}

function contactChildren(basics) {
  const entries = [];
  const add = (label, target = "", kind = "web") => {
    const display = cleanText(label);
    if (!display) return;
    const href = target ? safeURL(target, kind) : "";
    const child = href
      ? new ExternalHyperlink({ link: href, children: [new TextRun({ text: display, color: LINK, underline: {} })] })
      : new TextRun(display);
    if (entries.length) entries.push(new TextRun({ text: " | ", color: MUTED }));
    entries.push(child);
  };

  add(basics.email, basics.email, "email");
  add(basics.phone);
  add(basics.location);
  add(basics.website, basics.website);
  for (const profile of (basics.profiles || []).slice(0, 3)) {
    const label = cleanText(profile.network) || cleanText(profile.username) || cleanText(profile.url);
    add(label, profile.url);
  }
  return entries;
}

function decodePhoto(value) {
  const candidate = cleanText(value);
  if (!candidate || candidate.length > 1_000_000) return null;
  const match = /^data:image\/(png|jpe?g);base64,([A-Za-z0-9+/=\s]+)$/i.exec(candidate);
  if (!match) return null;
  try {
    const binary = globalThis.atob(match[2].replace(/\s/g, ""));
    const data = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    const type = /^png$/i.test(match[1]) ? "png" : "jpg";
    const valid = type === "png"
      ? data.length >= 8 && data[0] === 0x89 && data[1] === 0x50 && data[2] === 0x4e && data[3] === 0x47
      : data.length >= 3 && data[0] === 0xff && data[1] === 0xd8 && data[2] === 0xff;
    return valid ? { data, type } : null;
  } catch {
    return null;
  }
}

function titleParagraph(children, date = "", contentWidth = 9_000) {
  const runs = Array.isArray(children) ? children.filter(Boolean) : [children].filter(Boolean);
  if (cleanText(date)) {
    runs.push(new TextRun({ children: [new Tab()] }));
    runs.push(new TextRun({ text: cleanText(date), color: MUTED, size: 18 }));
  }
  return new Paragraph({
    children: runs,
    keepNext: true,
    keepLines: true,
    spacing: { before: 80, after: 35 },
    tabStops: cleanText(date) ? [{ type: "right", position: Math.min(contentWidth, 9_250) }] : undefined,
  });
}

function metadataParagraph(parts, keepNext = false) {
  const value = nonEmpty(parts).join(" | ");
  if (!value) return null;
  return new Paragraph({
    children: [new TextRun({ text: value, color: MUTED, size: 18 })],
    keepNext,
    keepLines: true,
    spacing: { after: 45 },
  });
}

function bodyParagraph(value, options = {}) {
  const content = cleanText(value);
  if (!content) return null;
  return new Paragraph({
    children: [new TextRun(content)],
    keepNext: Boolean(options.keepNext),
    keepLines: true,
    spacing: { after: options.after ?? 55 },
  });
}

function bulletParagraph(value) {
  const content = cleanText(value);
  if (!content) return null;
  return new Paragraph({
    text: content,
    bullet: { level: 0 },
    keepLines: true,
    spacing: { after: 28 },
    indent: { left: 330, hanging: 170 },
  });
}

function sectionHeading(label) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_1,
    children: [new TextRun({ text: cleanText(label), bold: true, allCaps: true, size: 20, characterSpacing: 32 })],
    keepNext: true,
    spacing: { before: 190, after: 85 },
    border: { bottom: { style: BorderStyle.SINGLE, size: 6, color: INK, space: 3 } },
  });
}

function entrySpacer() {
  return new Paragraph({
    children: [],
    spacing: { after: 25 },
    border: { bottom: { style: BorderStyle.SINGLE, size: 2, color: HAIRLINE, space: 2 } },
  });
}

function headerChildren(resume, contentWidth) {
  const basics = resume.basics || {};
  const identity = [
    new Paragraph({
      heading: HeadingLevel.TITLE,
      children: [new TextRun({ text: cleanText(basics.full_name) || "Resume", bold: true, size: 52 })],
      keepNext: true,
      spacing: { after: 45 },
    }),
  ];
  if (cleanText(basics.headline)) {
    identity.push(new Paragraph({
      children: [new TextRun({ text: cleanText(basics.headline), bold: true, allCaps: true, size: 22, characterSpacing: 48 })],
      keepNext: true,
      spacing: { after: 80 },
    }));
  }
  const contacts = contactChildren(basics);
  if (contacts.length) {
    identity.push(new Paragraph({
      children: contacts,
      keepLines: true,
      spacing: { after: 80 },
    }));
  }

  const photo = resume.template === "ats" ? null : decodePhoto(basics.photo_url);
  if (!photo) return identity;

  const photoWidth = 1_260;
  return [new Table({
    width: { size: contentWidth, type: WidthType.DXA },
    columnWidths: [contentWidth - photoWidth, photoWidth],
    layout: TableLayoutType.FIXED,
    borders: noBorders,
    rows: [new TableRow({
      cantSplit: true,
      children: [
        new TableCell({
          width: { size: contentWidth - photoWidth, type: WidthType.DXA },
          verticalAlign: VerticalAlign.CENTER,
          borders: noBorders,
          margins: { top: 0, bottom: 0, left: 0, right: 120 },
          children: identity,
        }),
        new TableCell({
          width: { size: photoWidth, type: WidthType.DXA },
          verticalAlign: VerticalAlign.CENTER,
          borders: noBorders,
          margins: { top: 0, bottom: 0, left: 0, right: 0 },
          children: [new Paragraph({
            alignment: AlignmentType.RIGHT,
            children: [new ImageRun({
              type: photo.type,
              data: photo.data,
              transformation: { width: 72, height: 72 },
              altText: {
                name: "Profile photo",
                title: `${cleanText(basics.full_name) || "Candidate"} profile photo`,
                description: "Optional resume profile photo",
              },
            })],
          })],
        }),
      ],
    })],
  })];
}

function renderExperience(items, resume, contentWidth) {
  const children = [];
  const filtered = (items || []).filter((item) => nonEmpty([item.position, item.company, item.summary, ...(item.highlights || [])]).length);
  filtered.forEach((item, index) => {
    const period = formatDOCXPeriod(item.start_date, item.end_date, item.current, resume.language);
    children.push(titleParagraph([
      cleanText(item.position) ? new TextRun({ text: cleanText(item.position), bold: true, size: 22 }) : null,
      cleanText(item.company) ? new TextRun({ text: `${cleanText(item.position) ? " | " : ""}${cleanText(item.company)}`, size: 21 }) : null,
    ], period, contentWidth));
    const meta = metadataParagraph([item.location, (item.skills || []).join(", ")], Boolean(cleanText(item.summary) || (item.highlights || []).some(cleanText)));
    if (meta) children.push(meta);
    const summary = bodyParagraph(item.summary, { keepNext: (item.highlights || []).some(cleanText) });
    if (summary) children.push(summary);
    children.push(...(item.highlights || []).map(bulletParagraph).filter(Boolean));
    if (index !== filtered.length - 1) children.push(entrySpacer());
  });
  return children;
}

function renderProjects(items, resume, contentWidth) {
  const children = [];
  const filtered = (items || []).filter((item) => nonEmpty([item.name, item.role, item.description, item.summary, ...(item.highlights || [])]).length);
  filtered.forEach((item, index) => {
    const name = cleanText(item.name) || cleanText(item.url) || "Project";
    const nameRun = item.url ? linkRun(name, item.url) : new TextRun({ text: name, bold: true, size: 22 });
    const period = formatDOCXPeriod(item.start_date, item.end_date, false, resume.language);
    children.push(titleParagraph([
      nameRun,
      cleanText(item.role) ? new TextRun({ text: ` | ${cleanText(item.role)}`, size: 20 }) : null,
    ], period, contentWidth));
    const description = bodyParagraph(item.description ?? item.summary, { keepNext: (item.highlights || []).some(cleanText) });
    if (description) children.push(description);
    children.push(...(item.highlights || []).map(bulletParagraph).filter(Boolean));
    const technologies = nonEmpty(item.technologies || item.keywords || []);
    if (technologies.length) children.push(metadataParagraph([`Technologies: ${technologies.join(", ")}`]));
    if (index !== filtered.length - 1) children.push(entrySpacer());
  });
  return children;
}

function renderPortfolio(items, contentWidth) {
  const children = [];
  const filtered = (items || []).filter((item) => nonEmpty([item.name, item.description, item.url, ...(item.highlights || [])]).length);
  filtered.forEach((item, index) => {
    const name = cleanText(item.name) || cleanText(item.url) || "Portfolio item";
    children.push(titleParagraph(item.url ? linkRun(name, item.url) : new TextRun({ text: name, bold: true, size: 22 }), "", contentWidth));
    const description = bodyParagraph(item.description, { keepNext: (item.highlights || []).some(cleanText) });
    if (description) children.push(description);
    children.push(...(item.highlights || []).map(bulletParagraph).filter(Boolean));
    if (item.url) children.push(metadataParagraph([cleanText(item.url)]));
    if (index !== filtered.length - 1) children.push(entrySpacer());
  });
  return children;
}

function renderEducation(items, resume, contentWidth) {
  const children = [];
  const filtered = (items || []).filter((item) => nonEmpty([item.institution, item.degree, item.study_type, item.area, ...(item.highlights || [])]).length);
  filtered.forEach((item, index) => {
    const qualification = nonEmpty([item.degree ?? item.study_type, item.area]).join(" - ");
    const period = formatDOCXPeriod(item.start_date, item.end_date, false, resume.language);
    children.push(titleParagraph([
      qualification ? new TextRun({ text: qualification, bold: true, size: 22 }) : null,
      cleanText(item.institution) ? new TextRun({ text: `${qualification ? " | " : ""}${cleanText(item.institution)}`, size: 21 }) : null,
    ], period, contentWidth));
    const meta = metadataParagraph([item.location, cleanText(item.score) ? `Score: ${cleanText(item.score)}` : ""], (item.highlights || []).some(cleanText));
    if (meta) children.push(meta);
    children.push(...(item.highlights || []).map(bulletParagraph).filter(Boolean));
    if (index !== filtered.length - 1) children.push(entrySpacer());
  });
  return children;
}

function renderSkills(items) {
  return (items || []).map((item) => {
    const skills = nonEmpty(item.items || item.keywords || []);
    const label = cleanText(item.name);
    const level = cleanText(item.level);
    const suffix = nonEmpty([skills.join(", "), level]).join(" | ");
    if (!label && !suffix) return null;
    return new Paragraph({
      children: [
        ...(label ? [new TextRun({ text: label, bold: true })] : []),
        ...(label && suffix ? [new TextRun(": ")] : []),
        ...(suffix ? [new TextRun(suffix)] : []),
      ],
      keepLines: true,
      spacing: { after: 45 },
    });
  }).filter(Boolean);
}

function renderCertifications(items) {
  return (items || []).map((item) => {
    const name = cleanText(item.name) || cleanText(item.url);
    if (!name) return null;
    const title = item.url ? linkRun(name, item.url) : new TextRun({ text: name, bold: true });
    const meta = nonEmpty([
      item.issuer,
      item.date,
      cleanText(item.expiry_date) ? `Expires ${cleanText(item.expiry_date)}` : "",
      cleanText(item.credential_id) ? `Credential ${cleanText(item.credential_id)}` : "",
    ]).join(" | ");
    return new Paragraph({
      children: [title, ...(meta ? [new TextRun({ text: ` - ${meta}`, color: MUTED })] : [])],
      keepLines: true,
      spacing: { after: 50 },
    });
  }).filter(Boolean);
}

function renderLanguages(items) {
  const values = (items || []).map((item) => {
    const name = cleanText(item.language ?? item.name);
    const fluency = cleanText(item.fluency);
    return nonEmpty([name, fluency]).join(" - ");
  }).filter(Boolean);
  if (!values.length) return [];
  return [new Paragraph({
    text: values.join(" | "),
    keepLines: true,
    spacing: { after: 40 },
  })];
}

function renderCustomSection(section, contentWidth) {
  const children = [];
  const items = (section.items || []).filter((item) => nonEmpty([item.title, item.subtitle, item.summary, ...(item.bullets || [])]).length);
  items.forEach((item, index) => {
    const label = cleanText(item.title) || cleanText(item.url) || "Item";
    children.push(titleParagraph([
      item.url ? linkRun(label, item.url) : new TextRun({ text: label, bold: true, size: 22 }),
      cleanText(item.subtitle) ? new TextRun({ text: ` | ${cleanText(item.subtitle)}`, size: 20 }) : null,
    ], item.date, contentWidth));
    const summary = bodyParagraph(item.summary, { keepNext: (item.bullets || []).some(cleanText) });
    if (summary) children.push(summary);
    children.push(...(item.bullets || []).map(bulletParagraph).filter(Boolean));
    if (index !== items.length - 1) children.push(entrySpacer());
  });
  return children;
}

function sectionBody(sectionId, resume, contentWidth) {
  if (sectionId === "summary") return cleanText(resume.summary) ? [bodyParagraph(resume.summary, { after: 20 })] : [];
  if (sectionId === "experience") return renderExperience(resume.experience, resume, contentWidth);
  if (sectionId === "projects") return renderProjects(resume.projects, resume, contentWidth);
  if (sectionId === "portfolio") return renderPortfolio(resume.portfolio, contentWidth);
  if (sectionId === "education") return renderEducation(resume.education, resume, contentWidth);
  if (sectionId === "skills") return renderSkills(resume.skills);
  if (sectionId === "certifications") return renderCertifications(resume.certifications);
  if (sectionId === "languages") return renderLanguages(resume.languages);
  const custom = (resume.custom_sections || []).find((section) => section.id === sectionId);
  return custom ? renderCustomSection(custom, contentWidth) : [];
}

function orderedSections(resume) {
  const customIds = (resume.custom_sections || []).map((section) => section.id).filter(Boolean);
  const allowed = new Set([...DEFAULT_ORDER, ...customIds]);
  const order = Array.isArray(resume.section_order) && resume.section_order.length ? resume.section_order : DEFAULT_ORDER;
  return [...new Set(order.filter((section) => allowed.has(section)))];
}

export function createResumeDOCXDocument(resumeRecord) {
  const resume = resumeRecord?.document || {};
  const title = cleanText(resumeRecord?.title) || `${cleanText(resume.basics?.full_name) || "Resume"} resume`;
  const pageSize = PAGE_SIZES[String(resume.paper_size || "a4").toLowerCase()] || PAGE_SIZES.a4;
  const contentWidth = pageSize.width - (PAGE_MARGIN * 2);
  const hidden = new Set(resume.hidden_sections || []);
  const children = [];

  for (const sectionId of orderedSections(resume)) {
    if (hidden.has(sectionId)) continue;
    if (sectionId === "basics") {
      children.push(...headerChildren(resume, contentWidth));
      continue;
    }
    const body = sectionBody(sectionId, resume, contentWidth).filter(Boolean);
    if (!body.length) continue;
    const custom = (resume.custom_sections || []).find((section) => section.id === sectionId);
    children.push(sectionHeading(custom?.title || SECTION_LABELS[sectionId] || sectionId), ...body);
  }

  return new Document({
    title,
    subject: "Resume exported from FORMA",
    creator: "FORMA - Smart Resume Builder",
    description: "Editable resume document generated locally by FORMA.",
    keywords: "resume, CV, FORMA",
    styles: {
      default: {
        document: {
          run: { font: "Arial", size: 20, color: INK },
          paragraph: { spacing: { after: 0 } },
        },
        title: {
          run: { font: "Arial", size: 52, bold: true, color: INK },
          paragraph: { spacing: { after: 45 } },
        },
        heading1: {
          run: { font: "Arial", size: 20, bold: true, color: INK },
          paragraph: { spacing: { before: 190, after: 85 }, keepNext: true },
        },
        hyperlink: { run: { color: LINK, underline: {} } },
      },
    },
    sections: [{
      properties: {
        page: {
          size: pageSize,
          margin: {
            top: PAGE_MARGIN,
            right: PAGE_MARGIN,
            bottom: PAGE_MARGIN,
            left: PAGE_MARGIN,
            header: 360,
            footer: 360,
            gutter: 0,
          },
        },
      },
      children: children.length ? children : [new Paragraph("Resume")],
    }],
  });
}

export async function createResumeDOCXBlob(resumeRecord) {
  const packed = await Packer.toBlob(createResumeDOCXDocument(resumeRecord));
  return new Blob([packed], { type: DOCX_MIME });
}

export async function exportResumeDOCX(resumeRecord) {
  const blob = await createResumeDOCXBlob(resumeRecord);
  const url = URL.createObjectURL(blob);
  const anchor = window.document.createElement("a");
  anchor.href = url;
  anchor.download = fileSafeName(resumeRecord?.title, "docx");
  anchor.rel = "noopener";
  window.document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
  return blob;
}

export { DOCX_MIME };
