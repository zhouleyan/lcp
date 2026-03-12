/**
 * Compute the usable IP range for a CIDR block.
 * For prefix < 31: first usable = network+1, last usable = broadcast-1
 * For /31: both IPs usable (RFC 3021)
 * For /32: single host IP
 * Returns null if CIDR is empty or invalid.
 */
export function cidrUsableRange(cidr: string): string | null {
  if (!cidr) return null
  const m = cidr.match(/^(\d+)\.(\d+)\.(\d+)\.(\d+)\/(\d+)$/)
  if (!m) return null

  const ip = ((+m[1] << 24) | (+m[2] << 16) | (+m[3] << 8) | +m[4]) >>> 0
  const prefix = +m[5]

  if (prefix > 32 || prefix < 0) return null
  if (prefix === 32) return numToIP(ip)

  const mask = prefix === 0 ? 0 : (~0 << (32 - prefix)) >>> 0
  const network = (ip & mask) >>> 0
  const broadcast = (network | ~mask) >>> 0

  if (prefix === 31) return `${numToIP(network)} - ${numToIP(broadcast)}`

  return `${numToIP(network + 1)} - ${numToIP(broadcast - 1)}`
}

function numToIP(n: number): string {
  return `${(n >>> 24) & 0xff}.${(n >>> 16) & 0xff}.${(n >>> 8) & 0xff}.${n & 0xff}`
}
