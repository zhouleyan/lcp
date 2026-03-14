/**
 * Check whether an IPv4 address falls within a CIDR range.
 */
export function isIPInCIDR(ip: string, cidr: string): boolean {
  const parts = ip.split(".").map(Number)
  if (parts.length !== 4 || parts.some((p) => isNaN(p) || p < 0 || p > 255)) return false
  const [net, bits] = cidr.split("/")
  const prefix = Number(bits)
  if (isNaN(prefix) || prefix < 0 || prefix > 32) return false
  const netParts = net.split(".").map(Number)
  if (netParts.length !== 4) return false
  const ipNum = ((parts[0] << 24) | (parts[1] << 16) | (parts[2] << 8) | parts[3]) >>> 0
  const netNum = ((netParts[0] << 24) | (netParts[1] << 16) | (netParts[2] << 8) | netParts[3]) >>> 0
  const mask = prefix === 0 ? 0 : (~0 << (32 - prefix)) >>> 0
  return (ipNum & mask) === (netNum & mask)
}
