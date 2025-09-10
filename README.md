# Real-Time Chat Application

This project is a real-time chat application featuring a Vue.js frontend, a Golang backend, and Apache Kafka for message queuing. The application is designed with a microservices-inspired architecture and includes Keycloak for authentication.

## Architecture

The application is composed of the following services:

-   **Frontend**: A Vue.js single-page application that provides the user interface for the chat. It communicates with the backend via REST and WebSockets.
-   **Backend**: A Golang service that handles business logic, message persistence, and real-time communication. It uses WebSockets to broadcast messages to clients.
-   **Kafka**: An Apache Kafka instance that serves as a message broker, decoupling the backend from the real-time message broadcasting.
-   **Keycloak**: An Identity Provider for user authentication and authorization.

For a detailed explanation of the architecture, see [ChatApp.md](ChatApp.md).

## Project Structure

The project is organized into two main directories:

-   `backend/`: Contains the Golang backend service, including its Dockerfile and Helm chart.
-   `frontend/`: Contains the Vue.js frontend application, including its Dockerfile and Helm chart.

Each directory contains the source code, a `Dockerfile` for containerization, and a `chart/` directory with a Helm chart for Kubernetes deployment.

## Local Development

This project uses [Tilt](https://tilt.dev/) to manage the local development environment. Tilt automates the process of building and deploying the services to a local Kubernetes cluster.

### Prerequisites

Before you begin, make sure you have the following tools installed:

-   [Docker](https://www.docker.com/get-started)
-   [Kubernetes](https://kubernetes.io/docs/tasks/tools/) (e.g., via Docker Desktop, Minikube, or Kind)
-   [Tilt](https://docs.tilt.dev/install.html)
-   [Helm](https://helm.sh/docs/intro/install/)

### Running the Application

1.  **Start the development environment:**

    Run the following command in the root directory of the project:

    ```sh
    tilt up
    ```

    This command will:
    -   Deploy Kafka and Keycloak to the `chatapp` namespace.
    -   Build the Docker images for the backend and frontend.
    -   Deploy the backend and frontend services using their Helm charts.
    -   Set up port forwarding for all services.

2.  **Access the application:**

    Once `tilt up` has finished, you can access the services at the following URLs:

    -   **Frontend**: [http://localhost:8081](http://localhost:8081)
    -   **Backend API**: [http://localhost:8082](http://localhost:8082)
    -   **Keycloak Admin Console**: [http://localhost:8083](http://localhost:8083)

3.  **Keycloak Configuration:**

    -   **Admin Credentials**:
        -   **Username**: `admin`
        -   **Password**: `admin`
    -   **Realm**: `chat-app`
    -   **Client ID**: `frontend-client`

    The `chat-app` realm and `frontend-client` are pre-configured in the `Tiltfile`.

### Stopping the Application

To stop the local development environment, press `Ctrl+C` in the terminal where `tilt up` is running, and then run:

```sh
tilt down
```
