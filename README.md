# sigweft

[![Release](https://img.shields.io/github/v/release/woodleighschool/sigweft?display_name=tag&sort=semver)](https://github.com/woodleighschool/sigweft/releases/latest)
[![CI](https://github.com/woodleighschool/sigweft/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/woodleighschool/sigweft/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod/go-version/woodleighschool/sigweft?logo=go)](https://github.com/woodleighschool/sigweft/blob/main/go.mod)
[![Container](https://img.shields.io/badge/container-ghcr.io-2496ED?logo=github&logoColor=white)](https://github.com/orgs/woodleighschool/packages/container/package/sigweft)
[![License](https://img.shields.io/github/license/woodleighschool/sigweft)](https://github.com/woodleighschool/sigweft/blob/main/LICENSE)

Generates email signatures from a form, with fields, branding and layout defined in YAML and HTML templates.

> [!WARNING]
> This project may be unstable or have bugs, use with caution.
> Also expect breaking changes between releases for now.

## Usage

Copy the example configuration and edit its YAML, template and images:

```bash
mkdir -p config
cp -R examples/studio config/my-org
cp .env.example .env
```

Set the public URL and OIDC credentials in `.env` to require sign-in, then run the container with that directory mounted:

```bash
docker run --rm -p 8080:8080 \
  --env-file .env \
  --volume "$PWD/config/my-org:/config:ro" \
  --env SIGWEFT_CONFIG=/config/sigweft.yaml \
  ghcr.io/woodleighschool/sigweft:rolling
```

The server listens on port 8080. Terminate HTTPS at the reverse proxy; clipboard access requires HTTPS or localhost. The container runs as UID/GID 65532, which needs read access to the mounted files.

Enter your details, check the preview and copy into your email application's signature settings. Entries are saved in the current browser. The server does not store signature entries.

## Configuration

Each instance loads one YAML file at startup. Template and image paths are relative to that file and must stay within its directory. Validate configuration after editing, then restart the service:

```bash
sigweft validate --config config/my-org/sigweft.yaml
sigweft serve --config config/my-org/sigweft.yaml
```

| Variable         | Default       | Purpose             |
| ---------------- | ------------- | ------------------- |
| `SIGWEFT_CONFIG` | `config.yaml` | Configuration file  |
| `SIGWEFT_LISTEN` | `:8080`       | HTTP listen address |

`--config` overrides `SIGWEFT_CONFIG`. `/healthz` is the health endpoint.

The [example YAML](examples/studio/sigweft.yaml) and [template](examples/studio/signature.html) show location choices, multiple roles, working hours, conditional content and images. The [JSON Schema](sigweft.schema.json) provides editor completion. Unknown keys and duplicate mappings are rejected.

### Authentication

SSO uses OpenID Connect. Set the issuer URL, client ID and client secret together to require sign-in. Leaving all three unset allows public access; partial configuration fails at startup.

| Variable                        | Default                                | Purpose                                 |
| ------------------------------- | -------------------------------------- | --------------------------------------- |
| `SIGWEFT_URL`                   | empty                                  | Public browser URL                      |
| `SIGWEFT_SESSION_COOKIE_SECURE` | `true`                                 | Send the session cookie over HTTPS only |
| `SIGWEFT_OIDC_ISSUER_URL`       | empty                                  | Identity provider's issuer URL          |
| `SIGWEFT_OIDC_CLIENT_ID`        | empty                                  | OAuth client ID                         |
| `SIGWEFT_OIDC_CLIENT_SECRET`    | empty                                  | OAuth client secret                     |
| `SIGWEFT_OIDC_REDIRECT_URL`     | `${SIGWEFT_URL}/api/auth/sso/callback` | Registered callback URL                 |
| `SIGWEFT_OIDC_SCOPES`           | `openid,email,profile`                 | Requested scopes, comma-separated       |
| `SIGWEFT_OIDC_EMAIL_CLAIM`      | `email`                                | Claim used to prefill email fields      |

For Entra, use a tenant-specific issuer (`https://login.microsoftonline.com/<tenant-id>/v2.0`). Register the callback as a Web redirect URI, require assignment on the enterprise application, and assign the staff group. Tenant membership alone does not establish staff access. [Entra application assignment](https://learn.microsoft.com/en-us/entra/identity-platform/howto-restrict-your-app-to-a-set-of-users) controls admission; Graph permissions and directory sync are not required.

Set `prefill: name` or `prefill: email` on an editable text or email field to use the corresponding identity claim. Saved edits take precedence. Staff can enter a shared mailbox's name and address without changing their signed-in identity. Missing claims leave the configured default or an empty field.

Drafts are stored in the browser per signed-in account. Sessions expire after 14 days and are held in memory, so this setup uses one server instance and restarts require sign-in again. Assignment changes take effect at the next sign-in. Log Out ends the application's session; the identity provider manages its own session.

For local HTTP development, set `SIGWEFT_SESSION_COOKIE_SECURE=false` and register `http://localhost:5177/api/auth/sso/callback`. Production uses HTTPS.

### Fields

Fields have an `id`, `label` and `type`. IDs use lowercase letters, digits and underscores, starting with a letter.

| Type        | Control            |
| ----------- | ------------------ |
| `text`      | Single line        |
| `email`     | Email address      |
| `tel`       | Phone number       |
| `multiline` | Text area          |
| `text-list` | Add/remove entries |
| `select`    | Configured choices |
| `checkbox`  | Checkbox           |

Use `required`, `default`, `description` and `placeholder` where needed. `read_only` fixes a field to its configured default. `max_length` defaults to 200 characters per entry, or 1,000 for multiline fields. Text lists accept up to 20 entries.

Select fields have `options`, each with a `value`, `label` and optional `data`. Use option data for location addresses and phone numbers; these do not need editable fields. The browser submits the selected value and the server resolves its data.

`email_clients` enables setup instructions in display order: `outlook-web`, `outlook-mac`, `outlook-windows`, `apple-mail`. Omit it to hide the instructions.

### Templates and images

Templates use Go `html/template`. Field values are escaped and validated before rendering.

| Value           | Contents                          |
| --------------- | --------------------------------- |
| `.Fields.<id>`  | Field value                       |
| `.Choices.<id>` | Selected option's data            |
| `.Data`         | Top-level YAML data               |
| `.Assets.<id>`  | Image source, dimensions and link |

Use `if`, `with` and `range` for optional content and lists. `upper` uppercases text, `lines` splits multiline text and `tel` builds a phone link. Missing template keys are errors. Use inline styles and presentation tables suitable for email.

Assets have `file`, `alt`, `width`, `height` and an optional `href`. Images must be PNG, JPEG or GIF, at most 2 MiB each. The display dimensions can differ from the source dimensions. Image bytes are included in the signature; no image server is required.

Keep deployment-specific configuration, branding and staff correspondence outside version control. The local `config/` directory is ignored and excluded from container builds.

### Email compatibility

The clipboard contains HTML and plain text. In the tested Chrome to Outlook for Mac path, PNG images became inline CID attachments. Apple Mail kept data URLs, which some recipients may reject. These results do not establish compatibility with other clients.

After changing a template, paste into the signature editor, save and reopen it, then send a message and inspect the received copy. Check images, spacing and links in light and dark mode. The browser preview cannot reproduce an email client's colour changes.

## Development

Mise owns the toolchain and commands:

```bash
mise install
mise run deps
mise run dev
mise run test
mise run lint
mise run check
```

Open <http://localhost:5177>. The default configuration is the fictional North Studio example. Export values from `.env` when testing SSO locally; the app reads environment variables. Use `SIGWEFT_CONFIG=config/my-org/sigweft.yaml mise run dev` for a local deployment.

Backend code lives under `cmd/sigweft` and `internal/`; the React app lives under `web/`. The built Go binary embeds the web bundle.

Run `mise run generate` after changing configuration types or the API contract. `mise run frontend-check` checks generated files, formatting, frontend lint and the build. Go tests use synthetic identities and a local OIDC provider.

Use `mise run build` for the binary or `mise run container-build` for a local container. `mise run format` formats the source; `mise run vulncheck` checks Go dependencies.

## License

Licensed under the [Apache License 2.0](LICENSE).
