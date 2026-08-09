# Client Portal

The portal is a separate layout and identity boundary. `PortalUser` credentials are hashed and sessions use a distinct HttpOnly cookie path. `PortalAccess` links that identity to selected Matters and flags which classes of information may be exposed.

The portal never returns a complete internal Matter. Responses reduce metadata to the allowed summary and include only `clientVisible` events/documents. Internal notes, confidential fields, Matter access rules, finance and audit are not portal data. A client cannot select another Matter merely by changing its ID.

Brand Studio supplies the office logo, colors, portal title and welcome text. V0.1 supports selected Matter and timeline views. Invitation delivery, password recovery, document preview and richer sharing administration are known follow-up work.
