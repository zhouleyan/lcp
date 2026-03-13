import { pkiApi } from "./client"
import { apiRequest } from "../client"
import type { Certificate, CertificateList, ListParams } from "../types"
import { getAccessToken } from "@/lib/auth"

export async function listCertificates(params?: ListParams): Promise<CertificateList> {
  return apiRequest(pkiApi.get("certificates", { searchParams: params as Record<string, string> }).json())
}

export async function getCertificate(id: string): Promise<Certificate> {
  return apiRequest(pkiApi.get(`certificates/${id}`).json())
}

export async function createCertificate(data: Pick<Certificate, "metadata" | "spec">): Promise<Certificate> {
  return apiRequest(pkiApi.post("certificates", { json: data }).json())
}

export async function deleteCertificate(id: string): Promise<void> {
  await apiRequest(pkiApi.delete(`certificates/${id}`).json())
}

export async function deleteCertificates(ids: string[]): Promise<void> {
  await apiRequest(pkiApi.delete("certificates", { json: { ids } }).json())
}

/**
 * Download a certificate file via blob. Triggers browser file download.
 *
 * Uses native fetch instead of ky because ky doesn't support blob downloads
 * natively and we need direct access to the response for createObjectURL.
 * Note: this bypasses the centralized 401 token refresh in api/client.ts.
 *
 * @param id - Certificate ID
 * @param file - File type: "cert.pem", "cert.crt", "key.pem", "key.key", "chain.pem", "chain.crt"
 */
export async function exportCertificateFile(id: string, file: string): Promise<void> {
  const token = getAccessToken()
  const response = await fetch(`/api/pki/v1/certificates/${id}/export?file=${encodeURIComponent(file)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!response.ok) {
    throw new Error(`Export failed: ${response.statusText}`)
  }
  const blob = await response.blob()
  // Extract filename from Content-Disposition header, fallback to generic name
  const disposition = response.headers.get("Content-Disposition") ?? ""
  const match = disposition.match(/filename="(.+?)"/)
  const downloadName = match?.[1] ?? file

  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = downloadName
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
