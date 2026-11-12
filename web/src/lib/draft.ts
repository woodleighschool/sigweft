import type { Form, Identity } from "./api-client/types.gen";
import { initialValues, type Values } from "./form.ts";

export function draftKey(form: Form, user?: Identity): string {
  return user
    ? `sigweft:${JSON.stringify([form.title, user.issuer, user.subject])}`
    : `sigweft:${form.title}`;
}

export function restoreDraft(form: Form, json: string | null, user?: Identity): Values {
  const prefills: Values = {};
  for (const field of form.fields ?? []) {
    if (field.prefill && user?.[field.prefill]) prefills[field.id] = user[field.prefill];
  }
  const defaults = initialValues(form.fields ?? [], prefills);
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
  return initialValues(form.fields ?? [], { ...prefills, ...values });
}

export function loadDraft(form: Form, user?: Identity): Values {
  try {
    return restoreDraft(form, localStorage.getItem(draftKey(form, user)), user);
  } catch {
    return restoreDraft(form, null, user);
  }
}

export function saveDraft(form: Form, user: Identity | undefined, values: Values): void {
  try {
    const key = draftKey(form, user);
    localStorage.setItem(key, JSON.stringify(values));
  } catch {
    // Storage can be unavailable in restricted browser sessions.
  }
}
