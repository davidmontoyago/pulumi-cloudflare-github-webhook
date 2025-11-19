export default {
    async fetch(request, env, ctx) {
        if (request.method !== "POST") {
            return new Response('{"error": "method not allowed"}', {
                status: 405,
                headers: {
                    'content-type': 'application/json',
                },
            });
        }

        let headers = request.headers;
        const githubWebhookSignature = headers.get('x-hub-signature-256');

        if (!githubWebhookSignature) {
            return new Response('{"error": "signature header is required"}', {
                status: 400,
                headers: {
                    'content-type': 'application/json',
                },
            });
        }

        const githubEvent = headers.get('x-github-event');
        if (!githubEvent) {
            return new Response('{"error": "event name is missing"}', {
                status: 400,
                headers: {
                    'content-type': 'application/json',
                },
            });
        }

        // check the payload signature
        const payloadJson = await request.clone().json();
        const payload = JSON.stringify(payloadJson);
        let valid = false;
        try {
            valid = await verifyGithubWebhookSignature(env.GITHUB_WEBHOOK_SECRET, githubWebhookSignature, payload);
        } catch (error) {
            return new Response('{"error": "Forbidden. Definitely not gonna happen with that signature."}', {
                status: 403,
                headers: {
                    'content-type': 'application/json',
                },
            });
        }
        if (!valid) {
            return new Response('{"error": "Forbidden. Not gonna happen with that signature."}', {
                status: 403,
                headers: {
                    'content-type': 'application/json',
                },
            });
        }

        // invoke the user provided handler
        const response = await handle(githubEvent, payloadJson, env);

        return new Response(
            JSON.stringify(response),
            {
                headers: {
                    'content-type': 'application/json',
                },
            });
    },
};

const encoder = new TextEncoder()

async function verifyGithubWebhookSignature(secret, header, payload) {
    let parts = header.split("=");
    let sigHex = parts[1];

    let algorithm = { name: "HMAC", hash: { name: 'SHA-256' } };

    let keyBytes = encoder.encode(secret);
    let extractable = false;
    let key = await crypto.subtle.importKey(
        "raw",
        keyBytes,
        algorithm,
        extractable,
        ["sign", "verify"],
    );

    let sigBytes = hexToBytes(sigHex);
    let dataBytes = encoder.encode(payload);
    let equal = await crypto.subtle.verify(
        algorithm.name,
        key,
        sigBytes,
        dataBytes,
    );

    return equal;
}

function hexToBytes(hex) {
    let len = hex.length / 2;
    let bytes = new Uint8Array(len);

    let index = 0;
    for (let i = 0; i < hex.length; i += 2) {
        let c = hex.slice(i, i + 2);
        let b = parseInt(c, 16);
        bytes[index] = b;
        index += 1;
    }

    return bytes;
}
