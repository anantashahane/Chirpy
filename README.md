# Chirpy API

This document describes the Chirpy API endpoints.

---

## 📊 API Endpoints

- GET `/admin/metrics/`
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
