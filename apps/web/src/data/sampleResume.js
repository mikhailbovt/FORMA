export const sectionCatalog = [
  { id: "basics", label: "Header" },
  { id: "summary", label: "Summary" },
  { id: "experience", label: "Experience" },
  { id: "projects", label: "Projects" },
  { id: "portfolio", label: "Portfolio" },
  { id: "education", label: "Education" },
  { id: "skills", label: "Skills" },
  { id: "certifications", label: "Certifications" },
  { id: "languages", label: "Languages" },
];

export const emptyResumeDocument = {
  basics: {
    full_name: "Your name",
    headline: "Your professional headline",
    email: "",
    phone: "",
    location: "",
    website: "",
    photo_url: "",
    profiles: [],
  },
  summary: "Write two or three focused sentences about the value you bring.",
  experience: [],
  projects: [],
  portfolio: [],
  education: [],
  skills: [],
  certifications: [],
  languages: [],
  custom_sections: [],
  section_order: ["basics", "summary", "experience", "projects", "education", "skills"],
  hidden_sections: [],
  template: "editorial",
  paper_size: "a4",
  language: "en",
};

export const sampleResume = {
  title: "Product Engineer CV",
  document: {
    ...emptyResumeDocument,
    basics: {
      full_name: "Alex Morgan",
      headline: "Product Engineer",
      email: "alex.morgan@example.com",
      phone: "+1 (415) 555-0198",
      location: "Austin, TX",
      website: "alexmorgan.dev",
      photo_url: "",
      profiles: [
        { id: "profile-linkedin", network: "LinkedIn", username: "alexmorgan", url: "https://linkedin.com/in/alexmorgan" },
      ],
    },
    summary:
      "Product engineer with 6+ years building user-centered SaaS products. I ship reliable, measurable features and collaborate across design, data, and engineering to drive impact.",
    experience: [
      {
        id: "exp-sentry",
        company: "Sentry Labs",
        position: "Product Engineer",
        location: "Austin, TX",
        start_date: "2022-04",
        end_date: "",
        current: true,
        summary: "Built core product features for a developer observability platform used by 70k+ teams.",
        highlights: [
          "Built alerting workflow that reduced noise and improved response time.",
          "Designed and shipped notification preferences that increased user activation by 18%.",
          "Partnered with the data team to instrument key signals and build actionable dashboards.",
        ],
        skills: ["TypeScript", "React", "Go", "PostgreSQL"],
      },
      {
        id: "exp-compass",
        company: "Compass",
        position: "Software Engineer",
        location: "San Francisco, CA",
        start_date: "2019-06",
        end_date: "2022-03",
        current: false,
        summary: "Worked on internal tooling and customer-facing features for a real estate platform.",
        highlights: [
          "Developed a bulk import tool that saved 20+ hours per week for operations teams.",
          "Improved search relevance, contributing to a 12% increase in engagement.",
          "Refactored frontend components to improve performance and maintainability.",
        ],
        skills: ["React", "Node.js", "GraphQL"],
      },
      {
        id: "exp-rivet",
        company: "Rivet",
        position: "Software Engineer",
        location: "New York, NY",
        start_date: "2017-08",
        end_date: "2019-05",
        current: false,
        summary: "Built features and integrations for enterprise customers.",
        highlights: [
          "Delivered an SSO integration used by more than 1,000 organizations.",
          "Automated a data pipeline, reducing manual work by 30%.",
        ],
        skills: ["JavaScript", "Python", "AWS"],
      },
    ],
    projects: [
      {
        id: "project-signal",
        name: "Signalboard",
        role: "Creator",
        start_date: "2024",
        end_date: "",
        description: "Open-source incident review workspace for small engineering teams.",
        highlights: ["Reached 1,200 GitHub stars and 40 community contributors."],
        technologies: ["Go", "React", "PostgreSQL"],
        url: "https://github.com/example/signalboard",
      },
    ],
    education: [
      {
        id: "edu-utexas",
        institution: "University of Texas at Austin",
        degree: "B.S.",
        area: "Computer Science",
        location: "Austin, TX",
        start_date: "2013",
        end_date: "2017",
        score: "",
        highlights: [],
      },
    ],
    skills: [
      { id: "skill-product", name: "Product engineering", items: ["Product strategy", "Experimentation", "Analytics"] },
      { id: "skill-stack", name: "Engineering", items: ["Go", "TypeScript", "React", "PostgreSQL", "AWS"] },
    ],
    certifications: [],
    languages: [
      { id: "lang-en", language: "English", fluency: "Native" },
      { id: "lang-es", language: "Spanish", fluency: "B2" },
    ],
  },
};

export function makeBlankResume(title = "Untitled resume") {
  return {
    title,
    document: structuredClone(emptyResumeDocument),
  };
}

export function makeSampleResume() {
  return structuredClone(sampleResume);
}
