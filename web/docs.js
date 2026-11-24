// Renders the API reference from the contract with Swagger UI, loaded from
// jsDelivr at a pinned version with subresource integrity (see docs.html).
"use strict";

window.addEventListener("DOMContentLoaded", () => {
  window.SwaggerUIBundle({
    url: "openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    defaultModelsExpandDepth: 0,
    // No badge from the online validator: it would send this page's address
    // to a third party, and CI already lints the contract.
    validatorUrl: null,
  });
});
