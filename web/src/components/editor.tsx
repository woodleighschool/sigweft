import { useEffect, useMemo, useState } from "react";
import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useStore } from "@tanstack/react-store";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ConfiguredField } from "@components/configured-field";
import { Preview } from "@components/preview";
import { Alert, AlertDescription, AlertTitle } from "@components/ui/alert";
import { Button } from "@components/ui/button";
import { FieldGroup } from "@components/ui/field";
import { renderSignature } from "@lib/api-client/sdk.gen";
import type { Form, Identity } from "@lib/api-client/types.gen";
import { formSchema } from "@lib/form";
import { loadDraft, saveDraft } from "@lib/draft";

export function Editor({ metadata, user }: { metadata: Form; user?: Identity }) {
  const definitions = useMemo(() => metadata.fields ?? [], [metadata.fields]);
  const [initial] = useState(() => loadDraft(metadata, user));
  const schema = useMemo(() => formSchema(definitions), [definitions]);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "change" }),
    validators: { onMount: schema, onDynamic: schema },
    listeners: {
      onChange: ({ formApi }) => saveDraft(metadata, user, formApi.state.values),
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const valid = useStore(form.store, (state) => state.isValid);
  const inputKey = JSON.stringify(values);
  const [renderInput, setRenderInput] = useState({ key: inputKey, values });
  useEffect(() => {
    const timer = setTimeout(() => setRenderInput({ key: inputKey, values }), 200);
    return () => clearTimeout(timer);
  }, [inputKey, values]);
  const preview = useQuery({
    queryKey: ["signature", renderInput.values],
    queryFn: async ({ signal }) =>
      (
        await renderSignature({
          body: { values: renderInput.values },
          signal,
          throwOnError: true,
        })
      ).data,
    placeholderData: keepPreviousData,
    gcTime: 0,
    staleTime: Infinity,
    retry: false,
  });
  const current = inputKey === renderInput.key && !preview.isPlaceholderData;
  const pending = !current || preview.isFetching;
  const ready =
    valid &&
    current &&
    !preview.isFetching &&
    !preview.isError &&
    !!preview.data &&
    Object.keys(preview.data.errors).length === 0;

  return (
    <>
      <div className="mb-9 flex flex-col gap-2">
        <h1 className="text-3xl font-semibold tracking-tight">Create Your Email Signature</h1>
        <p className="text-muted-foreground">
          {metadata.description ||
            "Enter your details, check the preview, then copy your signature."}
        </p>
      </div>
      <div className="grid min-w-0 gap-10 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)] lg:gap-12">
        <section aria-labelledby="details-title" className="min-w-0">
          <div className="mb-5 flex min-h-8 items-center">
            <h2 id="details-title" className="text-lg font-semibold">
              Details
            </h2>
          </div>
          <form noValidate onSubmit={(event) => event.preventDefault()}>
            <FieldGroup>
              {definitions.map((definition) => (
                <form.Field key={definition.id} name={definition.id}>
                  {(field) => {
                    const serverError = current ? preview.data?.errors[definition.id] : undefined;
                    const errors = field.state.meta.errors.length
                      ? field.state.meta.errors
                      : serverError
                        ? [{ message: serverError }]
                        : [];
                    return (
                      <ConfiguredField
                        definition={definition}
                        placeholder={definition.placeholder}
                        value={field.state.value}
                        onChange={field.handleChange}
                        onBlur={field.handleBlur}
                        errors={field.state.meta.isDirty ? errors : []}
                      />
                    );
                  }}
                </form.Field>
              ))}
            </FieldGroup>
          </form>
        </section>
        <div className="flex min-w-0 flex-col gap-4 lg:contents">
          {preview.isError ? (
            <section className="flex flex-col gap-4">
              <Alert variant="destructive">
                <AlertTitle>Could not update the preview</AlertTitle>
                <AlertDescription>
                  Check your connection and try again. Your details are still in the form.
                </AlertDescription>
              </Alert>
              <Button variant="outline" onClick={() => void preview.refetch()}>
                Try Again
              </Button>
            </section>
          ) : (
            <Preview
              clients={metadata.email_clients ?? []}
              result={preview.data}
              ready={ready}
              copyHint={!valid ? "Fill in the remaining details to copy." : undefined}
              pending={pending}
              inputKey={inputKey}
            />
          )}
        </div>
      </div>
    </>
  );
}
