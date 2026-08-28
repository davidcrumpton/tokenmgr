# Examples for Token Manager

Note these examples are provided to demonstrate how to use the Token Manager in different environments. They are not intended for production use and may require additional security measures or changes to meet your specific requirements. Always review and test the examples in your own environment before deployment.

## Node Red

This example shows how to use the Token Manager in Node Red. It demonstrates how to set up a flow that retrieves and manages tokens for API requests.  It contains two nodes: one validates the token structure and signature, and the other validates the expired time of the token. The flow is designed to be simple and easy to understand, making it a great starting point for developers looking to integrate token management into their Node Red applications.

## n8n

Import [`n8n-validate-token.json`](n8n-validate-token.json) into n8n. The workflow exposes a `POST /services/gitlab` Webhook, validates the Ed25519/EdDSA signature, checks the optional `exp` claim, and returns the same JSON response and HTTP status as the Node-RED example.  You may need to set `NODE_FUNCTION_ALLOW_BUILTIN` to `crypto` in your n8n deployment to allow the Code node to use the Node.js crypto builtin.

Before activating the workflow:

1. Replace `TOKENMGR_PUBKEY` in the **Validate Bearer token** Code node with the public key printed by `tokenmgr keygen` or `tokenmgr pubkey`.
2. Allow the Node.js `crypto` builtin for n8n Code nodes. For self-hosted n8n, set `NODE_FUNCTION_ALLOW_BUILTIN=crypto` according to your n8n deployment setup.
3. Activate the workflow and send the bearer token in the `Authorization` header to the webhook production URL.

## Ollama TLS nginx config

This example shows how to configure nginx to work with the Token Manager by front ending Ollama with HTTPS and requiring the Bearer token in the `Authorization` header. It demonstrates how to set up a reverse proxy that validates tokens for incoming API requests. The configuration includes directives to check the token's structure, signature, and expiration time, ensuring that only valid requests are forwarded to the backend server.
