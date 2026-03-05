import ky from "ky"

export const api = ky.create({
  prefixUrl: "/api/v1",
  hooks: {
    beforeError: [
      async (error) => {
        const { response } = error
        if (response && response.status === 401) {
          window.location.href = "/login"
        }
        return error
      },
    ],
  },
})
