import queryPlugin from "@tanstack/eslint-plugin-query";
import betterTailwindcss from "eslint-plugin-better-tailwindcss";

export default {
  // Generated sources and vendored components are not our lint policy.
  ignorePatterns: ["dist/**", "src/lib/api-client/**", "src/components/ui/**"],

  env: {
    builtin: true,
    browser: true,
  },

  options: {
    typeAware: true,
    reportUnusedDisableDirectives: "error",
    denyWarnings: true,
  },

  // Built-in Rust plugins use Oxlint categories rather than imported presets.
  plugins: ["eslint", "typescript", "unicorn", "oxc", "import", "react", "jsx-a11y", "promise"],

  jsPlugins: ["@tanstack/eslint-plugin-query", "eslint-plugin-better-tailwindcss"],

  categories: {
    correctness: "error",
    suspicious: "error",
  },

  settings: {
    "better-tailwindcss": {
      detectComponentClasses: true,
      entryPoint: "src/index.css",
      tsconfig: "tsconfig.json",
    },
  },

  rules: {
    // Use each JS plugin's ordinary recommended preset.
    ...queryPlugin.configs.recommended.rules,
    ...betterTailwindcss.configs.recommended.rules,

    // React 17+ automatic JSX transform.
    "react/react-in-jsx-scope": "off",

    // Vite loads global styles through a side-effect import in the entrypoint.
    "import/no-unassigned-import": ["error", { allow: ["**/*.css"] }],

    // Oxfmt owns source layout, including JSX attribute wrapping.
    "better-tailwindcss/enforce-consistent-line-wrapping": "off",
  },

  overrides: [
    {
      files: ["vite.config.ts"],
      env: {
        node: true,
      },
    },
  ],
};
