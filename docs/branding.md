# Brand Studio

Each firm owns one `FirmBranding` profile: system title, display name, light/dark logo, favicon, optional login background, primary/secondary/accent color, navigation style, border-radius style, support details and portal language.

The authenticated bootstrap returns branding. React writes palette values into CSS variables and updates the browser title. Login, sidebar, header, dashboard and client portal consume those tokens and use safe fallbacks. LawOffice OS remains the repository/product name but is not the dominant end-user identity.

Brand images follow a separate upload path with size checks and an image-only MIME allowlist. Storage keys are generated internally. SVG is excluded from V0.1 because safely sanitizing active vector content would add a new security boundary. Replaced assets are retained until a future cleanup/retention job confirms they are unreferenced.
