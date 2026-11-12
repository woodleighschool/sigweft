# AGENTS.md

Sigweft renders email signatures from external YAML, HTML templates and local or remote images. The Go server owns configuration, asset loading, final validation and rendering. React uses TanStack Form for local validation, browser draft storage and copying the rendered output.

- `internal/config` loads and validates organisation configuration. `sigweft.schema.json` is generated from its types.
- `internal/signature` resolves fields and renders the email artifact. It knows nothing about any particular organisation.
- `internal/account` owns browser sessions and uses the shared `goodies/auth/oidc` flow. Signed-in identity and editable signature fields are separate.
- `internal/server` exposes the form and rendering API. The OpenAPI document and TypeScript client are generated together.
- `web` contains the React app and the bundle embedded into the Go binary.
- `examples` contains editable configurations. People in examples and tests are fictional.

Use Mise tasks and pinned dependencies. Follow sibling repositories where useful, without copying unrelated features. Run relevant format, lint, test, generation and build checks. Keep generated files with the source change. Usage and configuration belong in README.md. Templates are trusted administrator input; field values are untrusted text.

The browser preview and clipboard use the same rendered HTML. A successful clipboard write does not prove Outlook embedded the images: verify received MIME before claiming that behaviour. Real organisation branding, configuration and staff correspondence belong outside version control; local deployments can use the ignored `config/` directory.

Use terse Conventional Commits. Do not commit, push or deploy unless asked.
