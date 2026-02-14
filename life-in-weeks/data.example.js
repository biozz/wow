export const lifeConfig = {
  // ISO date string, used as the origin for all weeks.
  birthDate: "1990-08-28",

  // Flat events: year, optional month, optional day.
  // Granularity = which fields are present.
  events: [
    { year: 2026, text: "Event for the whole year" },
    { year: 2026, month: 1, text: "Event for January" },
    { year: 2026, month: 1, day: 15, text: "Event for a specific day" },
  ],
};
