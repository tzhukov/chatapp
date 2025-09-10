# **Chat Application Design Document**

## **1\. Introduction**

This document outlines the design for a simple, real-time chat application. The primary goal is to create a robust and scalable system with clear, well-defined interfaces for agentic development. The architecture is composed of three main components: a Vue.js frontend, a Golang backend service, and an Apache Kafka messaging queue. A new, dedicated Identity Provider (IdP) will be added to handle all user authentication.

## **2\. System Architecture**

The chat application follows a microservices-inspired architecture with a clear separation of concerns.  
**Identity Provider (DexIdp):** A dedicated service responsible for user registration, login, and authorization. It will issue JSON Web Tokens (JWT) upon successful authentication, which the other services will use to verify user identity. DexIdp is an OpenID Connect (OIDC) provider that can be used to federate authentication against other upstream identity providers like LDAP, GitHub, etc.  
**Frontend (Vue.js):** A single-page application responsible for the user interface, rendering the chat history, and sending new messages. It communicates with the Golang backend via a REST API for initial data and a WebSocket connection for real-time updates. The frontend will redirect users to the IdP for login and attach the received JWT to all requests.  
**Backend (Golang):** A stateless service that handles API requests and message persistence. It is now responsible for validating the JWTs received from the frontend to ensure requests are from an authenticated user. It exposes REST and WebSocket endpoints. For scalability, the service acts as a producer to a Kafka topic for new messages and also consumes from Kafka to broadcast messages to connected clients via WebSockets.  
**Message Queue (Apache Kafka):** Serves as the central nervous system for real-time communication. It decouples the backend logic from the real-time broadcasting, ensuring that messages are reliably delivered and can be consumed by multiple services if needed in the future.

## **3\. Component Design**

### **3.1 Frontend (Vue.js)**

The frontend will be built with Vue 3\. The main components include:

* **ChatWindow.vue:** Displays the list of messages. It will receive messages from the WebSocket connection and append them to the chat history.  
* **MessageInput.vue:** A component with a text area and a "send" button. It will emit a message event to its parent component, which will then use the backend API to send the message.  
* **UserList.vue:** Displays a list of online users, although this feature is out of scope for the initial MVP.

The frontend will have a ChatService interface to handle all backend communication, ensuring the UI layer is completely decoupled from the data fetching and sending logic. The login process will now redirect the user to DexIdp, and a successful login will return a JWT that the frontend will store locally.

### **3.2 Backend (Golang)**

The Golang service is the core of the application logic.  
Core Responsibilities:

* Expose a REST API for message persistence and retrieval.  
* Provide a WebSocket endpoint for real-time communication.  
* Validate JWTs from incoming requests to authenticate users.  
* Act as a **Kafka Producer** for new messages.  
* Act as a **Kafka Consumer** to receive messages and broadcast them to all connected WebSocket clients.

### **3.3 Identity Provider (DexIdp)**

We will use DexIdp as the Identity Provider. Its primary role is to manage user identities and issue tokens that represent a user's authenticated session.  
Key Responsibilities:

* **User Management:** Handles user registration, password resets, and profile management (often by delegating to upstream providers).  
* **Authentication:** Manages the login process for users using OpenID Connect.  
* **Authorization:** Issues short-lived access tokens (JWT) and refresh tokens.  
* **Integration:** We will configure DexIdp as an OpenID Connect client for our frontend, defining the redirect\_uri for successful authentication callbacks. The Golang backend will be configured to validate tokens issued by this DexIdp instance by fetching the public key to verify the signature of the JWT.

### **3.4 Message Queue (Apache Kafka)**

Kafka is used to handle the real-time message flow.

* **Topic:** A single Kafka topic, chat-messages, will be used.  
* **Message Schema:** Messages on this topic will adhere to the Message interface defined in Section 4.2.  
* **Exchange Diagram:**

The diagram illustrates the message flow with the new authentication component:

1. A user attempts to access the frontend, which redirects them to DexIdp for authentication.  
2. Upon successful login, DexIdp redirects the user back to the frontend with an authorization code.  
3. The frontend exchanges the code for an access token (JWT) and a refresh token.  
4. The user sends a message from the Vue.js frontend, attaching the JWT to the Authorization header.  
5. The frontend makes a POST request to the Golang backend.  
6. The Golang backend validates the JWT, and upon success, persists the message to the database and publishes the message payload to the chat-messages Kafka topic.  
7. The Golang service also has a consumer running in parallel that listens to the chat-messages topic.  
8. Upon receiving a message from Kafka, the consumer broadcasts the message to all connected WebSocket clients.

## **4\. Interface Definitions**

### **4.1 OpenAPI Specification for the Golang Service**

The OpenAPI 3.0 specification defines the contract for the REST and WebSocket APIs. The new securitySchemes and security definitions now enforce JWT-based authentication.  
openapi: 3.0.0  
info:  
  title: Chat Service API  
  version: 1.0.0  
  description: API for the chat application backend service.  
servers:  
  \- url: \[https://ingress.local/v1\](https://ingress.local/v1)  
tags:  
  \- name: messages  
    description: Operations related to chat messages.  
paths:  
  /messages:  
    get:  
      tags:  
        \- messages  
      summary: Retrieve message history  
      description: Returns a list of all messages in the chat history.  
      operationId: getMessages  
      security:  
        \- bearerAuth: \[\]  
      responses:  
        '200':  
          description: A list of messages.  
          content:  
            application/json:  
              schema:  
                type: array  
                items:  
                  $ref: '\#/components/schemas/Message'  
    post:  
      tags:  
        \- messages  
      summary: Send a new message  
      description: Sends a new chat message to the system.  
      operationId: sendMessage  
      security:  
        \- bearerAuth: \[\]  
      requestBody:  
        required: true  
        content:  
          application/json:  
            schema:  
              type: object  
              properties:  
                user\_id:  
                  type: string  
                  description: The ID of the user sending the message.  
                content:  
                  type: string  
                  description: The message text.  
      responses:  
        '201':  
          description: Message successfully sent.  
          content:  
            application/json:  
              schema:  
                $ref: '\#/components/schemas/Message'  
        '400':  
          description: Invalid request body.  
components:  
  schemas:  
    Message:  
      type: object  
      properties:  
        message\_id:  
          type: string  
          description: Unique identifier for the message.  
        user\_id:  
          type: string  
          description: The ID of the user who sent the message.  
        content:  
          type: string  
          description: The message text.  
        timestamp:  
          type: string  
          format: date-time  
          description: The timestamp when the message was created.  
  securitySchemes:  
    bearerAuth:  
      type: http  
      scheme: bearer  
      bearerFormat: JWT

### **4.2 OpenID Connect Authentication Flow**

This section details the user login and registration process using the OpenID Connect Authorization Code Flow with DexIdp. The flow is designed to be secure and leverages standard protocols.

#### **User Registration**

1. **Redirect to DexIdp:** When a new user needs to sign up, the frontend will redirect them to a specific registration URL provided by DexIdp. For example, https://dexidp.local/sign-up.  
2. **User Input:** The user fills out the registration form on the DexIdp-hosted page.  
3. **DexIdp Processes:** DexIdp handles the creation of the new user account and any associated profile information.  
4. **Redirect and Login:** Upon successful registration, DexIdp automatically logs the user in and initiates the standard login flow (detailed below) by redirecting them back to the frontend with an authorization code.

#### **User Login (Authorization Code Flow)**

1. **Initiate Flow:** The frontend's login button triggers a redirect to DexIdp's authorization endpoint. This request includes several parameters:  
   * client\_id: The public identifier for the chat application frontend.  
   * redirect\_uri: The specific URL on the frontend where DexIdp should send the user back after authentication (e.g., https://ingress.local/auth/callback).  
   * response\_type: Set to code to indicate the Authorization Code Flow.  
   * scope: A list of permissions requested, typically including openid and profile.  
2. **User Authentication:** The user is presented with the DexIdp login page and enters their credentials.  
3. **Authorization Code Grant:** Upon successful authentication, DexIdp sends a redirect to the redirect\_uri provided in step 1, appending a one-time use code as a query parameter.  
4. **Token Exchange:** The frontend, upon receiving the code, makes a direct, server-to-server (or in-browser for single-page apps) POST request to DexIdp's token endpoint. This request includes the code, client\_id, client\_secret (for public clients, this may be omitted but is recommended for a server-side component), and redirect\_uri.  
5. **Token Issuance:** DexIdp validates the request and, if successful, responds with an access token (JWT), an ID token (JWT), and a refresh token.  
   * **Access Token:** Used to authenticate API calls to the Golang backend.  
   * **ID Token:** Contains basic user profile information.  
   * **Refresh Token:** Used to obtain a new access token without requiring the user to log in again.  
6. **Authenticated Requests:** The frontend stores the access token and includes it in the Authorization header of all subsequent API calls to the Golang backend (e.g., Authorization: Bearer \<access\_token\>).

### **4.3 Frontend Interfaces**

These TypeScript interfaces define the data structures for the frontend, making development with Vue.js more predictable. The ChatService will now need to handle authentication and token management.  
// src/types/message.ts  
export interface Message {  
  message\_id: string;  
  user\_id: string;  
  content: string;  
  timestamp: string; // ISO 8601 format  
}

// src/services/chatService.ts  
export interface ChatService {  
  getMessages(): Promise\<Message\[\]\>;  
  sendMessage(message: { user\_id: string, content: string }): Promise\<Message\>;  
  connectWebSocket(onMessage: (message: Message) \=\> void): WebSocket;  
  login(): void; // Added for redirection to IdP  
  logout(): void; // Added for user logout  
  getToken(): string | null; // Added to retrieve the stored token  
}

## **5\. Service Configuration for DexIdp Integration**

### **5.1 Frontend (Vue.js) OIDC Configuration**

The frontend, acting as a public OpenID Connect client, needs to be configured with the following parameters, which would typically be loaded from environment variables.

* DEX\_ISSUER\_URL: https://ingress.local/dex \- The base URL of the DexIdp instance.  
* DEX\_CLIENT\_ID: chat-app-frontend \- The unique identifier for the frontend application registered with DexIdp.  
* DEX\_REDIRECT\_URI: https://ingress.local/dex/auth/callback \- The URI where DexIdp will redirect the user after authentication. This must match a URI registered with the DexIdp client.  
* DEX\_SCOPES: openid profile \- The list of requested permissions. openid is mandatory for OIDC, and profile grants access to basic user information.

These values will be used to construct the authorization URL for the login redirect and for the token exchange request. A library such as oidc-client-js or similar can simplify this process.

### **5.2 Backend (Golang) JWT Validation Configuration**

The Golang backend service is configured to securely validate incoming JWTs from the frontend. This validation is done on every authenticated request to ensure the token is legitimate and has not been tampered with.

1. **Discovery:** The backend will fetch the OpenID Connect discovery document from DexIdp's well-known configuration endpoint, usually found at https://dexidp.local/dex/.well-known/openid-configuration. This document contains critical information like the issuer URL, token endpoint, and most importantly, the JWKS URI.  
2. **JWKS Fetch:** The backend will then fetch the JSON Web Key Set (JWKS) from the URI provided in the discovery document. The JWKS contains the public keys that correspond to the private keys DexIdp uses to sign its JWTs.  
3. **JWT Validation:** For each incoming API request with a Bearer token, the backend performs the following checks:  
   * **Signature Verification:** The token's signature is verified using the public key from the JWKS. This proves the token was issued by DexIdp and has not been altered.  
   * **exp (Expiration) Claim:** The token's expiration time is checked to ensure it is still valid.  
   * **iss (Issuer) Claim:** The issuer claim in the JWT is verified against the expected issuer URL (https://dexidp.local).  
   * **aud (Audience) Claim:** The audience claim is checked to ensure the token is intended for this specific backend service.

A Golang library for JWTs (e.g., go-jwt or oidc) can automate these validation steps. The backend service's configuration will need to specify the DEX\_ISSUER\_URL and DEX\_AUDIENCE values to perform these checks.

## **6\. Conclusion**

This updated design document now provides the necessary configuration details for integrating the chat application's frontend and backend with DexIdp. By clearly defining these parameters and validation steps, we can ensure that the developer agent can securely connect the services and establish a robust authentication system.  
Let me know if you would like to dive deeper into the specific configuration for DexIdp, such as setting up connectors or client registrations.