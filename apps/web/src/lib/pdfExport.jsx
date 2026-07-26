import React from "react";
import {
  Document,
  Font,
  Image,
  Link as RendererLink,
  Page,
  StyleSheet,
  Text as RendererText,
  View,
  pdf,
} from "@react-pdf/renderer";
import dejavuBoldURL from "dejavu-fonts-ttf/ttf/DejaVuSans-Bold.ttf?url";
import dejavuRegularURL from "dejavu-fonts-ttf/ttf/DejaVuSans-ExtraLight.ttf?url";
import { fileSafeName } from "./resume.js";

const PDF_CYRILLIC = "Forma Cyrillic";

function fontAsset(bundledURL, relativePath) {
  if (import.meta.env.MODE !== "test") return bundledURL;
  const value = new URL(relativePath, import.meta.url);
  const pathname = decodeURIComponent(value.pathname);
  return /^\/[A-Za-z]:\//.test(pathname) ? pathname.slice(1) : pathname;
}

// These full TTF files come from the dejavu-fonts-ttf dependency. Latin text
// uses PDF standard fonts; DejaVu is only embedded for Cyrillic glyph runs.
Font.register({
  family: PDF_CYRILLIC,
  fonts: [
    { src: fontAsset(dejavuRegularURL, "../../node_modules/dejavu-fonts-ttf/ttf/DejaVuSans-ExtraLight.ttf"), fontWeight: 400 },
    { src: fontAsset(dejavuBoldURL, "../../node_modules/dejavu-fonts-ttf/ttf/DejaVuSans-Bold.ttf"), fontWeight: 700 },
  ],
});
Font.registerHyphenationCallback((word) => [word]);

function scriptAwareChildren(children) {
  return React.Children.map(children, (child) => {
    if (typeof child !== "string" && typeof child !== "number") return child;
    return String(child).split(/([\p{Script=Cyrillic}\p{Mark}]+)/gu).filter(Boolean).map((part, index) => (
      /\p{Script=Cyrillic}/u.test(part)
        ? <RendererText style={{ fontFamily: PDF_CYRILLIC }} key={`${part}-${index}`}>{part}</RendererText>
        : part
    ));
  });
}

function Text({ children, ...props }) {
  return <RendererText {...props}>{scriptAwareChildren(children)}</RendererText>;
}

function Link({ children, ...props }) {
  return <RendererLink {...props}>{scriptAwareChildren(children)}</RendererLink>;
}

const SECTION_LABELS = {
  experience: "Experience",
  projects: "Projects",
  portfolio: "Portfolio",
  education: "Education",
  skills: "Skills",
  certifications: "Certifications",
  languages: "Languages",
};

function text(value) {
  return String(value || "").trim();
}

function formatDate(value, locale) {
  const [year, month] = text(value).split("-");
  if (!year) return "";
  if (!month) return year;
  const date = new Date(Number(year), Number(month) - 1, 1);
  if (Number.isNaN(date.getTime())) return text(value);
  return new Intl.DateTimeFormat(locale || "en", { month: "short", year: "numeric" }).format(date);
}

export function formatPDFPeriod(startDate, endDate, current = false, locale = "en") {
  return [formatDate(startDate, locale), current ? "Present" : formatDate(endDate, locale)].filter(Boolean).join(" - ");
}

function normalizeHref(value, kind = "web") {
  const candidate = text(value);
  if (!candidate) return "";
  if (kind === "email") return `mailto:${candidate}`;
  if (/^[a-z][a-z0-9+.-]*:/i.test(candidate)) return candidate;
  return `https://${candidate}`;
}

function safePhotoSource(value) {
  const candidate = text(value);
  return /^data:image\/(?:png|jpe?g);base64,/i.test(candidate) ? candidate : "";
}

function getTemplateConfig(template, paperSize) {
  const id = ["editorial", "ats", "compact", "modern", "classic", "portrait", "minimal"].includes(template) ? template : "editorial";
  const sans = "Helvetica";
  const serif = "Times-Roman";
  const serifBold = "Times-Bold";
  const compact = id === "compact";
  const minimal = id === "minimal";
  const modern = id === "modern";
  const classic = id === "classic";
  const ats = id === "ats";
  const letter = paperSize === "letter";
  const defaultPaddingY = minimal ? 48 : compact ? 28 : ats ? 34 : 38;
  return {
    id,
    sans,
    serif,
    bodyFamily: classic ? serif : sans,
    nameFamily: modern || ats || minimal ? sans : serif,
    roleFamily: modern || ats ? sans : serifBold,
    pagePaddingX: minimal ? 54 : compact ? 37 : ats ? 40 : 45,
    pagePaddingY: Math.max(24, defaultPaddingY - (letter ? 8 : 0)),
    baseSize: compact ? 8.6 : letter ? 9.3 : 9.5,
    lineHeight: compact ? 1.33 : letter ? 1.38 : 1.4,
    sectionGap: minimal ? (letter ? 19 : 25) : compact ? 13 : letter ? 15 : 19,
    entryGap: minimal ? (letter ? 11 : 14) : compact ? 7 : letter ? 8 : 10,
    nameSize: modern ? 35 : compact ? 32 : ats ? 23 : minimal ? 32 : 42,
    classic,
    ats,
    compact,
    modern,
    minimal,
    portrait: id === "portrait",
    letter,
  };
}

function createStyles(config) {
  const headerCentered = config.classic;
  const singleColumnHeader = config.ats || config.classic;
  return StyleSheet.create({
    page: {
      paddingTop: config.pagePaddingY,
      paddingRight: config.pagePaddingX,
      paddingBottom: config.pagePaddingY,
      paddingLeft: config.pagePaddingX,
      color: "#101010",
      backgroundColor: "#ffffff",
      fontFamily: config.bodyFamily,
      fontSize: config.baseSize,
      lineHeight: config.lineHeight,
    },
    pageAccent: {
      borderTopWidth: config.modern || config.portrait ? 5 : 0,
      borderTopColor: "#101010",
    },
    header: {
      flexDirection: singleColumnHeader ? "column" : "row",
      alignItems: headerCentered ? "center" : "flex-start",
      justifyContent: headerCentered ? "center" : "space-between",
      marginBottom: config.minimal ? (config.letter ? 22 : 29) : config.compact ? 16 : config.letter ? 19 : 23,
      textAlign: headerCentered ? "center" : "left",
    },
    portraitHeader: {
      flexDirection: "row",
      alignItems: "center",
    },
    identity: {
      flexGrow: 1,
      flexShrink: 1,
      paddingRight: singleColumnHeader ? 0 : 18,
    },
    name: {
      fontFamily: config.nameFamily,
      fontSize: config.nameSize,
      fontWeight: config.modern || config.ats ? 700 : config.minimal ? 500 : 400,
      lineHeight: 0.94,
      letterSpacing: config.modern || config.minimal ? -1.1 : config.ats ? 0 : -0.45,
      textTransform: config.ats ? "uppercase" : "none",
    },
    headline: {
      marginTop: 6,
      fontFamily: config.sans,
      fontSize: config.ats ? 9.3 : 10,
      fontWeight: 600,
      letterSpacing: 1.35,
      textTransform: "uppercase",
    },
    contacts: {
      width: singleColumnHeader ? "auto" : config.portrait ? 145 : 170,
      marginTop: singleColumnHeader ? 10 : 2,
      flexDirection: singleColumnHeader ? "row" : "column",
      flexWrap: "wrap",
      justifyContent: headerCentered ? "center" : "flex-start",
    },
    contact: {
      marginRight: singleColumnHeader ? 12 : 0,
      marginBottom: 4,
      color: "#2f2f2d",
      fontFamily: config.sans,
      fontSize: config.compact ? 8 : 8.5,
      textDecoration: "none",
    },
    photo: {
      width: config.portrait ? 68 : 52,
      height: config.portrait ? 68 : 52,
      marginRight: config.portrait ? 16 : 0,
      marginLeft: config.portrait ? 0 : 14,
      objectFit: "cover",
      borderWidth: 0.7,
      borderColor: "#c8c8c3",
      borderRadius: config.classic ? 26 : 0,
    },
    summary: {
      marginBottom: 0,
      borderTopWidth: config.modern ? 4 : 0,
      borderTopColor: "#101010",
      borderBottomWidth: config.modern ? 0 : 0.7,
      borderBottomColor: "#b8b8b2",
      paddingTop: config.modern ? 15 : 0,
      paddingBottom: config.modern ? 0 : config.minimal ? (config.letter ? 17 : 22) : config.compact ? 12 : config.letter ? 14 : 17,
      fontSize: config.compact ? 9.3 : config.letter ? 10.5 : 10.8,
      lineHeight: config.letter ? 1.46 : 1.52,
    },
    sectionTitle: {
      marginBottom: config.minimal ? (config.letter ? 12 : 16) : config.compact ? 8 : config.letter ? 10 : 12,
      borderBottomWidth: config.modern || config.ats || config.classic ? 0.7 : 0,
      borderBottomColor: "#777777",
      paddingBottom: config.modern || config.ats || config.classic ? 4 : 0,
      fontFamily: config.sans,
      fontSize: config.compact ? 8 : 8.5,
      fontWeight: 700,
      letterSpacing: 1.25,
      textAlign: config.classic ? "center" : "left",
      textTransform: "uppercase",
    },
    entry: {
      marginBottom: config.entryGap,
      paddingBottom: config.entryGap,
      borderBottomWidth: 0.45,
      borderBottomColor: config.minimal ? "#eeeeeb" : "#ddddda",
    },
    lastEntry: {
      marginBottom: 0,
      paddingBottom: 0,
      borderBottomWidth: 0,
    },
    sectionSpacer: {
      height: config.sectionGap,
    },
    experience: {
      flexDirection: "row",
    },
    entryMeta: {
      width: config.ats ? 105 : config.compact ? 100 : 120,
      paddingRight: 14,
      color: "#363633",
      fontSize: config.compact ? 7.8 : 8.2,
    },
    entryBody: {
      flexGrow: 1,
      flexShrink: 1,
    },
    company: {
      marginBottom: 2,
      fontFamily: config.sans,
      fontSize: config.compact ? 7.8 : 8,
      fontWeight: 700,
      letterSpacing: 0.8,
      textTransform: "uppercase",
    },
    roleLine: {
      flexDirection: "row",
      alignItems: "flex-start",
      justifyContent: "space-between",
    },
    role: {
      flexGrow: 1,
      flexShrink: 1,
      paddingRight: 12,
      fontFamily: config.roleFamily,
      fontSize: config.compact ? 10.5 : config.ats ? 10 : 12,
      fontWeight: config.ats || config.modern ? 700 : 600,
      lineHeight: 1.15,
    },
    date: {
      color: "#555552",
      fontSize: config.compact ? 7.8 : 8.2,
      textAlign: "right",
    },
    entrySummary: {
      marginTop: 4,
      marginBottom: 4,
      fontSize: config.compact ? 8.4 : 9.1,
      lineHeight: 1.42,
    },
    bulletRow: {
      flexDirection: "row",
      paddingLeft: 2,
      marginTop: 2,
    },
    bullet: {
      width: 10,
      fontSize: config.compact ? 8.4 : 9,
    },
    bulletText: {
      flexGrow: 1,
      flexShrink: 1,
      fontSize: config.compact ? 8.4 : 9,
      lineHeight: 1.38,
    },
    technology: {
      marginTop: 5,
      color: "#555552",
      fontSize: config.compact ? 7.8 : 8.2,
    },
    link: {
      color: "#101010",
      textDecoration: "none",
    },
    education: {
      flexDirection: "row",
      justifyContent: "space-between",
    },
    educationSide: {
      width: "48%",
    },
    educationRight: {
      width: "48%",
      textAlign: "right",
    },
    skillRow: {
      flexDirection: "row",
      marginBottom: 5,
    },
    skillName: {
      width: 112,
      paddingRight: 12,
      fontWeight: 700,
    },
    skillItems: {
      flexGrow: 1,
      flexShrink: 1,
    },
    inlineRow: {
      flexDirection: "row",
      justifyContent: "space-between",
      marginBottom: 5,
    },
    inlineMain: {
      flexGrow: 1,
      flexShrink: 1,
      paddingRight: 12,
      fontWeight: 600,
    },
    inlineMeta: {
      color: "#555552",
      textAlign: "right",
    },
  });
}

function BulletList({ items, styles }) {
  return (items || []).filter((item) => text(item)).map((item, index) => (
    <View style={styles.bulletRow} key={`${item}-${index}`}>
      <Text style={styles.bullet}>{"\u2022"}</Text>
      <Text style={styles.bulletText}>{text(item)}</Text>
    </View>
  ));
}

function Contact({ value, href, styles }) {
  if (!text(value)) return null;
  return href
    ? <Link src={href} style={styles.contact}>{text(value)}</Link>
    : <Text style={styles.contact}>{text(value)}</Text>;
}

function Header({ resume, styles, config }) {
  const basics = resume.basics || {};
  const photo = config.ats ? "" : safePhotoSource(basics.photo_url);
  const contacts = (
    <View style={styles.contacts}>
      <Contact value={basics.email} href={normalizeHref(basics.email, "email")} styles={styles} />
      <Contact value={basics.phone} styles={styles} />
      <Contact value={basics.location} styles={styles} />
      <Contact value={basics.website} href={normalizeHref(basics.website)} styles={styles} />
      {(basics.profiles || []).slice(0, 2).map((profile, index) => (
        <Contact
          key={profile.id || profile.url || index}
          value={text(profile.username) ? `${text(profile.network) || "Profile"}: ${text(profile.username)}` : text(profile.network) || text(profile.url)}
          href={normalizeHref(profile.url)}
          styles={styles}
        />
      ))}
    </View>
  );
  const identity = (
    <View style={styles.identity}>
      <Text style={styles.name}>{text(basics.full_name) || "Resume"}</Text>
      {text(basics.headline) && <Text style={styles.headline}>{text(basics.headline)}</Text>}
    </View>
  );

  return (
    <View style={[styles.header, config.portrait && styles.portraitHeader]} wrap={false}>
      {config.portrait && photo && <Image src={photo} style={styles.photo} />}
      {identity}
      {contacts}
      {!config.portrait && photo && <Image src={photo} style={styles.photo} />}
    </View>
  );
}

function SectionFrame({ label, styles, children, spaced }) {
  return (
    <>
      {spaced && <View style={styles.sectionSpacer} />}
      <Text style={styles.sectionTitle} minPresenceAhead={32}>{label}</Text>
      {children}
    </>
  );
}

function EntryFrame({ styles, index, count, children, style, keepTogether = false }) {
  return <View style={[styles.entry, index === count - 1 && styles.lastEntry, style]} minPresenceAhead={54} wrap={!keepTogether}>{children}</View>;
}

function Experience({ items, resume, styles }) {
  return items.map((item, index) => (
    <EntryFrame
      styles={styles}
      index={index}
      count={items.length}
      style={styles.experience}
      keepTogether={(item.highlights || []).length <= 8 && JSON.stringify(item).length < 1_400}
      key={item.id || index}
    >
      <View style={styles.entryMeta}>
        {text(item.company) && <Text style={styles.company}>{text(item.company)}</Text>}
        {text(item.location) && <Text>{text(item.location)}</Text>}
        {formatPDFPeriod(item.start_date, item.end_date, item.current, resume.language) && <Text>{formatPDFPeriod(item.start_date, item.end_date, item.current, resume.language)}</Text>}
      </View>
      <View style={styles.entryBody}>
        {text(item.position) && <Text style={styles.role}>{text(item.position)}</Text>}
        {text(item.summary) && <Text style={styles.entrySummary}>{text(item.summary)}</Text>}
        <BulletList items={item.highlights} styles={styles} />
      </View>
    </EntryFrame>
  ));
}

function Projects({ items, resume, styles }) {
  return items.map((item, index) => (
    <EntryFrame styles={styles} index={index} count={items.length} keepTogether={(item.highlights || []).length <= 8 && JSON.stringify(item).length < 1_400} key={item.id || index}>
      <View style={styles.roleLine}>
        {item.url ? <Link src={normalizeHref(item.url)} style={[styles.role, styles.link]}>{text(item.name) || text(item.url)}</Link> : <Text style={styles.role}>{text(item.name)}</Text>}
        <Text style={styles.date}>{formatPDFPeriod(item.start_date, item.end_date, false, resume.language)}</Text>
      </View>
      {text(item.description) && <Text style={styles.entrySummary}>{text(item.description)}</Text>}
      <BulletList items={item.highlights} styles={styles} />
      {(item.technologies || []).length > 0 && <Text style={styles.technology}>{item.technologies.filter(Boolean).join(", ")}</Text>}
    </EntryFrame>
  ));
}

function Portfolio({ items, styles }) {
  return items.map((item, index) => (
    <EntryFrame styles={styles} index={index} count={items.length} keepTogether={(item.highlights || []).length <= 8 && JSON.stringify(item).length < 1_400} key={item.id || index}>
      {item.url ? <Link src={normalizeHref(item.url)} style={[styles.role, styles.link]}>{text(item.name) || text(item.url)}</Link> : <Text style={styles.role}>{text(item.name)}</Text>}
      {text(item.description) && <Text style={styles.entrySummary}>{text(item.description)}</Text>}
      <BulletList items={item.highlights} styles={styles} />
    </EntryFrame>
  ));
}

function Education({ items, resume, styles }) {
  return items.map((item, index) => (
    <EntryFrame styles={styles} index={index} count={items.length} style={styles.education} keepTogether key={item.id || index}>
      <View style={styles.educationSide}>
        <Text style={styles.company}>{text(item.institution)}</Text>
        {text(item.location) && <Text>{text(item.location)}</Text>}
      </View>
      <View style={styles.educationRight}>
        <Text style={styles.role}>{[text(item.degree), text(item.area)].filter(Boolean).join(" ")}</Text>
        <Text>{formatPDFPeriod(item.start_date, item.end_date, false, resume.language)}</Text>
        {text(item.score) && <Text>{text(item.score)}</Text>}
      </View>
    </EntryFrame>
  ));
}

function Skills({ items, styles }) {
  return items.map((item, index) => (
    <View style={styles.skillRow} key={item.id || index} wrap={false}>
      <Text style={styles.skillName}>{text(item.name)}</Text>
      <Text style={styles.skillItems}>{(item.items || []).filter(Boolean).join(", ")}</Text>
    </View>
  ));
}

function Certifications({ items, styles }) {
  return items.map((item, index) => (
    <View style={styles.inlineRow} key={item.id || index} wrap={false}>
      <Text style={styles.inlineMain}>{text(item.name)}</Text>
      <Text style={styles.inlineMeta}>{[text(item.issuer), text(item.date)].filter(Boolean).join(" | ")}</Text>
    </View>
  ));
}

function Languages({ items, styles }) {
  return <Text>{items.map((item) => [text(item.language), text(item.fluency)].filter(Boolean).join(" - ")).filter(Boolean).join(" | ")}</Text>;
}

function renderSection(id, resume, styles, spaced = false) {
  const items = Array.isArray(resume[id]) ? resume[id] : [];
  if (id === "summary" && text(resume.summary)) return <View style={styles.summary} key={id}><Text>{text(resume.summary)}</Text></View>;
  if (!SECTION_LABELS[id] || items.length === 0) return null;
  let body = null;
  if (id === "experience") body = <Experience items={items} resume={resume} styles={styles} />;
  if (id === "projects") body = <Projects items={items} resume={resume} styles={styles} />;
  if (id === "portfolio") body = <Portfolio items={items} styles={styles} />;
  if (id === "education") body = <Education items={items} resume={resume} styles={styles} />;
  if (id === "skills") body = <Skills items={items} styles={styles} />;
  if (id === "certifications") body = <Certifications items={items} styles={styles} />;
  if (id === "languages") body = <Languages items={items} styles={styles} />;
  return <SectionFrame label={SECTION_LABELS[id]} styles={styles} spaced={spaced} key={id}>{body}</SectionFrame>;
}

export function ResumePDFDocument({ resume, title = "Resume" }) {
  const config = getTemplateConfig(resume?.template, resume?.paper_size);
  const styles = createStyles(config);
  const hidden = new Set(resume?.hidden_sections || []);
  const order = resume?.section_order || ["basics", "summary", "experience", "projects", "education", "skills"];
  const bodyOrder = order.filter((id) => id !== "basics" && !hidden.has(id));
  const pageSize = resume?.paper_size === "letter" ? "LETTER" : "A4";

  return (
    <Document
      title={title}
      author={text(resume?.basics?.full_name)}
      subject="Resume exported from FORMA"
      creator="FORMA - Smart Resume Builder"
      producer="FORMA - Smart Resume Builder"
      language={resume?.language || "en"}
    >
      <Page size={pageSize} style={[styles.page, styles.pageAccent]} wrap>
        {!hidden.has("basics") && <Header resume={resume || {}} styles={styles} config={config} />}
        {bodyOrder.map((id, index) => renderSection(id, resume || {}, styles, index > 0))}
      </Page>
    </Document>
  );
}

export async function createResumePDFBlob(resume) {
  return pdf(<ResumePDFDocument resume={resume.document} title={resume.title} />).toBlob();
}

export async function exportResumePDF(resume) {
  const blob = await createResumePDFBlob(resume);
  const url = URL.createObjectURL(blob);
  const anchor = window.document.createElement("a");
  anchor.href = url;
  anchor.download = fileSafeName(resume.title, "pdf");
  anchor.rel = "noopener";
  window.document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
  return blob;
}
