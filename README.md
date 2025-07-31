# Chirpy API

This document describes the Chirpy API endpoints.

---

## API Endpoints
### Admin
- 📊 GET `/admin/metrics/`
  - Returns an HTML page showing the number of times the Chirpy server’s file handler has been accessed. This is primarily intended for administrative monitoring.
  - 🔒 Authorization: None required. Make sure this route is protected by other means (e.g. middleware) if needed.
  - 🧾 Request:
    - **Method:** `GET`
    - **URL:** `/admin/metrics/`
    - **Headers:** None required
    - **Body:** None
  - ✅ Response
    - **Status Code:** `200 OK`
    - **Headers:**
      - `Content-Type: text/html`
    - **Body:**
      - Returns an HTML document containing the current count of file server visits.
- ♻️ POST `/admin/reset/`
  Resets the file server hit counter and deletes all user records. **Only available in the development environment.**
  - 🔒 **Authorization:** None in code. This route is environment-gated: only works when `.env` includes `PLATFORM=dev`.
  - 🧾 **Request:**
      - **Method:** `POST`
      - **URL:** `/admin/reset/`
      - **Headers:** None required
      - **Body:** None
  - ✅ **Response:**
    - **Status Code:**
      - `200 OK` if reset successful and in dev mode
      - `403 Forbidden` if called outside of dev mode
    - **Headers:** None
    - **Body:** Empty HTML document

### Application
- 🧭 GET `/app/`
  - Resets the server’s visit counter and deletes all user records. **Only available in the dev environment.**
  - No explicit authentication in code. Is protected from production access, only works when `.env` has key value pair of `PLATFORM="dev"`.
  - 🧾 Request:
    - **Method:** `POST`
    - **URL:** `/admin/reset/`
    - **Headers:** None required
    - **Body:** None
  - ✅ Response
    - **Status Code:** `200 OK/403 Forbidden`
    - **Headers:**
      - None.
    - **Body:**
      - Returns an empty html document with appropriate status code.

### API
#### Users
- 👤 POST `/api/users`
  Creates a new Chirpy user account.
  - 🔒 **Authorization:** None required
  - 🧾 **Request:**
  - **Method:** `POST`
  - **URL:** `/api/users`
  - **Headers:**
    - `Content-Type: application/json`
  - **Body:**
    ```json
    {
      "email": "user@example.com",
      "password": "supersecret"
    }
    ```
  - ✅ **Response:**
    - **Status Code:** `201 Created`
  - **Headers:**
    - `Content-Type: application/json`
  - **Body:**
    ```json
    {
      "id": "uuid-string",
      "created_at": "timestamp",
      "updated_at": "timestamp",
      "email": "user@example.com",
      "is_red": false
    }
    ```
  - ❌ **Error Responses:**
    - `420`: Malformed JSON.
    - `401`: Password hashing failed.
    - `406`: Email already exists.
    - `404`: Failed to encode response JSON
- 🔐 POST `/api/login`
  Authenticates a user and returns an access token and refresh token.
  - 🔒 **Authorization:** None required
  - 🧾 **Request:**
    - **Method:** `POST`
    - **URL:** `/api/login`
  - **Headers:**
    - `Content-Type: application/json`
  - **Body:**
      ```json
      {
        "email": "user@example.com",
        "password": "supersecret"
      }
      ```
  - ✅ **Response:**
    - **Status Code:** `200 OK`
    - **Headers:**
      - `Content-Type: application/json`
    - **Body:**
      ```json
      {
        "id": "uuid-string",
        "created_at": "timestamp",
        "updated_at": "timestamp",
        "email": "user@example.com",
        "token": "access.jwt.token",
        "refresh_token": "refresh.token.value",
        "is_red": false
      }
      ```
  - ❌ **Error Responses:**
    - `420`: Malformed JSON.
    - `401`:
      - User not found (`"No such user, user@example.com"`)
      - Incorrect password
    - `403`: JWT generation failure
    - `503`:
      - Refresh token generation error
      - Response encoding failure.
- 🔄 PUT `/api/users`
  Updates the logged in user's email and password.
  - 🔒 **Authorization:** Requires Bearer token in the `Authorization` header, which is a `JWT` token returned for `POST /api/login` under token, **not** refresh token.
  - 🧾 **Request:**
    - **Method:** `PUT`
    - **URL:** `/api/users`
    - **Headers:**
      - `Authorization: Bearer <access_token>`
      - `Content-Type: application/json`
    - **Body:**
      ```json
      {
        "email": "user@example.com",
        "password": "newpassword123"
      }
      ```
  - ✅ **Response:**
    - **Status Code:** `200 OK`
    - **Headers:**
      - `Content-Type: application/json`
    - **Body:**
      ```json
      {
        "id": "uuid-string",
        "created_at": "timestamp.string",
        "updated_at": "timestamp.string",
        "email": "string",
        "token": "string",
        "refresh_token": "string",
        "is_red": false
      }
      ```
    - ❌ **Error Responses:**
      - `401`:
        - Missing or invalid token.
        - Unauthorized or missing user in database.
        - Malformed request body or JSON decode issues.
        - Password hashing or DB update error.
- 🔁 POST `/api/refresh`
  Issues a new access token using a valid refresh token.
  - 🔒 **Authorization:** Requires a valid access token in the `Authorization` header.
  - 🧾 **Request:**
    - **Method:** `POST`
    - **URL:** `/api/refresh`
    - **Headers:**
      - `Authorization: Bearer <access_token>`
      - `Content-Type: application/json`
    - **Body:** None
  - ✅ **Response:**
    - **Status Code:** `200 OK`
    - **Headers:**
      - `Content-Type: application/json`
    - **Body:**
      ```json
      {
        "token": "new.access.jwt.token"
      }
      ```
  - ❌ **Error Responses:**
    - `401`: Refresh token not found/expired/revoker
    - `404`: Bearer token parsing error.
    - `503`:
      - Token generation failure
      - JSON encoding failure
- 🔒 POST `/api/revoke`
  Revokes a valid refresh token so it can no longer be used. Generate new token by loggin back in.
  - 🔒 **Authorization:** Requires the refresh token in the `Authorization` header.
  - 🧾 **Request:**
    - **Method:** `POST`
    - **URL:** `/api/revoke`
    - **Headers:**
      - `Authorization: Bearer <refresh_token>`
      - `Content-Type: application/json`
    - **Body:** None
  - ✅ **Response:**
    - **Status Code:** `204 No Content`
    - **Body:** Empty
  - ❌ **Error Responses:**
    - `404`:
      - Bearer token parsing failed or not found.
      - Revocation failed due to user id not found.

#### Chirps
- 🐦 POST `/api/chirps`
  Creates a new chirp associated with the authenticated user.
  - 🔒 **Authorization:** Requires a valid JWT access token in the `Authorization` header.
  - 🧾 **Request:**
    - **Method:** `POST`
    - **URL:** `/api/chirps`
    - **Headers:**
      - `Authorization: Bearer <access_token>`
      - `Content-Type: application/json`
      - **Body:**
        ```json
        {
          "body": "your chirp text here"
        }
        ```
        - `body` must be within the character length limit of 140.
  - ✅ **Response:**
    - **Status Code:** `201 Created`
    - **Headers:**
      - `Content-Type: application/json`
    - **Body:**
      ```json
      {
        "id": "uuid",
        "created_at": "timestamp",
        "updated_at": "timestamp",
        "body": "your chirp text here",
        "user_id": "uuid"
      }
      ```
  - ❌ **Error Responses:**
    - `406`:
      - Invalid JSON
      - Chirp too long
      - JWT missing or unreadable
      - JSON encoding error
    - `401`:
      - Invalid JWT
    - `422`:
      - Database error (e.g. user ID not found).
- 📥 GET `/api/chirps/`
  Fetches all chirps from the database. Supports optional sorting and filtering.
  - 🔓 **Authorization:** Not required
  - 🧾 **Request:**
    - **Method:** `GET`
    - **URL:** `/api/chirps/?sort=desc&author_id=<uuid>`
      - **Query Parameters (optional):**
        - `sort`: If set to `desc`, returns chirps in reverse chronological order. Default is ascending.
        - `author_id`: If provided, filters chirps by the given author's user ID.
  - ✅ **Response:**
    - **Status Code:** `200 OK`
    - **Headers:**
      - `Content-Type: application/json`
      - **Body:**
        ```json
        [
          {
            "id": "uuid",
            "created_at": "timestamp",
            "updated_at": "timestamp",
            "body": "chirp text",
            "user_id": "uuid"
          },
          ...
        ]
        ```
  - ❌ **Error Responses:**
    - `500`:
      - Failure to access database
      - JSON encoding error
- 📄 GET `/api/chirps/{chirpID}`
  Fetches a specific chirp by its unique ID.
  - 🔓 **Authorization:** Not required
  - 🧾 **Request:**
    - **Method:** `GET`
    - **URL:** `/api/chirps/{chirpID}`
    - **Path Parameters:**
      - `chirpID` (UUID): The ID of the chirp to retrieve
    - ✅ **Response:**
      - **Status Code:** `200 OK`
    - **Headers:**
      - `Content-Type: application/json`
    - **Body:**
      ```json
      {
        "id": "uuid",
        "created_at": "timestamp",
        "updated_at": "timestamp",
        "body": "chirp text",
        "user_id": "uuid"
      }
      ```
  - ❌ **Error Responses:**
    - `404 Not Found`: If the chirp with given ID is invalid or doesn't exist.
- 🗑️ DELETE `/api/chirps/{chirpID}`
  Deletes a specific chirp if the requesting user is the original author.
  - 🔐 **Authorization:** Required (Bearer token)
  - 🧾 **Request:**
    - **Method:** `DELETE`
    - **URL:** `/api/chirps/{chirpID}`
    - **Headers:**
      - `Authorization: Bearer <token>`
    - **Path Parameters:**
      - `chirpID` (UUID): ID of the chirp to delete
  - ✅ **Response:**
    - **Status Code:** `204 No Content`
  - **Body:** _Empty_
  - ❌ **Error Responses:**
    - `401 Unauthorized`: If token is missing or invalid
    - `403 Forbidden`: If the user is not the author of the chirp
    - `404 Not Found`: If the chirp does not exist

#### Webhooks
- 🔔 POST `/api/polka/webhooks`
  Handles webhook notifications from Polka to upgrade a user to "Chirpy Red".
  - 🔐 **Authorization:** Required (Polka webhook key via `Authorization` header)
  - 🧾 **Request:**
    - **Method:** `POST`
    - **URL:** `/api/polka/webhooks`
    - **Headers:**
      - `Authorization: <polka-webhook-secret>`
      - **Body (JSON):**
        ```json
        {
          "event": "user.upgraded",
          "data": {
            "user_id": "uuid-string"
          }
        }
        ```
  - ✅ **Response:**
    - **Status Code:** `204 No Content`
    - **Body:** _Empty_
  - ❌ **Error Responses:**
    - `401 Unauthorized`: Missing or incorrect Polka secret
    - `404 Not Found`: Invalid user ID, user not found, or decoding failure.
