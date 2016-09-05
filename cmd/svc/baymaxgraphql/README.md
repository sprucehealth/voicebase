Baymax GraphQL API
==================
The Baymax GraphQL service is responsible for the external API communication via GraphQL for baymax. There are also a couple of restful endpoints for managing media objects. You can introspect our schema definition by running: https://dev-baymax-api.carefront.net/schema at any time. 

This service communicates with almost all of the other services in the network to serve the necessary data to clients.

GraphQL References
------------------

GraphQL:

- Spec: https://facebook.github.io/graphql/
- Pagination at Facebook: https://github.com/facebook/graphql/issues/4#issuecomment-118162627
- Go pkg: https://github.com/graphql-go/graphql
- Go pkg docs: https://godoc.org/github.com/graphql-go/graphql
- Related to relay:
    - Object ID Spec: https://facebook.github.io/relay/graphql/objectidentification.htm
    - Mutation Spec: https://facebook.github.io/relay/graphql/mutations.htm
    - Cursor Connections Spec: https://facebook.github.io/relay/graphql/connections.htm
    - Spec: https://facebook.github.io/relay/docs/graphql-relay-specification.html

Example reference APIs:

- Slack: https://api.slack.com/methods
    - Real-time: https://api.slack.com/rtm
    - Channel history: https://api.slack.com/methods/channels.history
- AWS for idempotent requests: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html

WebSocket libraries:

- iOS: https://github.com/square/SocketRocket
- Go: https://github.com/gorilla/websocket (seems more full-featured than /x/net/websocket)
