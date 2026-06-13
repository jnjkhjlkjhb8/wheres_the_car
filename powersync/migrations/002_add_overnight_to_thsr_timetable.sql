ALTER TABLE thsr_timetable
  ADD COLUMN IF NOT EXISTS overnight boolean NOT NULL DEFAULT false;
