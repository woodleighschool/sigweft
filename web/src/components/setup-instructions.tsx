import { ArrowUpRightIcon } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@components/ui/tabs";
import { enabledClients } from "@lib/email-clients";

export function SetupInstructions({ clients }: { clients: string[] }) {
  const enabled = enabledClients(clients);
  if (!enabled.length) return null;
  return (
    <section aria-labelledby="setup-title">
      <h2 id="setup-title" className="mb-5 flex min-h-8 items-center text-lg font-semibold">
        3. Add to Your Email App
      </h2>
      <Tabs defaultValue={enabled[0].id}>
        <TabsList variant="line" className="h-auto flex-wrap justify-start">
          {enabled.map((client) => (
            <TabsTrigger key={client.id} value={client.id}>
              {client.label}
            </TabsTrigger>
          ))}
        </TabsList>
        {enabled.map((client) => (
          <TabsContent key={client.id} value={client.id} className="pt-3 text-muted-foreground">
            <ol className="flex list-decimal flex-col gap-2 pl-5">
              {client.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
            <div className="mt-4 flex flex-wrap gap-x-5 gap-y-2">
              {"href" in client && (
                <a
                  className="inline-flex items-center gap-1 text-primary underline underline-offset-4"
                  href={client.href}
                  target="_blank"
                  rel="noreferrer"
                >
                  {client.linkLabel}
                  <ArrowUpRightIcon aria-hidden="true" className="size-4" />
                </a>
              )}
              <a
                className="inline-flex items-center gap-1 underline underline-offset-4"
                href={client.help}
                target="_blank"
                rel="noreferrer"
              >
                Help
                <ArrowUpRightIcon aria-hidden="true" className="size-4" />
              </a>
            </div>
          </TabsContent>
        ))}
      </Tabs>
    </section>
  );
}
