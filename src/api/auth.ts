export const adminAuthApi = {
  login() {
    return "admin"
  },

  logout() {
    return ""
  },

  currentUser() {
    return requestJson('/admin/auth/me')
      .then((payload) => unwrapApiResponse(payload))
  }
}
