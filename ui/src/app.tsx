import { BrowserRouter, useRoutes } from "react-router"
import { routes } from "./routes"
import { Toaster } from "@/components/ui/sonner"

function AppRoutes() {
  return useRoutes(routes)
}

export default function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
      <Toaster position="top-center" toastOptions={{ style: { zIndex: 9999 } }} offset="4rem" />
    </BrowserRouter>
  )
}
