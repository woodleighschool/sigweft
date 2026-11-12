export const emailClients = {
  "outlook-web": {
    label: "Outlook Web",
    steps: [
      "Open Settings → Accounts → Signatures.",
      "Add a signature, name it and paste into the editor.",
      "Choose it for new messages and replies, then save.",
    ],
    href: "https://outlook.office.com/mail/options/accounts-category/signatures-subcategory",
    linkLabel: "Open Outlook Web",
    help: "https://support.microsoft.com/en-us/outlook/mail/how-to-add-and-change-an-email-signature-in-outlook",
  },
  "outlook-mac": {
    label: "Outlook for Mac",
    steps: [
      "Open Outlook → Settings → Signatures.",
      "Add a signature, name it and paste into the editor.",
      "Save it and select it for new messages and replies under your account.",
    ],
    href: "ms-outlook://",
    linkLabel: "Open Outlook",
    help: "https://support.microsoft.com/en-us/outlook/create-insert-or-change-a-signature-in-outlook-for-mac",
  },
  "outlook-windows": {
    label: "Outlook for Windows",
    steps: [
      "In new Outlook, open Settings → Accounts → Signatures. In classic Outlook, open File → Options → Mail → Signatures.",
      "Create a signature, name it and paste into the editor.",
      "Choose it for new messages and replies, then save.",
    ],
    help: "https://support.microsoft.com/en-us/outlook/mail/how-to-add-and-change-an-email-signature-in-outlook",
  },
  "apple-mail": {
    label: "Apple Mail",
    steps: [
      "Open Mail → Settings → Signatures and select your email account.",
      "Add a signature and turn off Always match my default message font.",
      "Paste into the editor, then select it in Choose Signature.",
    ],
    href: "x-help-action://openApp?bundleId=com.apple.mail",
    linkLabel: "Open Mail",
    help: "https://support.apple.com/guide/mail/change-signatures-settings-cpmlprefsig/mac",
  },
} satisfies Record<
  string,
  { label: string; steps: string[]; href?: string; linkLabel?: string; help: string }
>;

export function enabledClients(ids: string[]) {
  return [...new Set(ids)].flatMap((id) => {
    const entry = Object.entries(emailClients).find(([key]) => key === id);
    return entry ? [{ id, ...entry[1] }] : [];
  });
}
