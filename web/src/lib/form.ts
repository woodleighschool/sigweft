import type { Field } from "./api-client/types.gen";
import { z } from "zod";

export type Value = string | string[] | boolean;
export type Values = Record<string, Value>;

function isValue(value: unknown): value is Value {
  return (
    typeof value === "string" ||
    typeof value === "boolean" ||
    (Array.isArray(value) && value.every((item) => typeof item === "string"))
  );
}

function fieldDefault(field: Field): Value {
  if (isValue(field.default)) return field.default;
  return field.type === "checkbox" ? false : field.type === "text-list" ? [] : "";
}

export function initialValues(fields: Field[], saved: Values = {}): Values {
  return Object.fromEntries(
    fields.map((field) => [
      field.id,
      field.read_only ? fieldDefault(field) : (saved[field.id] ?? fieldDefault(field)),
    ]),
  );
}

export function formSchema(fields: Field[]) {
  const shape: Record<string, z.ZodType<Value, Value>> = {};
  for (const field of fields) {
    const required = "This field is required.";
    if (field.type === "checkbox") {
      shape[field.id] = z.boolean().refine((value) => !field.required || value, required);
      continue;
    }
    const max = field.max_length ?? (field.type === "multiline" ? 1000 : 200);
    const text = z
      .string()
      .trim()
      .refine((value) => Array.from(value).length <= max, `Use at most ${max} characters.`)
      .refine(
        (value) =>
          !/\p{Cc}/u.test(
            field.type === "multiline"
              ? value.replaceAll("\r\n", "\n").replaceAll("\n", "")
              : value,
          ),
        "Remove unsupported control characters.",
      );
    if (field.type === "text-list") {
      shape[field.id] = z
        .array(text)
        .max(20, "Use at most 20 entries.")
        .refine((value) => !field.required || value.some(Boolean), required);
      continue;
    }
    let schema = text.refine((value) => !field.required || value !== "", required);
    if (field.type === "email") {
      schema = schema.refine(
        (value) => value === "" || z.regexes.html5Email.test(value),
        "Enter an email address.",
      );
    }
    if (field.type === "tel") {
      schema = schema.refine(
        (value) =>
          value === "" ||
          (/^\+?[0-9 ()-]+$/.test(value) &&
            value.replace(/\D/g, "").length >= 3 &&
            value.replace(/\D/g, "").length <= 15),
        "Enter a phone number using digits, spaces, +, brackets or hyphens.",
      );
    }
    if (field.type === "select") {
      schema = schema.refine(
        (value) => value === "" || field.options?.some((option) => option.value === value),
        "Choose an available option.",
      );
    }
    shape[field.id] = schema;
  }
  return z.object(shape);
}

export function clipboardContent(signature: { html: string; text: string }): Record<string, Blob> {
  return {
    "text/html": new Blob([signature.html], { type: "text/html" }),
    "text/plain": new Blob([signature.text], { type: "text/plain" }),
  };
}
