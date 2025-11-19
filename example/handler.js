// Example handler function for GitHub webhook events
// This function is called after the webhook signature is validated
//
// Function signature: async function handle(githubEvent, payload, env)
//
// Parameters:
//   - githubEvent: string - The GitHub event type (e.g., "push", "pull_request")
//   - payload: object - The parsed JSON payload from GitHub
//   - env: object - Environment variables configured in the worker
//
// Returns:
//   - object - Response object that will be returned to GitHub

async function handle(githubEvent, payload, env) {
    console.log(`Received GitHub event: ${githubEvent}`);

    // Example: Handle push events
    if (githubEvent === "push") {
        const branch = payload.ref.replace('refs/heads/', '');
        const commits = payload.commits.length;
        const pusher = payload.pusher.name;

        console.log(`Push to ${branch} by ${pusher}: ${commits} commit(s)`);

        // Add your custom logic here
        // For example: trigger a deployment, send a notification, etc.

        return {
            status: "success",
            message: `Processed ${commits} commit(s) on ${branch}`,
            event: githubEvent
        };
    }

    // Example: Handle pull request events
    if (githubEvent === "pull_request") {
        const action = payload.action;
        const prNumber = payload.pull_request.number;
        const prTitle = payload.pull_request.title;

        console.log(`Pull request #${prNumber} ${action}: ${prTitle}`);

        // Add your custom logic here

        return {
            status: "success",
            message: `Processed PR #${prNumber} (${action})`,
            event: githubEvent
        };
    }

    // Default handler for other events
    return {
        status: "received",
        message: `Event ${githubEvent} acknowledged`,
        event: githubEvent
    };
}
