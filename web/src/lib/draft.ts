import type { Form } from "./api-client/types.gen";
import { initialValues, type Values } from "./form.ts";

export function restoreDraft(form: Form, json: string | null): Values {
  const defaults = initialValues(form.fields ?? []);
  const values: Values = {};
  if (!json) return defaults;
  let saved: unknown;
  try {
    saved = JSON.parse(json);
  } catch {
    return defaults;
  }
  if (!saved || typeof saved !== "object" || Array.isArray(saved)) return defaults;
  for (const field of form.fields ?? []) {
    if (field.read_only || !(field.id in saved)) continue;
    const value: unknown = Reflect.get(saved, field.id);
    if (field.type === "checkbox") {
      if (typeof value === "boolean") values[field.id] = value;
    } else if (field.type === "text-list") {
      if (Array.isArray(value) && value.every((item) => typeof item === "string"))
        values[field.id] = value;
    } else if (typeof value === "string") {
      values[field.id] = value;
    }
  }
  return initialValues(form.fields ?? [], values);
}

export function loadDraft(form: Form): Values {
  try {
    return restoreDraft(form, localStorage.getItem(`sigweft:${form.title}`));
  } catch {
    return initialValues(form.fields ?? []);
  }
}

export function saveDraft(title: string, values: Values): void {
  try {
    const key = `sigweft:${title}`;
    localStorage.setItem(key, JSON.stringify(values));
  } catch {
    // Storage can be unavailable in restricted browser sessions.
  }
}
