# Streamline

**Streamline** is a robust demo application that showcases a real-time event streaming server built using the Fiber framework and `net/http`. This demo is designed to demonstrate efficient handling of Server-Sent Events (SSE) and resource management, ensuring that resources are properly cleaned up after a client disconnects.

## Key Features

- **Real-Time Event Streaming**: Leverages Fiber's high-performance capabilities alongside `net/http` for handling Server-Sent Events, delivering real-time updates to clients efficiently.
  
- **Graceful Resource Management**: The server is engineered to handle client disconnections gracefully. Resources, including Kafka and Redis clients, are effectively cleaned up when a client disconnects, ensuring that no dead resources are left lingering.

- **Kafka and Redis Integration**: Integrates with Kafka and Redis for reliable event streaming and messaging, providing a robust backend for handling high-throughput and real-time data.

- **Dual Server Setup**: Demonstrates the use of both Fiber and `net/http` in a single application, with Fiber handling general routes and `net/http` managing specific routes for event streaming and patching.

- **Context-Aware Resource Cleanup**: Utilizes context cancellation to manage the lifecycle of streaming connections and background tasks, ensuring that all resources are properly released upon client disconnection.

## How It Works

- **Fiber Framework**: Used for handling general HTTP requests and routing, providing a fast and efficient web server environment.

- **`net/http` for SSE**: Employs `net/http` to manage SSE endpoints, offering detailed control over streaming and connection management.

- **Context Management**: Implements context-based error handling and resource cleanup to prevent issues such as resource leaks and ensure that disconnected clients do not leave lingering connections.

- **Resource Cleanup**: Automatically closes Kafka and Redis connections, and stops processing when the context is canceled, ensuring that no dead resources remain.

## Setup & Usage

1. **Configuration**: Load configuration settings for Kafka, Redis, and server ports from environment variables.

2. **Start Servers**: Run both Fiber and `net/http` servers concurrently, with Fiber handling standard routes and `net/http` managing SSE and event patching.

3. **Resource Management**: Observe how the application handles client disconnections and cleans up resources, ensuring that all processes terminate gracefully.

## Acknowledgements

- [Fiber](https://github.com/gofiber/fiber)
- [Gorilla Mux](https://github.com/gorilla/mux)
- [Kafka](https://kafka.apache.org/)
- [Redis](https://redis.io/)
