# 🛡️ Echo Backend: The Silent Sentinel

Welcome to the core of the Echo Messaging Platform. This backend is engineered for **Security**, **Scalability**, and **Developer Happiness**. It uses a professional "Repository Pattern" architecture that ensures 100% unit testability.

---

## 🚀 Quick Start (Local Development)

### 1. Prerequisites
- **Go 1.23+** installed.
- **MongoDB** (Local or Atlas) running.
- **Redis** (Optional for now, required for live presence).

### 2. Configuration
Copy the `.env` file and fill in your secrets:
```bash
# PostgreSQL (Optional for legacy support)
DB_HOST=localhost
DB_NAME=Echo

# MongoDB (Core)
MONGODB_URI=mongodb://localhost:27017
DB_NAME=Echo

# PASETO v2 (Local) & Sessions
PASETO_SECRET=your-secret-here
SESSION_SECRET=your-session-secret
```

### 3. Run the Server
The server now includes a built-in `.env` loader. Just run:
```bash
go run ./cmd/server
```
The server will soar on **http://localhost:8001**.

---

## 🧪 "Go's Love": Unit Testing

We use the **Repository Pattern** to isolate database logic. This allows us to test our business logic without needing a real database.

### How to Run Tests
To run all tests and see the "Green Checks":
```bash
go test -v ./features/...
```

### Learning Unit Testing
If you want to learn how to write new tests:
1.  Open `features/user/user_test.go`.
2.  Look for the **"Arrange, Act, Assert"** pattern.
3.  Study how **MockUserStore** mimics the database to return specific results.

---

## 📖 Live API Documentation (Swagger)

Stop guessing what the API parameters are! Use the built-in interactive documentation.

### 1. Access the UI
Once the server is running, visit:
**[http://localhost:8001/swagger/index.html](http://localhost:8001/swagger/index.html)**

### 2. Updating Documentation
We use **swaggo/swag**. If you add a new endpoint or change a parameter:
1.  Update the `@Summary` and `@Param` comments in your handler.
2.  Run the generator command:
    ```bash
    swag init -g cmd/server/main.go
    ```
3.  Refresh your browser. Your docs are now synced!

---

## 📁 Architecture Overview

- **`/cmd/server`**: Entry point (`main.go`). Handles configuration and server startup.
- **`/core`**: Shared utilities, database connections, and middleware.
- **`/features`**: Independent modules (Auth, User, Chat, Room).
- **`/docs`**: Automatically generated Swagger specification.

---

## 🛡️ Security Best Practices
- **Passwords**: Always hashed with **Argon2id** (via `utils.HashPassword`).
- **Tokens**: **PASETO v2 Local** (`v2.local.*`) — ChaCha20-Poly1305 symmetric encryption. No algorithm negotiation means zero JWT `alg:none` attack surface.
- **Refreshing**: Uses a **30-day secure rotation** logic. Old sessions are atomically invalidated.
- **Validation**: Every OTP check is **Atomic** to prevent double-redemption attacks.

---

### **Maintainer's Note**
This codebase is designed to be readable. If you see a variable like `isVerified` or `authRequest`, it is named that way so that **human beings** can understand the code at first glance. 
