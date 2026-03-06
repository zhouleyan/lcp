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
      <Toaster position="top-center" />
    </BrowserRouter>
  )
}
