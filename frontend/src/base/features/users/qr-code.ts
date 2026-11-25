import { encode } from "uqr"

/** The modules of a QR code (true = dark), with a quiet zone around them. */
export const qrModules = (value: string) => encode(value, { ecc: "M", border: 2 }).data

/** Saves the QR code as PNG, which messengers accept as an image. */
export function downloadQrCode(value: string, fileName: string) {
  const data = qrModules(value)
  const scale = 10
  const canvas = document.createElement("canvas")
  canvas.width = canvas.height = data.length * scale
  const ctx = canvas.getContext("2d")
  if (!ctx) return
  ctx.fillStyle = "#fff"
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  ctx.fillStyle = "#000"
  data.forEach((row, y) => row.forEach((dark, x) => dark && ctx.fillRect(x * scale, y * scale, scale, scale)))
  const link = document.createElement("a")
  link.href = canvas.toDataURL("image/png")
  link.download = fileName
  link.click()
}
