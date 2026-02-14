// --- Config -----------------------------------------------------------------

const TOTAL_YEARS = 100;
const WEEKS_PER_YEAR = 52;
const WEEKS_PER_ROW = 26;
const ROWS_PER_YEAR = WEEKS_PER_YEAR / WEEKS_PER_ROW; // 2 with the defaults
const TOTAL_ROWS = TOTAL_YEARS * ROWS_PER_YEAR; // 200 rows

const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

const DAY_MS = 24 * 60 * 60 * 1000;

const YEAR_COLORS = [
  "#0ea5e9",
  "#8b5cf6",
  "#22c55e",
  "#f97316",
  "#e11d48",
  "#a855f7",
  "#14b8a6",
  "#facc15",
  "#fb7185",
  "#38bdf8",
];

const WEEK_ASPECT_X = 1.6;

const ZOOM_STEP = 0.5;
const ZOOM_MIN = 0.5;
const ZOOM_MAX = 4;

let currentZoom = 1;

// --- Helpers ----------------------------------------------------------------

const pad2 = (n) => String(n).padStart(2, "0");

const toDateKey = (year, month, day) =>
  `${year}-${pad2(month)}-${pad2(day ?? 1)}`;

const isLeapYear = (year) =>
  (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;

function dayOfYearUTC(year, month, day) {
  const date = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));
  const startOfYear = new Date(Date.UTC(year, 0, 1, 0, 0, 0, 0));
  const diffDays = Math.floor(
    (date.getTime() - startOfYear.getTime()) / DAY_MS,
  );
  return diffDays + 1;
}

function normalizedEffectiveDayOfYear(year, month, day) {
  const doy = dayOfYearUTC(year, month, day);
  if (!isLeapYear(year)) return doy;

  const feb29 = dayOfYearUTC(year, 2, 29);
  const mar1 = feb29 + 1;

  if (doy === feb29) {
    // Treat Feb 29 as sharing the same visual week as Feb 28.
    return feb29 - 1;
  }
  if (doy >= mar1) {
    // Compact the leap day by shifting everything from Mar 1 onwards back by 1.
    return doy - 1;
  }
  return doy;
}

function normalizedWeekInYear(year, month, day) {
  const effective = normalizedEffectiveDayOfYear(year, month, day);
  const baseIndex = Math.floor((effective - 1) / 7);
  // Fold any trailing days into the last visual week so we always have 52.
  return Math.min(WEEKS_PER_YEAR - 1, baseIndex);
}

function dateFromDayOfYearUTC(year, dayOfYear) {
  const startOfYear = new Date(Date.UTC(year, 0, 1, 0, 0, 0, 0));
  const date = new Date(startOfYear.getTime() + (dayOfYear - 1) * DAY_MS);
  return {
    year: date.getUTCFullYear(),
    month: date.getUTCMonth() + 1,
    day: date.getUTCDate(),
  };
}

function canonicalDateForWeek(year, weekInYear) {
  // Use the first effective day of this visual week as the canonical date.
  let effectiveDay = weekInYear * 7 + 1;
  if (effectiveDay > 365) effectiveDay = 365;

  if (!isLeapYear(year)) {
    return dateFromDayOfYearUTC(year, effectiveDay);
  }

  const feb29 = dayOfYearUTC(year, 2, 29);
  const feb28 = feb29 - 1;

  let realDay;
  if (effectiveDay < feb28) {
    realDay = effectiveDay;
  } else if (effectiveDay === feb28) {
    // Canonicalize the shared Feb 28 / Feb 29 week to Feb 28.
    realDay = feb28;
  } else {
    // For everything after the compacted leap day, expand back out.
    realDay = effectiveDay + 1;
  }

  return dateFromDayOfYearUTC(year, realDay);
}

function safeParseBirthDate(iso) {
  if (!iso) throw new Error("lifeConfig.birthDate is required");
  const d = new Date(`${iso}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) {
    throw new Error(`Invalid birth date in lifeConfig: ${iso}`);
  }
  return d;
}

function buildEventIndexFromFlat(events) {
  const byYear = new Map();
  const byMonth = new Map();
  const byDate = new Map();

  for (const ev of events) {
    if (!ev || typeof ev.year !== "number") continue;
    const year = ev.year;
    const yearKey = String(year);

    if (typeof ev.month === "number" && typeof ev.day === "number") {
      const monthKey = `${year}-${pad2(ev.month)}`;
      const dateKey = toDateKey(year, ev.month, ev.day);
      const existing = byDate.get(dateKey) ?? [];
      byDate.set(dateKey, existing.concat(ev));
    } else if (typeof ev.month === "number") {
      const monthKey = `${year}-${pad2(ev.month)}`;
      const existing = byMonth.get(monthKey) ?? [];
      byMonth.set(monthKey, existing.concat(ev));
    } else {
      const existing = byYear.get(yearKey) ?? [];
      byYear.set(yearKey, existing.concat(ev));
    }
  }

  return { byYear, byMonth, byDate };
}

function buildEventIndex(config) {
  const flatEvents = Array.isArray(config.events) ? config.events : null;
  if (flatEvents?.length) {
    return buildEventIndexFromFlat(flatEvents);
  }

  const byYear = new Map();
  const byMonth = new Map();
  const byDate = new Map();

  const years = Array.isArray(config.years) ? config.years : [];

  for (const yearEntry of years) {
    if (!yearEntry || typeof yearEntry.year !== "number") continue;
    const year = yearEntry.year;
    const yearKey = String(year);

    if (Array.isArray(yearEntry.events) && yearEntry.events.length) {
      const existing = byYear.get(yearKey) ?? [];
      byYear.set(yearKey, existing.concat(yearEntry.events));
    }

    const months = Array.isArray(yearEntry.months) ? yearEntry.months : [];
    for (const monthEntry of months) {
      if (!monthEntry || typeof monthEntry.month !== "number") continue;
      const month = monthEntry.month;
      const monthKey = `${year}-${pad2(month)}`;

      if (Array.isArray(monthEntry.events) && monthEntry.events.length) {
        const existing = byMonth.get(monthKey) ?? [];
        byMonth.set(monthKey, existing.concat(monthEntry.events));
      }

      const days = Array.isArray(monthEntry.days) ? monthEntry.days : [];
      for (const dayEntry of days) {
        if (!dayEntry || typeof dayEntry.day !== "number") continue;
        const day = dayEntry.day;
        const dateKey = toDateKey(year, month, day);

        if (Array.isArray(dayEntry.events) && dayEntry.events.length) {
          const existing = byDate.get(dateKey) ?? [];
          byDate.set(dateKey, existing.concat(dayEntry.events));
        }
      }
    }
  }

  return { byYear, byMonth, byDate };
}

function buildDayEventsByWeekIndex(eventIndex, birthDate) {
  const map = new Map();
  const birthYear = birthDate.getUTCFullYear();

  for (const [dateKey, events] of eventIndex.byDate.entries()) {
    const [yStr, mStr, dStr] = dateKey.split("-");
    const year = Number(yStr);
    const month = Number(mStr);
    const day = Number(dStr);

    if (
      !Number.isFinite(year) ||
      !Number.isFinite(month) ||
      !Number.isFinite(day)
    )
      continue;

    const yearOffset = year - birthYear;
    if (yearOffset < 0 || yearOffset >= TOTAL_YEARS) continue;

    const weekInYear = normalizedWeekInYear(year, month, day);
    const globalIndex = yearOffset * WEEKS_PER_YEAR + weekInYear;

    const existing = map.get(globalIndex) ?? [];
    map.set(globalIndex, existing.concat(events));
  }

  return map;
}

function computeWeekSize(gridEl) {
  const parent = gridEl.parentElement ?? gridEl;
  const parentRect = parent.getBoundingClientRect();

  const availableWidth = parentRect.width || window.innerWidth * 0.6;
  let availableHeight = parentRect.height || window.innerHeight;
  // Reserve space for paddings / footer so the grid fits at 1x.
  availableHeight = Math.max(0, availableHeight - 56);

  const widthSize = availableWidth / (WEEKS_PER_ROW * WEEK_ASPECT_X);
  const heightSize = availableHeight / TOTAL_ROWS;

  const baseSize = Math.max(2, Math.min(widthSize, heightSize));
  return baseSize;
}

function computeAgeLabel(birthDate, pointDate) {
  let years = pointDate.getUTCFullYear() - birthDate.getUTCFullYear();
  let months = pointDate.getUTCMonth() - birthDate.getUTCMonth();
  let days = pointDate.getUTCDate() - birthDate.getUTCDate();

  if (days < 0) {
    months -= 1;
  }
  if (months < 0) {
    years -= 1;
    months += 12;
  }

  if (years < 0) return "";

  const parts = [];
  if (years > 0) parts.push(`${years}y`);
  if (months > 0) parts.push(`${months}m`);
  return parts.join(" ");
}

// --- Rendering --------------------------------------------------------------

function renderYearLabels(labelsEl) {
  if (!labelsEl) return;
  labelsEl.innerHTML = "";
  for (let row = 0; row < TOTAL_ROWS; row++) {
    const yearOffset = Math.floor(row / ROWS_PER_YEAR);
    const isFirstRowOfYear = row % ROWS_PER_YEAR === 0;
    const div = document.createElement("div");
    div.className = "year-label";
    // Show label at first row of next decade (one full year lower)
    if (isFirstRowOfYear && yearOffset > 0 && yearOffset % 10 === 1) {
      div.textContent = String(yearOffset - 1);
    }
    labelsEl.appendChild(div);
  }
}

function renderWeeks(gridEl, eventIndex, dayEventsByWeekIndex, birthDate) {
  gridEl.innerHTML = "";
  gridEl.style.setProperty("--weeksPerRow", String(WEEKS_PER_ROW));

  const birthYear = birthDate.getUTCFullYear();
  const totalWeeks = TOTAL_YEARS * WEEKS_PER_YEAR;

  // Weeks before birth within the birth year are greyed out.
  const birthWeekInYear = normalizedWeekInYear(
    birthYear,
    birthDate.getUTCMonth() + 1,
    birthDate.getUTCDate(),
  );
  const preBirthWeeks = birthWeekInYear;

  const weekSquares = new Array(totalWeeks);

  // Precompute which week indices correspond to birthdays (one per year).
  const birthdayWeeks = new Set();
  for (let yearOffset = 0; yearOffset < TOTAL_YEARS; yearOffset += 1) {
    const year = birthYear + yearOffset;
    const birthdayWeekInYear = normalizedWeekInYear(
      year,
      birthDate.getUTCMonth() + 1,
      birthDate.getUTCDate(),
    );
    const globalIndex = yearOffset * WEEKS_PER_YEAR + birthdayWeekInYear;
    if (globalIndex >= 0 && globalIndex < totalWeeks) {
      birthdayWeeks.add(globalIndex);
    }
  }

  for (let weekIndex = 0; weekIndex < totalWeeks; weekIndex += 1) {
    const yearOffset = Math.floor(weekIndex / WEEKS_PER_YEAR);
    const weekInYear = weekIndex % WEEKS_PER_YEAR;
    const year = birthYear + yearOffset;

    const { month, day } = canonicalDateForWeek(year, weekInYear);
    const dateKey = toDateKey(year, month, day);

    const colorIndex =
      ((yearOffset % YEAR_COLORS.length) + YEAR_COLORS.length) %
      YEAR_COLORS.length;
    const yearColor = YEAR_COLORS[colorIndex];

    const monthKey = `${year}-${pad2(month)}`;
    const hasEvents =
      (eventIndex.byYear.get(String(year))?.length ?? 0) > 0 ||
      (eventIndex.byMonth.get(monthKey)?.length ?? 0) > 0 ||
      (dayEventsByWeekIndex.get(weekIndex)?.length ?? 0) > 0;

    const div = document.createElement("div");
    div.className = "week";
    if (weekIndex < preBirthWeeks) div.classList.add("pre-birth");
    if (hasEvents) div.classList.add("has-events");
    // Mark birthday weeks so we can style them specially.
    if (birthdayWeeks.has(weekIndex)) div.classList.add("birthday");

    div.dataset.date = dateKey;
    div.dataset.year = String(year);
    div.dataset.month = String(month);
    div.dataset.day = String(day);
    div.dataset.weekIndex = String(weekIndex);
    div.dataset.yearColor = yearColor;
    div.style.setProperty("--year-color", yearColor);

    gridEl.appendChild(div);
    weekSquares[weekIndex] = div;
  }

  // Highlight the current week (from grid origin = start of birth year).
  const now = new Date();
  const nowYear = now.getUTCFullYear();
  const nowMonth = now.getUTCMonth() + 1;
  const nowDay = now.getUTCDate();
  const nowYearOffset = nowYear - birthYear;

  if (nowYearOffset >= 0 && nowYearOffset < TOTAL_YEARS) {
    const nowWeekInYear = normalizedWeekInYear(nowYear, nowMonth, nowDay);
    const diffWeeks = nowYearOffset * WEEKS_PER_YEAR + nowWeekInYear;

    if (
      diffWeeks >= preBirthWeeks &&
      diffWeeks < weekSquares.length
    ) {
      weekSquares[diffWeeks].classList.add("current");
    }
  }
}

const MONTH_NAMES = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
];

function updateEventsPanel({
  detailDateEl,
  detailYearEl,
  detailMonthEl,
  detailWeekIndexEl,
  detailAgeEl,
  eventsEmptyEl,
  eventsGroupsEl,
  birthDate,
  target,
  eventIndex,
  dayEventsByWeekIndex,
}) {
  const dateKey = target.dataset.date;
  const year = Number(target.dataset.year);
  const month = Number(target.dataset.month);
  const day = Number(target.dataset.day);
  const weekIndex = Number(target.dataset.weekIndex);

  if (!dateKey || Number.isNaN(year) || Number.isNaN(month) || Number.isNaN(day))
    return;

  const displayDate = new Date(
    Date.UTC(year, month - 1, day, 0, 0, 0, 0),
  );

  const yearKey = String(year);
  const monthKey = `${year}-${pad2(month)}`;

  const yearEvents = eventIndex.byYear.get(yearKey) ?? [];
  const monthEvents = eventIndex.byMonth.get(monthKey) ?? [];
  const dayEvents = dayEventsByWeekIndex.get(weekIndex) ?? [];

  const ageLabel = computeAgeLabel(birthDate, displayDate);

  detailDateEl.textContent = dateKey;
  detailYearEl.textContent = String(year);
  if (detailMonthEl) detailMonthEl.textContent = MONTH_NAMES[month - 1] ?? "";
  detailWeekIndexEl.textContent = `Week #${weekIndex + 1} of ${TOTAL_YEARS * WEEKS_PER_YEAR}`;
  detailAgeEl.textContent = ageLabel ? `Age ${ageLabel}` : "";

  eventsGroupsEl.innerHTML = "";

  const hasAny =
    yearEvents.length > 0 || monthEvents.length > 0 || dayEvents.length > 0;

  eventsEmptyEl.style.display = hasAny ? "none" : "block";

  const appendGroup = (label, items) => {
    if (!items.length) return;

    const title = document.createElement("div");
    title.className = "event-group-title";
    title.textContent = label;
    eventsGroupsEl.appendChild(title);

    for (const item of items) {
      const eventEl = document.createElement("div");
      eventEl.className = "event-item";
      eventEl.textContent = item.text ?? "";
      eventsGroupsEl.appendChild(eventEl);
    }
  };

  appendGroup("Year", yearEvents);
  appendGroup("Month", monthEvents);
  appendGroup("Day", dayEvents);
}

function setupHover(
  gridEl,
  detailElements,
  birthDate,
  eventIndex,
  dayEventsByWeekIndex,
) {
  let lastTarget = null;

  gridEl.addEventListener("mouseover", (event) => {
    const target = event.target.closest(".week");
    if (!target || target === lastTarget || !gridEl.contains(target)) return;

    lastTarget = target;
    updateEventsPanel({
      ...detailElements,
      birthDate,
      target,
      eventIndex,
      dayEventsByWeekIndex,
    });
  });
}

function setupSizing(gridEl, zoomFactor = 1) {
  const baseSize = computeWeekSize(gridEl);
  const size = baseSize * zoomFactor;
  const wrapper = gridEl.closest("[data-role='grid-with-labels']") ?? gridEl.parentElement;
  if (wrapper) {
    wrapper.style.setProperty("--weekSize", `${size}px`);
    wrapper.style.setProperty("--zoom-factor", String(zoomFactor));
  }
  gridEl.style.setProperty("--weekSize", `${size}px`);
}

function updateZoomUI(zoomInEl, zoomOutEl, labelEl, zoom) {
  if (labelEl) {
    const rounded = Math.round(zoom * 10) / 10;
    labelEl.textContent =
      Number.isInteger(rounded) ? `${rounded}×` : `${rounded.toFixed(1)}×`;
  }
  if (zoomInEl) zoomInEl.disabled = zoom >= ZOOM_MAX;
  if (zoomOutEl) zoomOutEl.disabled = zoom <= ZOOM_MIN;
}

// --- Entry point ------------------------------------------------------------

window.addEventListener("DOMContentLoaded", async () => {
  const gridEl = document.querySelector("[data-role='life-grid']");
  if (!gridEl) return;

  let lifeConfig;
  try {
    const response = await fetch("./data.json");
    if (!response.ok) {
      throw new Error(`Failed to load data.json: ${response.status} ${response.statusText}`);
    }
    lifeConfig = await response.json();
  } catch (error) {
    console.error("Unable to load lifeConfig from data.json:", error);
    return;
  }

  const detailDateEl = document.querySelector("[data-role='detail-date']");
  const detailYearEl = document.querySelector("[data-role='detail-year']");
  const detailMonthEl = document.querySelector("[data-role='detail-month']");
  const detailWeekIndexEl = document.querySelector(
    "[data-role='detail-week-index']",
  );
  const detailAgeEl = document.querySelector("[data-role='detail-age']");
  const eventsEmptyEl = document.querySelector("[data-role='events-empty']");
  const eventsGroupsEl = document.querySelector("[data-role='events-groups']");
  const zoomInEl = document.querySelector("[data-role='zoom-in']");
  const zoomOutEl = document.querySelector("[data-role='zoom-out']");
  const zoomLabelEl = document.querySelector("[data-role='zoom-label']");

  if (
    !detailDateEl ||
    !detailYearEl ||
    !detailWeekIndexEl ||
    !detailAgeEl ||
    !eventsEmptyEl ||
    !eventsGroupsEl
  ) {
    console.warn("Detail panel elements not found; hover details will be limited.");
  }

  const birthDate = safeParseBirthDate(lifeConfig.birthDate);
  const eventIndex = buildEventIndex(lifeConfig);
  const dayEventsByWeekIndex = buildDayEventsByWeekIndex(eventIndex, birthDate);

  const yearLabelsEl = document.querySelector("[data-role='year-labels']");
  renderYearLabels(yearLabelsEl);
  renderWeeks(gridEl, eventIndex, dayEventsByWeekIndex, birthDate);
  setupSizing(gridEl, currentZoom);
  updateZoomUI(zoomInEl, zoomOutEl, zoomLabelEl, currentZoom);
  setupHover(
    gridEl,
    {
      detailDateEl: detailDateEl ?? { textContent: "" },
      detailYearEl: detailYearEl ?? { textContent: "" },
      detailMonthEl: detailMonthEl ?? null,
      detailWeekIndexEl: detailWeekIndexEl ?? { textContent: "" },
      detailAgeEl: detailAgeEl ?? { textContent: "" },
      eventsEmptyEl: eventsEmptyEl ?? { style: { display: "none" } },
      eventsGroupsEl: eventsGroupsEl ?? { innerHTML: "" },
    },
    birthDate,
    eventIndex,
    dayEventsByWeekIndex,
  );

  if (zoomInEl) {
    zoomInEl.addEventListener("click", () => {
      currentZoom = Math.min(
        ZOOM_MAX,
        Math.round((currentZoom + ZOOM_STEP) * 10) / 10,
      );
      setupSizing(gridEl, currentZoom);
      updateZoomUI(zoomInEl, zoomOutEl, zoomLabelEl, currentZoom);
    });
  }
  if (zoomOutEl) {
    zoomOutEl.addEventListener("click", () => {
      currentZoom = Math.max(
        ZOOM_MIN,
        Math.round((currentZoom - ZOOM_STEP) * 10) / 10,
      );
      setupSizing(gridEl, currentZoom);
      updateZoomUI(zoomInEl, zoomOutEl, zoomLabelEl, currentZoom);
    });
  }

  window.addEventListener("resize", () => setupSizing(gridEl, currentZoom));
});

