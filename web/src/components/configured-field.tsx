import { PlusIcon, XIcon } from "lucide-react";
import { Button } from "@components/ui/button";
import { Checkbox } from "@components/ui/checkbox";
import { Field, FieldDescription, FieldError, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@components/ui/select";
import { Textarea } from "@components/ui/textarea";
import type { Field as Definition } from "@lib/api-client/types.gen";
import type { Value } from "@lib/form";

type Props = {
  definition: Definition;
  value: Value;
  onChange: (value: Value) => void;
  onBlur: () => void;
  errors?: Array<{ message?: string } | undefined>;
  placeholder?: string;
};

export function ConfiguredField({
  definition: d,
  value,
  onChange,
  onBlur,
  errors,
  placeholder,
}: Props) {
  const error = !!errors?.length;
  const id = `field-${d.id}`;
  const describedBy =
    [d.description ? `${id}-description` : "", error ? `${id}-error` : ""]
      .filter(Boolean)
      .join(" ") || undefined;
  const shared = {
    id,
    name: d.id,
    "aria-required": d.required,
    "aria-invalid": error,
    "aria-describedby": describedBy,
    onBlur,
  };
  let control;
  switch (d.type) {
    case "select":
      control = (
        <Select
          value={typeof value === "string" && value !== "" ? value : null}
          items={d.options ?? []}
          onValueChange={(next) => onChange(next ?? "")}
          disabled={d.read_only}
        >
          <SelectTrigger {...shared} className="w-full">
            <SelectValue placeholder={placeholder || "Choose an option"} />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {!d.required && <SelectItem value="">None</SelectItem>}
              {d.options?.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      );
      break;
    case "checkbox":
      control = (
        <Checkbox
          {...shared}
          checked={value === true}
          onCheckedChange={onChange}
          disabled={d.read_only}
        />
      );
      break;
    case "multiline":
      control = (
        <Textarea
          {...shared}
          value={typeof value === "string" ? value : ""}
          onChange={(event) => onChange(event.target.value)}
          readOnly={d.read_only}
          maxLength={d.max_length}
          placeholder={placeholder}
          rows={3}
        />
      );
      break;
    case "text-list": {
      const entries = Array.isArray(value) && value.length ? value : [""];
      control = (
        <div className="flex flex-col gap-2">
          {entries.map((entry, index) => (
            <div key={index} className="flex gap-2">
              <Input
                {...shared}
                id={index === 0 ? id : `${id}-${index}`}
                aria-label={`${d.label} ${index + 1}`}
                value={entry}
                readOnly={d.read_only}
                maxLength={d.max_length}
                placeholder={placeholder}
                onChange={(event) =>
                  onChange(entries.map((item, i) => (i === index ? event.target.value : item)))
                }
              />
              {entries.length > 1 && !d.read_only && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`Remove ${d.label.toLowerCase()} ${index + 1}`}
                  onClick={() => onChange(entries.filter((_, i) => i !== index))}
                >
                  <XIcon />
                </Button>
              )}
            </div>
          ))}
          {!d.read_only && entries.length < 20 && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="self-start"
              onClick={() => onChange([...entries, ""])}
            >
              <PlusIcon data-icon="inline-start" />
              Add Another
            </Button>
          )}
        </div>
      );
      break;
    }
    default:
      control = (
        <Input
          {...shared}
          type={d.type}
          value={typeof value === "string" ? value : ""}
          onChange={(event) => onChange(event.target.value)}
          readOnly={d.read_only}
          maxLength={d.max_length}
          placeholder={placeholder}
        />
      );
  }
  return (
    <Field data-invalid={error} data-disabled={d.read_only}>
      <FieldLabel htmlFor={id}>
        <span>
          {d.label}
          {d.required && (
            <span className="text-destructive" aria-hidden="true">
              *
            </span>
          )}
        </span>
      </FieldLabel>
      {control}
      {d.description && (
        <FieldDescription id={`${id}-description`}>{d.description}</FieldDescription>
      )}
      {error && <FieldError id={`${id}-error`} errors={errors} />}
    </Field>
  );
}
